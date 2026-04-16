package server

import (
	"bytes"
	"config-generator/config"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func newTextMessage(role, content string) openAIChatMessage {
	return openAIChatMessage{
		Role: role,
		Content: []openAIMessagePart{
			{Type: "text", Text: content},
		},
	}
}

func normalizeBaseURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return defaultAISettings().BaseURL
	}
	return strings.TrimRight(trimmed, "/")
}

// testAISettingsConnection validates endpoint/model/key by sending a minimal JSON-mode request.
func (s *Server) testAISettingsConnection(override *aiSettingsUpdateRequest) (map[string]interface{}, error) {
	var (
		apiKey   string
		settings aiSettingsFile
		err      error
	)

	if override != nil {
		settings, err = s.loadAISettingsFile()
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(override.BaseURL) != "" {
			settings.BaseURL = strings.TrimSpace(override.BaseURL)
		}
		if strings.TrimSpace(override.Model) != "" {
			settings.Model = strings.TrimSpace(override.Model)
		}

		switch {
		case override.ClearAPIKey:
			apiKey = ""
		case strings.TrimSpace(override.APIKey) != "":
			apiKey = strings.TrimSpace(override.APIKey)
		default:
			apiKey, settings, err = s.getAIAPIKey()
			if err != nil && !errors.Is(err, errAISettingsMissing) {
				return nil, err
			}
		}
	} else {
		apiKey, settings, err = s.getAIAPIKey()
		if err != nil {
			return nil, err
		}
	}

	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("请先填写真实的 API Key，或先保存已有配置")
	}
	if strings.TrimSpace(settings.BaseURL) == "" || strings.TrimSpace(settings.Model) == "" {
		return nil, errors.New("请完整填写 Base URL 和 Model")
	}
	fmt.Printf("[AI] test connection baseUrl=%s model=%s\n", settings.BaseURL, settings.Model)

	reqBody := openAIChatRequest{
		Model: settings.Model,
		Messages: []openAIChatMessage{
			newTextMessage("system", "Reply with a short JSON object."),
			newTextMessage("user", `{"ping":"ok"}`),
		},
		ResponseFormat: map[string]string{"type": "json_object"},
		Temperature:    0,
		MaxTokens:      64,
	}

	_, err = s.aiClient.ChatCompletion(normalizeBaseURL(settings.BaseURL), apiKey, reqBody)
	if err != nil {
		fmt.Printf("[AI] test connection failed: %v\n", err)
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": "AI 接口连接测试成功",
	}, nil
}

func doOpenAIChatCompletion(baseURL, apiKey string, reqBody openAIChatRequest) (string, error) {
	fmt.Printf("[AI] request baseUrl=%s model=%s messages=%d maxTokens=%d\n",
		redactBaseURL(baseURL), reqBody.Model, len(reqBody.Messages), reqBody.MaxTokens)
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, normalizeBaseURL(baseURL)+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Printf("[AI] response http=%d bytes=%d\n", resp.StatusCode, len(body))

	var parsed openAIChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		if resp.StatusCode >= http.StatusBadRequest {
			return "", fmt.Errorf("AI 接口返回错误: %s", strings.TrimSpace(string(body)))
		}
		return "", err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", errors.New(parsed.Error.Message)
		}
		return "", fmt.Errorf("AI 接口调用失败: HTTP %d", resp.StatusCode)
	}

	if len(parsed.Choices) == 0 {
		return "", errors.New("AI 接口未返回有效结果")
	}
	if reason := strings.TrimSpace(parsed.Choices[0].FinishReason); reason == "length" || reason == "max_tokens" {
		fmt.Printf("[AI] response truncated finish_reason=%s\n", reason)
		return "", errors.New("AI 输出被截断，请减少一次生成内容，或提高模型输出上限后重试")
	}

	return parsed.Choices[0].Message.Content, nil
}

// parseAIModelResponse attempts multiple JSON recovery strategies for model outputs.
func parseAIModelResponse(raw string) (*aiModelResponse, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, errors.New("AI 返回内容为空")
	}

	var result aiModelResponse
	candidates := buildAIJSONCandidates(clean)
	for _, candidate := range candidates {
		if err := json.Unmarshal([]byte(candidate), &result); err == nil {
			return &result, nil
		}
	}

	snippet := clean
	if len([]rune(snippet)) > 240 {
		snippet = string([]rune(snippet)[:240]) + "..."
	}
	return &aiModelResponse{
		AssistantMessage: clean,
		DraftFiles:       []aiDraftFile{},
		PlannedActions:   []aiPlannedAction{},
		Warnings: []string{
			fmt.Sprintf("当前模型未按 JSON 协议返回结构化结果，已按纯文本展示；未生成可保存草稿与配置补丁。原始片段: %s", snippet),
		},
		FollowUpQuestions: []string{},
		ConfigPatch: aiConfigPatch{
			ServiceTop:   map[string]config.ServiceTopo{},
			ServerConfig: map[string]config.ServiceConfig{},
		},
	}, nil
}

func buildAIJSONCandidates(raw string) []string {
	candidates := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	appendCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}

	appendCandidate(raw)
	appendCandidate(stripMarkdownCodeFence(raw))
	appendCandidate(extractFirstJSONObject(raw))
	appendCandidate(extractFirstJSONObject(stripMarkdownCodeFence(raw)))
	appendCandidate(repairJSONLikeContent(raw))
	appendCandidate(repairJSONLikeContent(stripMarkdownCodeFence(raw)))
	appendCandidate(repairJSONLikeContent(extractFirstJSONObject(raw)))
	appendCandidate(repairJSONLikeContent(extractFirstJSONObject(stripMarkdownCodeFence(raw))))
	return candidates
}

func stripMarkdownCodeFence(raw string) string {
	clean := strings.TrimSpace(raw)
	if !strings.HasPrefix(clean, "```") {
		return clean
	}

	lines := strings.Split(clean, "\n")
	if len(lines) < 2 {
		return clean
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		return clean
	}

	endIndex := -1
	for idx := len(lines) - 1; idx >= 1; idx-- {
		if strings.TrimSpace(lines[idx]) == "```" {
			endIndex = idx
			break
		}
	}
	if endIndex == -1 {
		return clean
	}
	return strings.TrimSpace(strings.Join(lines[1:endIndex], "\n"))
}

func extractFirstJSONObject(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}

	start := -1
	depth := 0
	inString := false
	escaped := false
	for idx, r := range clean {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if r == '{' {
			if depth == 0 {
				start = idx
			}
			depth++
			continue
		}
		if r == '}' {
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				return strings.TrimSpace(clean[start : idx+1])
			}
		}
	}
	return ""
}

func repairJSONLikeContent(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(clean) + 64)

	inString := false
	escaped := false

	for idx := 0; idx < len(clean); idx++ {
		ch := clean[idx]
		if !inString {
			builder.WriteByte(ch)
			if ch == '"' {
				inString = true
			}
			continue
		}

		if escaped {
			builder.WriteByte(ch)
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			if isValidJSONEscape(clean, idx) {
				builder.WriteByte(ch)
				escaped = true
			} else {
				builder.WriteString(`\\`)
			}
		case '"':
			if isLikelyStringTerminator(clean, idx) {
				builder.WriteByte(ch)
				inString = false
			} else {
				builder.WriteString(`\"`)
			}
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			builder.WriteByte(ch)
		}
	}

	return stripTrailingCommasOutsideStrings(builder.String())
}

func isValidJSONEscape(raw string, idx int) bool {
	if idx+1 >= len(raw) {
		return false
	}
	switch raw[idx+1] {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		return true
	case 'u':
		if idx+5 >= len(raw) {
			return false
		}
		for pos := idx + 2; pos <= idx+5; pos++ {
			if !isHex(raw[pos]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func isHex(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isLikelyStringTerminator(raw string, idx int) bool {
	for pos := idx + 1; pos < len(raw); pos++ {
		switch raw[pos] {
		case ' ', '\n', '\r', '\t':
			continue
		case ':', ',', '}', ']':
			return true
		default:
			return false
		}
	}
	return true
}

func stripTrailingCommasOutsideStrings(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}

	var builder strings.Builder
	builder.Grow(len(raw))

	inString := false
	escaped := false

	for idx := 0; idx < len(raw); idx++ {
		ch := raw[idx]
		if inString {
			builder.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			builder.WriteByte(ch)
			continue
		}
		if ch == ',' {
			next := idx + 1
			for next < len(raw) && (raw[next] == ' ' || raw[next] == '\n' || raw[next] == '\r' || raw[next] == '\t') {
				next++
			}
			if next < len(raw) && (raw[next] == '}' || raw[next] == ']') {
				continue
			}
		}
		builder.WriteByte(ch)
	}

	return builder.String()
}

func resolveSelectedDrafts(session *aiSession, selectedDraftPaths []string) []aiDraftFile {
	selectedDrafts := make([]aiDraftFile, 0)
	if len(selectedDraftPaths) == 0 {
		return append(selectedDrafts, session.DraftFiles...)
	}

	selected := make(map[string]struct{}, len(selectedDraftPaths))
	for _, path := range selectedDraftPaths {
		selected[path] = struct{}{}
	}
	for _, draft := range session.DraftFiles {
		if _, ok := selected[draft.Path]; ok {
			selectedDrafts = append(selectedDrafts, draft)
		}
	}
	return selectedDrafts
}

func buildPromptPreviewSummary(trace aiPromptTrace, selectedDrafts []aiDraftFile, attachments []aiPromptPreviewAttachment) string {
	parts := make([]string, 0, 4)
	if len(trace.SkillRefs) > 0 {
		parts = append(parts, fmt.Sprintf("%d 条规则", len(trace.SkillRefs)))
	}
	if len(trace.ExampleRefs) > 0 {
		parts = append(parts, fmt.Sprintf("%d 个示例", len(trace.ExampleRefs)))
	}
	if len(selectedDrafts) > 0 {
		parts = append(parts, fmt.Sprintf("%d 个草稿", len(selectedDrafts)))
	}
	if len(attachments) > 0 {
		parts = append(parts, fmt.Sprintf("%d 个附件", len(attachments)))
	}
	if len(parts) == 0 {
		return "当前没有命中额外上下文"
	}
	return "本次发送将带上 " + strings.Join(parts, "、")
}

func resolveAIModel(requestModel, sessionModel, settingsModel string) string {
	if model := strings.TrimSpace(requestModel); model != "" {
		return model
	}
	if model := strings.TrimSpace(sessionModel); model != "" {
		return model
	}
	if model := strings.TrimSpace(settingsModel); model != "" {
		return model
	}
	return strings.TrimSpace(defaultAISettings().Model)
}

// buildAIPromptPreview returns a trace of which skills/examples/attachments will be sent in the next request.
func (s *Server) buildAIPromptPreview(session *aiSession, req aiSessionMessageRequest, attachmentIDs []string) (aiPromptPreviewResponse, error) {
	selectedDrafts := resolveSelectedDrafts(session, req.SelectedDraftPaths)
	bundle, err := s.buildAIPromptBundle(session, req, selectedDrafts, attachmentIDs)
	if err != nil {
		return aiPromptPreviewResponse{}, err
	}

	attachments := make([]aiPromptPreviewAttachment, 0, len(attachmentIDs))
	for _, attachmentID := range attachmentIDs {
		attachment, err := s.findAttachment(session, attachmentID)
		if err != nil {
			continue
		}
		attachments = append(attachments, aiPromptPreviewAttachment{
			ID:   attachment.ID,
			Name: attachment.Name,
			Kind: attachment.Kind,
		})
	}

	selectedDraftPaths := make([]string, 0, len(selectedDrafts))
	for _, draft := range selectedDrafts {
		selectedDraftPaths = append(selectedDraftPaths, draft.Path)
	}

	return aiPromptPreviewResponse{
		Summary:            buildPromptPreviewSummary(bundle.PromptTrace, selectedDrafts, attachments),
		PromptTrace:        bundle.PromptTrace,
		SelectedDraftPaths: selectedDraftPaths,
		SelectedSkillIDs:   append([]string(nil), req.SelectedSkillIDs...),
		Attachments:        attachments,
		HasSessionRules:    strings.TrimSpace(req.SessionRules) != "",
	}, nil
}

// buildAIChatRequest composes one OpenAI-compatible request with system rules, history, drafts and attachments.
func (s *Server) buildAIChatRequest(session *aiSession, req aiSessionMessageRequest, attachmentIDs []string) (openAIChatRequest, error) {
	selectedDrafts := resolveSelectedDrafts(session, req.SelectedDraftPaths)

	bundle, err := s.buildAIPromptBundle(session, req, selectedDrafts, attachmentIDs)
	if err != nil {
		return openAIChatRequest{}, fmt.Errorf("build prompt bundle failed: %w", err)
	}

	messages := []openAIChatMessage{newTextMessage("system", bundle.SystemPrompt)}

	for _, message := range session.Messages {
		builder := strings.TrimSpace(message.Content)
		if len(message.AttachmentIDs) > 0 {
			builder += "\n\n关联附件：" + strings.Join(s.buildMessageAttachmentPreview(session, message.AttachmentIDs), ", ")
		}
		if len(message.Warnings) > 0 {
			builder += "\n\nWarnings:\n- " + strings.Join(message.Warnings, "\n- ")
		}
		if len(message.DraftFiles) > 0 {
			paths := make([]string, 0, len(message.DraftFiles))
			for _, file := range message.DraftFiles {
				paths = append(paths, file.Path)
			}
			builder += "\n\n涉及草稿文件：" + strings.Join(paths, ", ")
		}
		if promptTraceSummary := formatPromptTraceSummary(message.PromptTrace); strings.TrimSpace(promptTraceSummary) != "" {
			builder += "\n\n历史规则来源：\n" + promptTraceSummary
		}
		messages = append(messages, newTextMessage(message.Role, builder))
	}

	var userParts []openAIMessagePart
	userText := strings.TrimSpace(req.Message)
	if userText == "" {
		userText = "请基于当前上下文继续。"
	}

	if len(selectedDrafts) > 0 {
		builder := strings.Builder{}
		builder.WriteString(userText)
		builder.WriteString("\n\n当前可编辑草稿：")
		for _, draft := range selectedDrafts {
			builder.WriteString("\n---\n路径: ")
			builder.WriteString(draft.Path)
			if draft.Reason != "" {
				builder.WriteString("\n原因: ")
				builder.WriteString(draft.Reason)
			}
			builder.WriteString("\n内容:\n")
			builder.WriteString(draft.Content)
		}
		userText = builder.String()
	}

	userParts = append(userParts, openAIMessagePart{Type: "text", Text: userText})

	for _, attachmentID := range attachmentIDs {
		attachment, err := s.findAttachment(session, attachmentID)
		if err != nil {
			continue
		}

		if attachment.Kind == "text" {
			textContent := attachment.PreviewText
			if textContent == "" {
				textContent = loadTextPreview(attachment.StoredPath, aiTextFileLimit)
			}
			userParts = append(userParts, openAIMessagePart{
				Type: "text",
				Text: fmt.Sprintf("附件 %s（文本内容）:\n%s", attachment.Name, textContent),
			})
			continue
		}

		imageBytes, err := os.ReadFile(attachment.StoredPath)
		if err != nil {
			return openAIChatRequest{}, err
		}
		encoded := base64.StdEncoding.EncodeToString(imageBytes)
		userParts = append(userParts,
			openAIMessagePart{
				Type: "text",
				Text: fmt.Sprintf("附件 %s 是截图/照片。请先识别其中可见文字，再按模板规范生成草稿；不清晰部分写入 warnings。", attachment.Name),
			},
			openAIMessagePart{
				Type: "image_url",
				ImageURL: &openAIImageURLHolder{
					URL: "data:" + attachment.MimeType + ";base64," + encoded,
				},
			},
		)
	}

	messages = append(messages, openAIChatMessage{
		Role:    "user",
		Content: userParts,
	})

	settings, err := s.loadAISettingsFile()
	if err != nil {
		return openAIChatRequest{}, fmt.Errorf("load AI settings failed: %w", err)
	}
	resolvedModel := resolveAIModel(req.Model, session.SelectedModel, settings.Model)
	if resolvedModel == "" {
		return openAIChatRequest{}, errors.New("妯″瀷涓嶈兘涓虹┖")
	}

	session.PromptTrace = bundle.PromptTrace
	session.SelectedModel = resolvedModel
	responseFormat := map[string]string{"type": "text"}
	if bundle.Structured {
		responseFormat = map[string]string{"type": "json_object"}
	}
	return openAIChatRequest{
		Model:          resolvedModel,
		Messages:       messages,
		ResponseFormat: responseFormat,
		Temperature:    0.2,
		MaxTokens:      8192,
	}, nil
}

func mergeDraftFiles(existing []aiDraftFile, updates []aiDraftFile) []aiDraftFile {
	merged := make(map[string]aiDraftFile, len(existing)+len(updates))
	for _, file := range existing {
		merged[file.Path] = file
	}
	for _, file := range updates {
		normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(file.Path)))
		file.Path = normalized
		if file.Source == "" {
			file.Source = "ai"
		}
		merged[file.Path] = file
	}
	paths := make([]string, 0, len(merged))
	for path := range merged {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := make([]aiDraftFile, 0, len(paths))
	for _, path := range paths {
		result = append(result, merged[path])
	}
	return result
}

// syncSavedDraftsToSession keeps session draft snapshots consistent with reviewed content saved from UI.
// This avoids "templates 已落盘但会话仍显示旧草稿" when frontend refreshes session data after save.
func syncSavedDraftsToSession(session *aiSession, savedPaths []string, incoming []aiDraftFile) {
	if session == nil || len(savedPaths) == 0 || len(incoming) == 0 {
		return
	}
	savedPathSet := make(map[string]struct{}, len(savedPaths))
	for _, path := range savedPaths {
		normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if normalized != "" {
			savedPathSet[normalized] = struct{}{}
		}
	}
	if len(savedPathSet) == 0 {
		return
	}

	filtered := make([]aiDraftFile, 0, len(incoming))
	for _, draft := range incoming {
		normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(draft.Path)))
		if normalized == "" {
			continue
		}
		if _, ok := savedPathSet[normalized]; !ok {
			continue
		}
		draft.Path = normalized
		if strings.TrimSpace(draft.Source) == "" {
			draft.Source = "ai"
		}
		filtered = append(filtered, draft)
	}
	if len(filtered) == 0 {
		return
	}

	session.DraftFiles = mergeDraftFiles(session.DraftFiles, filtered)
	for idx := range session.Messages {
		if len(session.Messages[idx].DraftFiles) == 0 {
			continue
		}
		session.Messages[idx].DraftFiles = mergeDraftFiles(session.Messages[idx].DraftFiles, filtered)
	}
}

// handleAISettings manages AI endpoint settings retrieval and updates.
func (s *Server) handleAISettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		settings, err := s.getAISettingsResponse()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(settings)

	case http.MethodPut:
		var req aiSettingsUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		settings, err := s.updateAISettings(req)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, errAIMasterKeyMissing) {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}
		json.NewEncoder(w).Encode(settings)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAISettingsTest performs connectivity verification for current or temporary settings.
func (s *Server) handleAISettingsTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req aiSettingsUpdateRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	result, err := s.testAISettingsConnection(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("[AI] test connection ok\n")
	json.NewEncoder(w).Encode(result)
}

// handleAIModels returns the provider-visible model list from the configured OpenAI-compatible endpoint.
func (s *Server) handleAIModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey, settings, err := s.getAIAPIKey()
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, errAIMasterKeyMissing) && !errors.Is(err, errAISettingsMissing) {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}

	fmt.Printf("[AI] list models baseUrl=%s\n", settings.BaseURL)
	models, err := listCompatibleAIModels(settings.BaseURL, apiKey)
	if err != nil {
		fmt.Printf("[AI] list models failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("[AI] list models ok count=%d\n", len(models))
	json.NewEncoder(w).Encode(models)
}

// handleAIRules manages the global rules document used in every AI prompt.
func (s *Server) handleAIRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		content, err := s.loadAIRules()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"content": content})

	case http.MethodPut:
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.saveAIRules(payload.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "模板规则已保存",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAIPromptCatalog returns skill/example catalogs used by the Skill management UI.
func (s *Server) handleAIPromptCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	skills, err := s.listAISkillCatalog()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	examples, err := s.buildExampleIndex()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(aiPromptCatalogResponse{
		Skills:   skills,
		Examples: examples,
	})
}

// handleAISkillFile updates one skill markdown file by absolute relative path.
func (s *Server) handleAISkillFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		path := strings.TrimSpace(r.URL.Query().Get("path"))
		if path == "" {
			http.Error(w, "Missing skill path", http.StatusBadRequest)
			return
		}
		item, err := s.loadAISkillCatalogItem(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Skill not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(item)
	case http.MethodPut:
		var req aiSkillFileUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		item, err := s.saveAISkillContent(req.Path, req.Content)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Skill not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(item)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAICustomSkillsCollection lists and creates custom skills under data/ai/skills.
func (s *Server) handleAICustomSkillsCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		skills, err := s.listCustomAISkills()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(skills)
	case http.MethodPost:
		var req aiCustomSkillUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := s.loadCustomSkill(req.ID); err == nil {
			http.Error(w, "同名自定义 skill 已存在，请改用更新操作", http.StatusConflict)
			return
		} else if err != nil && !os.IsNotExist(err) && !strings.Contains(err.Error(), "no such file") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		skill, err := s.saveCustomSkill(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(skill)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAICustomSkillsDetail fetches, updates, or deletes one custom skill by id.
func (s *Server) handleAICustomSkillsDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	skillID := strings.TrimPrefix(r.URL.Path, "/api/ai/skills/")
	skillID = strings.TrimSpace(strings.Trim(skillID, "/"))
	if skillID == "" {
		http.Error(w, "Invalid skill path", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		skill, err := s.loadCustomSkill(skillID)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Skill not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(skill)
	case http.MethodPut:
		var req aiCustomSkillUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := s.loadCustomSkill(skillID); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Skill not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.ID = skillID
		skill, err := s.saveCustomSkill(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(skill)
	case http.MethodDelete:
		if err := s.deleteCustomSkill(skillID); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Skill not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "自定义 skill 已删除",
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAITemplateSessionCollection handles create/list/delete operations at session collection level.
func (s *Server) handleAITemplateSessionCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		sessions, err := s.listAISessions()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(sessions)
	case http.MethodPost:
		session, err := s.createAISession()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(session)
	case http.MethodDelete:
		var payload struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		sessionID := strings.TrimSpace(r.URL.Query().Get("id"))
		if sessionID == "" {
			sessionID = strings.TrimSpace(payload.ID)
		}
		if sessionID == "" {
			http.Error(w, "session id 不能为空", http.StatusBadRequest)
			return
		}
		if err := s.deleteAISession(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(aiSessionDeleteResponse{DeletedSessionID: sessionID})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAITemplateSessionDetail routes session sub-resources such as message/upload/save/meta.
func (s *Server) handleAITemplateSessionDetail(w http.ResponseWriter, r *http.Request) {
	sessionPath := strings.TrimPrefix(r.URL.Path, "/api/ai/template/session/")
	sessionPath = strings.Trim(sessionPath, "/")
	if sessionPath == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	parts := strings.Split(sessionPath, "/")
	sessionID := parts[0]
	session, err := s.loadAISession(sessionID)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(session)
			return
		case http.MethodDelete:
			if err := s.deleteAISession(sessionID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(aiSessionDeleteResponse{DeletedSessionID: sessionID})
			return
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	switch parts[1] {
	case "message":
		s.handleAISessionMessage(w, r, session)
	case "meta":
		s.handleAISessionMeta(w, r, session)
	case "preview":
		s.handleAISessionPreview(w, r, session)
	case "drafts":
		s.handleAISessionDrafts(w, r, session)
	case "upload":
		s.handleAISessionUpload(w, r, session)
	case "save":
		s.handleAISessionSave(w, r, session)
	case "attachment":
		if len(parts) != 3 {
			http.Error(w, "Invalid attachment path", http.StatusBadRequest)
			return
		}
		s.handleAISessionAttachment(w, r, session, parts[2])
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleAISessionMeta updates editable metadata like session title.
func (s *Server) handleAISessionMeta(w http.ResponseWriter, r *http.Request, session *aiSession) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req aiSessionMetaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.updateAISessionTitle(session, req.Title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updated, err := s.loadAISession(session.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(updated)
}

// handleAISessionDrafts deletes selected drafts from session state and optionally from disk.
func (s *Server) handleAISessionDrafts(w http.ResponseWriter, r *http.Request, session *aiSession) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req aiSessionDeleteDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	deletedDrafts, deletedTemplates, err := s.deleteDraftFiles(session, req.Paths, req.RemoveFromDisk)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := s.loadAISession(session.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"deletedDrafts":    deletedDrafts,
		"deletedTemplates": deletedTemplates,
		"session":          updated,
	})
}

// handleAISessionPreview returns preflight prompt-trace details before an actual model call.
func (s *Server) handleAISessionPreview(w http.ResponseWriter, r *http.Request, session *aiSession) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	principal, ok := authPrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cfg, _, _, err := s.loadConfigForPrincipal(principal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.cfg = cfg

	var req aiSessionMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SelectedSkillIDs == nil {
		req.SelectedSkillIDs = append([]string(nil), session.SelectedSkillIDs...)
	}
	req.SelectedSkillIDs = normalizeSelectedSkillIDs(req.SelectedSkillIDs)
	fmt.Printf("[AI] preview session=%s messageLen=%d draftPaths=%d skills=%d\n",
		session.ID, len(strings.TrimSpace(req.Message)), len(req.SelectedDraftPaths), len(req.SelectedSkillIDs))

	attachmentIDs := make([]string, 0)
	for _, attachment := range session.Attachments {
		if attachment.Pending {
			attachmentIDs = append(attachmentIDs, attachment.ID)
		}
	}

	preview, err := s.buildAIPromptPreview(session, req, attachmentIDs)
	if err != nil {
		fmt.Printf("[AI] preview failed session=%s err=%v\n", session.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[AI] preview ok session=%s skills=%d examples=%d attachments=%d\n",
		session.ID, len(preview.PromptTrace.SkillRefs), len(preview.PromptTrace.ExampleRefs), len(preview.Attachments))
	json.NewEncoder(w).Encode(preview)
}

// handleAISessionMessage dispatches one conversation turn, optionally via SSE stream mode.
func (s *Server) handleAISessionMessage(w http.ResponseWriter, r *http.Request, session *aiSession) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	principal, ok := authPrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cfg, _, _, err := s.loadConfigForPrincipal(principal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.cfg = cfg

	var req aiSessionMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionLock := s.getAISessionLock(session.ID)
	sessionLock.Lock()
	defer sessionLock.Unlock()

	latestSession, err := s.loadAISession(session.ID)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("stream") == "1" || strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream") {
		s.handleAISessionMessageStream(w, r, latestSession, req)
		return
	}
	result, status, err := s.runAISessionMessage(latestSession, req, nil)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	json.NewEncoder(w).Encode(result.Session)
	return

	if req.SelectedSkillIDs == nil {
		req.SelectedSkillIDs = append([]string(nil), session.SelectedSkillIDs...)
	}
	req.SelectedSkillIDs = normalizeSelectedSkillIDs(req.SelectedSkillIDs)

	attachmentIDs := consumePendingAttachments(session)
	fmt.Printf("[AI] message session=%s messageLen=%d pendingAttachments=%d draftPaths=%d skills=%d\n",
		session.ID, len(strings.TrimSpace(req.Message)), len(attachmentIDs), len(req.SelectedDraftPaths), len(req.SelectedSkillIDs))
	chatReq, err := s.buildAIChatRequest(session, req, attachmentIDs)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errAIMasterKeyMissing) || errors.Is(err, errAISettingsMissing) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	apiKey, settings, err := s.getAIAPIKey()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawResponse, err := s.aiClient.ChatCompletion(normalizeBaseURL(settings.BaseURL), apiKey, chatReq)
	if err != nil {
		if len(attachmentIDs) > 0 {
			for _, attachmentID := range attachmentIDs {
				attachment, findErr := s.findAttachment(session, attachmentID)
				if findErr == nil {
					attachment.Pending = true
				}
			}
		}
		if len(attachmentIDs) > 0 {
			http.Error(w, "AI 调用失败，当前模型或接口可能不支持图像输入，或请求格式被拒绝: "+err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	modelResponse, err := parseAIModelResponse(rawResponse)
	if err != nil {
		fmt.Printf("[AI] response parse failed session=%s err=%v\n", session.ID, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if modelResponse.PlannedActions, err = validatePlannedActions(modelResponse.PlannedActions); err != nil {
		fmt.Printf("[AI] planned actions invalid session=%s err=%v\n", session.ID, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	for idx := range modelResponse.DraftFiles {
		normalizedPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(modelResponse.DraftFiles[idx].Path)))
		modelResponse.DraftFiles[idx].Path = normalizedPath
		if !strings.HasPrefix(normalizedPath, "templates/") {
			http.Error(w, "AI 返回了非法模板路径", http.StatusBadGateway)
			return
		}
		modelResponse.DraftFiles[idx].NeedsReview = true
		if modelResponse.DraftFiles[idx].Source == "" {
			modelResponse.DraftFiles[idx].Source = "ai"
		}
		for _, attachmentID := range attachmentIDs {
			attachment, findErr := s.findAttachment(session, attachmentID)
			if findErr == nil && attachment.Kind == "image" {
				modelResponse.DraftFiles[idx].Source = "image"
			}
		}
	}

	var configIssues []aiConfigIssue
	modelResponse.ConfigPatch, configIssues = s.enrichConfigPatchForDrafts(modelResponse.DraftFiles, modelResponse.ConfigPatch)
	for _, issue := range configIssues {
		if issue.Severity == "warning" || issue.Severity == "error" {
			modelResponse.Warnings = mergeStringLists(modelResponse.Warnings, []string{issue.Message})
		}
	}
	fmt.Printf("[AI] response session=%s drafts=%d actions=%d warnings=%d followUps=%d configIssues=%d\n",
		session.ID, len(modelResponse.DraftFiles), len(modelResponse.PlannedActions), len(modelResponse.Warnings),
		len(modelResponse.FollowUpQuestions), len(configIssues))

	messageID, err := generateID("msg_")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Messages = append(session.Messages, aiSessionMessage{
		ID:            messageID,
		Role:          "user",
		Content:       strings.TrimSpace(req.Message),
		CreatedAt:     nowRFC3339(),
		AttachmentIDs: attachmentIDs,
	})

	assistantMessageID, err := generateID("msg_")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Messages = append(session.Messages, aiSessionMessage{
		ID:                assistantMessageID,
		Role:              "assistant",
		Content:           strings.TrimSpace(modelResponse.AssistantMessage),
		CreatedAt:         nowRFC3339(),
		Warnings:          modelResponse.Warnings,
		FollowUpQuestions: modelResponse.FollowUpQuestions,
		DraftFiles:        modelResponse.DraftFiles,
		PlannedActions:    modelResponse.PlannedActions,
		ConfigPatch:       modelResponse.ConfigPatch,
		ConfigIssues:      configIssues,
		PromptTrace:       session.PromptTrace,
	})

	session.SessionRules = req.SessionRules
	session.SelectedSkillIDs = req.SelectedSkillIDs
	session.DraftFiles = mergeDraftFiles(session.DraftFiles, modelResponse.DraftFiles)
	session.PlannedActions = modelResponse.PlannedActions
	session.ConfigPatch = mergeAIConfigPatches(session.ConfigPatch, modelResponse.ConfigPatch)
	session.ConfigIssues = mergeAIConfigIssues(session.ConfigIssues, configIssues)

	if err := s.saveAISession(session); err != nil {
		fmt.Printf("[AI] save session failed session=%s err=%v\n", session.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.populateAttachmentURLs(session)
	json.NewEncoder(w).Encode(session)
}

func readUploadedPart(file multipart.File) ([]byte, error) {
	defer file.Close()
	return io.ReadAll(file)
}

func redactBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return trimmed
	}
	return parsed.Scheme + "://" + parsed.Host
}

func sortedServiceNames(services map[string]struct{}) []string {
	names := make([]string, 0, len(services))
	for serviceName := range services {
		names = append(names, serviceName)
	}
	sort.Strings(names)
	return names
}

func countConfigIssueSeverity(issues []aiConfigIssue) (errorsCount, warningsCount, infosCount int) {
	for _, issue := range issues {
		switch strings.ToLower(strings.TrimSpace(issue.Severity)) {
		case "error":
			errorsCount++
		case "warning":
			warningsCount++
		default:
			infosCount++
		}
	}
	return
}

func suggestActionForConfigIssue(issue aiConfigIssue) (string, string) {
	serviceName := strings.TrimSpace(issue.Service)
	if serviceName == "" {
		serviceName = "当前服务"
	}
	field := strings.TrimSpace(issue.Field)
	switch {
	case field == "serviceTop":
		return "补齐 serviceTop 节点拓扑", fmt.Sprintf("服务 %s 缺少 serviceTop。请在“配置管理 > 服务拓扑”补齐 nodes/description/id_auto_derive 后重试。", serviceName)
	case field == "serverConfig":
		return "确认 serverConfig 骨架", fmt.Sprintf("服务 %s 将自动创建 serverConfig 骨架。同步成功后再到“配置管理 > 服务配置”补齐 vars 值。", serviceName)
	case strings.HasPrefix(field, "vars."):
		varName := strings.TrimPrefix(field, "vars.")
		if strings.TrimSpace(varName) == "" {
			varName = "未命名变量"
		}
		return "补齐 vars 变量值", fmt.Sprintf("服务 %s 的变量 %s 需要确认取值。请到“配置管理 > 服务配置”补值后重试。", serviceName, varName)
	default:
		if strings.EqualFold(issue.Severity, "error") {
			return "处理阻塞项", issue.Message
		}
		return "确认配置建议", issue.Message
	}
}

func buildConfigNextActions(issues []aiConfigIssue) []aiConfigNextAction {
	actions := make([]aiConfigNextAction, 0, len(issues))
	for _, issue := range issues {
		action, detail := suggestActionForConfigIssue(issue)
		actions = append(actions, aiConfigNextAction{
			Severity: issue.Severity,
			Service:  issue.Service,
			Field:    issue.Field,
			Action:   action,
			Detail:   detail,
		})
	}
	return actions
}

// handleAISessionUpload stores one text/image attachment as pending context for the next message.
func (s *Server) handleAISessionUpload(w http.ResponseWriter, r *http.Request, session *aiSession) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(aiImageFileLimit + aiTextFileLimit); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file 字段不能为空", http.StatusBadRequest)
		return
	}

	data, err := readUploadedPart(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = http.DetectContentType(data)
	}

	attachment, err := s.saveUploadedAttachment(session, header.Filename, contentType, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("[AI] upload session=%s name=%s kind=%s size=%d\n", session.ID, attachment.Name, attachment.Kind, attachment.Size)

	if err := s.saveAISession(session); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(attachment)
}

// handleAISessionSave persists reviewed draft templates and optionally applies config patch suggestions.
func (s *Server) handleAISessionSave(w http.ResponseWriter, r *http.Request, session *aiSession) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	principal, ok := authPrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if principal.Role != roleAdmin {
		http.Error(w, "仅管理员可以保存模板到正式目录", http.StatusForbidden)
		return
	}
	activeCfg, activePath, activeTemplateID, err := s.loadConfigForPrincipal(principal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.cfg = activeCfg

	var req aiSessionSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Files) == 0 {
		http.Error(w, "files 不能为空", http.StatusBadRequest)
		return
	}
	fmt.Printf("[AI] save session=%s files=%d applyConfigPatch=%v\n", session.ID, len(req.Files), req.ApplyConfigPatch)

	savedFiles, createdDirs, err := s.saveDraftFiles(req.Files)
	if err != nil {
		fmt.Printf("[AI] save failed session=%s err=%v\n", session.ID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Save succeeds on disk first, then mirror current reviewed content back into session snapshots.
	syncSavedDraftsToSession(session, savedFiles, req.Files)
	fmt.Printf("[AI] save session=%s syncedDraftSnapshots=%d\n", session.ID, len(savedFiles))

	configApplied := false
	appliedServices := make([]string, 0)
	selectedServices := selectServicesFromDrafts(req.Files)
	selectedConfigIssues := selectConfigIssues(session.ConfigIssues, selectedServices)
	selectedPatch := selectConfigPatchServices(session.ConfigPatch, selectedServices)
	selectedServiceNames := sortedServiceNames(selectedServices)
	fmt.Printf("[AI] save context session=%s user=%s template=%s selectedServices=%v cachedIssues=%d\n",
		session.ID, principal.Username, activeTemplateID, selectedServiceNames, len(selectedConfigIssues))
	if req.ApplyConfigPatch {
		// Re-check config issues against current config on every save-with-sync to avoid stale blocking.
		livePatch, liveIssues := s.enrichConfigPatchForDrafts(req.Files, selectedPatch)
		selectedPatch = selectConfigPatchServices(livePatch, selectedServices)
		selectedConfigIssues = selectConfigIssues(liveIssues, selectedServices)
		replaceSessionPatchAndIssuesForServices(session, selectedServices, selectedPatch, selectedConfigIssues)
		fmt.Printf("[AI] save recheck session=%s services=%v liveIssues=%d patchTop=%d patchCfg=%d\n",
			session.ID, selectedServiceNames, len(selectedConfigIssues), len(selectedPatch.ServiceTop), len(selectedPatch.ServerConfig))
		for _, issue := range selectedConfigIssues {
			fmt.Printf("[AI] save recheck issue session=%s severity=%s service=%s field=%s message=%s\n",
				session.ID, issue.Severity, issue.Service, issue.Field, issue.Message)
		}
		if hasBlockingConfigIssues(selectedConfigIssues) {
			errorsCount, warningsCount, infosCount := countConfigIssueSeverity(selectedConfigIssues)
			fmt.Printf("[AI] save blocked session=%s user=%s template=%s services=%v issues=%d errors=%d warnings=%d infos=%d\n",
				session.ID, principal.Username, activeTemplateID, selectedServiceNames, len(selectedConfigIssues), errorsCount, warningsCount, infosCount)
			for _, issue := range selectedConfigIssues {
				fmt.Printf("[AI] save blocked issue session=%s severity=%s service=%s field=%s message=%s\n",
					session.ID, issue.Severity, issue.Service, issue.Field, issue.Message)
			}
			for idx := range session.DraftFiles {
				for _, savedPath := range savedFiles {
					if session.DraftFiles[idx].Path == savedPath {
						session.DraftFiles[idx].NeedsReview = false
					}
				}
			}
			if err := s.saveAISession(session); err != nil {
				fmt.Printf("[AI] save session failed session=%s err=%v\n", session.ID, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Printf("[AI] save session=%s done files=%d configApplied=false blockingIssues=%d\n",
				session.ID, len(savedFiles), len(selectedConfigIssues))
			json.NewEncoder(w).Encode(map[string]interface{}{
				"savedFiles":      savedFiles,
				"createdDirs":     createdDirs,
				"configApplied":   false,
				"appliedServices": appliedServices,
				"configIssues":    selectedConfigIssues,
				"nextActions":     buildConfigNextActions(selectedConfigIssues),
				"message":         "Template saved, but config sync is blocked. Please resolve severity=error issues and retry.",
			})
			return
		}

		appliedServices = applyConfigPatchToConfig(activeCfg, selectedPatch)
		if err := config.SaveConfig(activePath, activeCfg); err != nil {
			fmt.Printf("[AI] save config failed session=%s err=%v\n", session.ID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if principal.Role == roleAdmin {
			userCount, syncLog := s.syncRootConfigToAllUsers(activeCfg)
			fmt.Printf("[AI] save config root synced users=%d %s\n", userCount, formatSyncLog(syncLog))
		}
		fmt.Printf("[AI] save config synced user=%s template=%s services=%d\n", principal.Username, activeTemplateID, len(appliedServices))
		configApplied = true
		removeAppliedServicesFromSession(session, selectedServices)
	}

	for idx := range session.DraftFiles {
		for _, savedPath := range savedFiles {
			if session.DraftFiles[idx].Path == savedPath {
				session.DraftFiles[idx].NeedsReview = false
			}
		}
	}
	if err := s.saveAISession(session); err != nil {
		fmt.Printf("[AI] save session failed session=%s err=%v\n", session.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[AI] save session=%s done files=%d configApplied=%v appliedServices=%d\n",
		session.ID, len(savedFiles), configApplied, len(appliedServices))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"savedFiles":      savedFiles,
		"createdDirs":     createdDirs,
		"configApplied":   configApplied,
		"appliedServices": appliedServices,
		"configIssues":    selectedConfigIssues,
		"nextActions":     buildConfigNextActions(selectedConfigIssues),
	})
}

func (s *Server) handleAISessionAttachment(w http.ResponseWriter, r *http.Request, session *aiSession, attachmentID string) {
	switch r.Method {
	case http.MethodGet:
		attachment, err := s.findAttachment(session, attachmentID)
		if err != nil {
			http.Error(w, "Attachment not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, attachment.StoredPath)
	case http.MethodDelete:
		if err := s.deletePendingAttachment(session, attachmentID); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Attachment not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.saveAISession(session); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"deletedAttachmentId": attachmentID,
			"session":             session,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
