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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type openAIChatRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIChatMessage `json:"messages"`
	ResponseFormat map[string]string   `json:"response_format,omitempty"`
	Temperature    float64             `json:"temperature,omitempty"`
	MaxTokens      int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string              `json:"role"`
	Content []openAIMessagePart `json:"content,omitempty"`
}

type openAIMessagePart struct {
	Type     string                `json:"type"`
	Text     string                `json:"text,omitempty"`
	ImageURL *openAIImageURLHolder `json:"image_url,omitempty"`
}

type openAIImageURLHolder struct {
	URL string `json:"url"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

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

	_, err = doOpenAIChatCompletion(normalizeBaseURL(settings.BaseURL), apiKey, reqBody)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": "AI 接口连接测试成功",
	}, nil
}

func doOpenAIChatCompletion(baseURL, apiKey string, reqBody openAIChatRequest) (string, error) {
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

	return parsed.Choices[0].Message.Content, nil
}

func parseAIModelResponse(raw string) (*aiModelResponse, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, errors.New("AI 返回内容为空")
	}

	var result aiModelResponse
	if err := json.Unmarshal([]byte(clean), &result); err == nil {
		return &result, nil
	}

	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(clean[start:end+1]), &result); err == nil {
			return &result, nil
		}
	}

	return nil, errors.New("AI 返回内容不是合法 JSON")
}

func (s *Server) buildAIChatRequest(session *aiSession, req aiSessionMessageRequest, attachmentIDs []string) (openAIChatRequest, error) {
	rules, err := s.loadAIRules()
	if err != nil {
		return openAIChatRequest{}, err
	}

	systemPrompt := templateRulesSummary()
	if strings.TrimSpace(rules) != "" {
		systemPrompt += "\n\n全局模板规范补充：\n" + strings.TrimSpace(rules)
	}
	if strings.TrimSpace(req.SessionRules) != "" {
		systemPrompt += "\n\n当前会话补充规则：\n" + strings.TrimSpace(req.SessionRules)
	}

	if configContext := buildConfigPromptContext(s.cfg); strings.TrimSpace(configContext) != "" {
		systemPrompt += "\n\n当前配置上下文：\n" + configContext
	}
	systemPrompt += "\n\n输出要求补充：请尽量返回 configPatch，用于同步 serviceTop 和 serverConfig.vars；如果信息不足，请在 followUpQuestions 里明确提出。"

	messages := []openAIChatMessage{newTextMessage("system", systemPrompt)}

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
		messages = append(messages, newTextMessage(message.Role, builder))
	}

	selectedDrafts := make([]aiDraftFile, 0)
	if len(req.SelectedDraftPaths) == 0 {
		selectedDrafts = append(selectedDrafts, session.DraftFiles...)
	} else {
		selected := make(map[string]struct{}, len(req.SelectedDraftPaths))
		for _, path := range req.SelectedDraftPaths {
			selected[path] = struct{}{}
		}
		for _, draft := range session.DraftFiles {
			if _, ok := selected[draft.Path]; ok {
				selectedDrafts = append(selectedDrafts, draft)
			}
		}
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

	_, settings, err := s.getAIAPIKey()
	if err != nil {
		return openAIChatRequest{}, err
	}

	return openAIChatRequest{
		Model:          settings.Model,
		Messages:       messages,
		ResponseFormat: map[string]string{"type": "json_object"},
		Temperature:    0.2,
		MaxTokens:      3200,
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
	json.NewEncoder(w).Encode(result)
}

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

func (s *Server) handleAITemplateSessionCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := s.createAISession()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(session)
}

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
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)
		return
	}

	switch parts[1] {
	case "message":
		s.handleAISessionMessage(w, r, session)
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

func (s *Server) handleAISessionMessage(w http.ResponseWriter, r *http.Request, session *aiSession) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req aiSessionMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	attachmentIDs := consumePendingAttachments(session)
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

	rawResponse, err := doOpenAIChatCompletion(normalizeBaseURL(settings.BaseURL), apiKey, chatReq)
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
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if modelResponse.PlannedActions, err = validatePlannedActions(modelResponse.PlannedActions); err != nil {
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
	})

	session.SessionRules = req.SessionRules
	session.DraftFiles = mergeDraftFiles(session.DraftFiles, modelResponse.DraftFiles)
	session.PlannedActions = modelResponse.PlannedActions
	session.ConfigPatch = mergeAIConfigPatches(session.ConfigPatch, modelResponse.ConfigPatch)
	session.ConfigIssues = mergeAIConfigIssues(session.ConfigIssues, configIssues)

	if err := s.saveAISession(session); err != nil {
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

	if err := s.saveAISession(session); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(attachment)
}

func (s *Server) handleAISessionSave(w http.ResponseWriter, r *http.Request, session *aiSession) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req aiSessionSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Files) == 0 {
		http.Error(w, "files 不能为空", http.StatusBadRequest)
		return
	}

	savedFiles, createdDirs, err := s.saveDraftFiles(req.Files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	configApplied := false
	appliedServices := make([]string, 0)
	selectedServices := selectServicesFromDrafts(req.Files)
	selectedConfigIssues := selectConfigIssues(session.ConfigIssues, selectedServices)
	if req.ApplyConfigPatch {
		selectedPatch := selectConfigPatchServices(session.ConfigPatch, selectedServices)
		if hasBlockingConfigIssues(selectedConfigIssues) {
			for idx := range session.DraftFiles {
				for _, savedPath := range savedFiles {
					if session.DraftFiles[idx].Path == savedPath {
						session.DraftFiles[idx].NeedsReview = false
					}
				}
			}
			if err := s.saveAISession(session); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"savedFiles":      savedFiles,
				"createdDirs":     createdDirs,
				"configApplied":   false,
				"appliedServices": appliedServices,
				"configIssues":    selectedConfigIssues,
				"message":         "模板已保存，但配置补丁存在阻塞问题，请先修正后再同步配置",
			})
			return
		}

		appliedServices = applyConfigPatchToConfig(s.cfg, selectedPatch)
		if err := config.SaveConfig(s.configPath, s.cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"savedFiles":      savedFiles,
		"createdDirs":     createdDirs,
		"configApplied":   configApplied,
		"appliedServices": appliedServices,
		"configIssues":    selectedConfigIssues,
	})
}

func (s *Server) handleAISessionAttachment(w http.ResponseWriter, r *http.Request, session *aiSession, attachmentID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	attachment, err := s.findAttachment(session, attachmentID)
	if err != nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, attachment.StoredPath)
}
