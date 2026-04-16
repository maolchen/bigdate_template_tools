package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"config-generator/config"
	"config-generator/generator"
)

const (
	aiMasterKeyEnv        = "CONFIG_GENERATOR_AI_MASTER_KEY"
	aiTextFileLimit       = 1 << 20
	aiImageFileLimit      = 5 << 20
	aiMaxAttachments      = 3
	aiMaxImageAttachments = 2
)

var (
	errAIMasterKeyMissing = errors.New("未设置 CONFIG_GENERATOR_AI_MASTER_KEY，AI 功能无法解密或保存 API Key")
	errAISettingsMissing  = errors.New("尚未配置 AI 接口信息")
)

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

func defaultAISessionTitle() string {
	return "新会话"
}

func defaultAISettings() aiSettingsFile {
	return aiSettingsFile{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4.1",
	}
}

func maskAPIKey(apiKey string) string {
	if strings.TrimSpace(apiKey) == "" {
		return ""
	}
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

func generateID(prefix string) (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}

// ensureAIDirs creates all AI-related data directories required by settings, sessions and uploads.
func (s *Server) ensureAIDirs() error {
	if err := os.MkdirAll(s.aiDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.aiSessionsDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.aiUploadsDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.aiSkillsDir, 0755); err != nil {
		return err
	}
	if err := s.ensureDefaultAISkillFiles(); err != nil {
		return err
	}
	return nil
}

// loadAISettingsFile loads persisted AI endpoint settings and fills defaults for missing fields.
func (s *Server) loadAISettingsFile() (aiSettingsFile, error) {
	settings := defaultAISettings()
	data, err := os.ReadFile(s.aiSettingsPath)
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return aiSettingsFile{}, err
	}
	if len(data) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return aiSettingsFile{}, err
	}
	if strings.TrimSpace(settings.BaseURL) == "" {
		settings.BaseURL = defaultAISettings().BaseURL
	}
	if strings.TrimSpace(settings.Model) == "" {
		settings.Model = defaultAISettings().Model
	}
	return settings, nil
}

// saveAISettingsFile persists AI endpoint settings to disk.
func (s *Server) saveAISettingsFile(settings aiSettingsFile) error {
	if err := s.ensureAIDirs(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.aiSettingsPath, data, 0644)
}

// getAIMasterKey derives a stable AES key from CONFIG_GENERATOR_AI_MASTER_KEY.
func (s *Server) getAIMasterKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(aiMasterKeyEnv))
	if raw == "" {
		return nil, errAIMasterKeyMissing
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

func encryptWithKey(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptWithKey(key []byte, ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("密文格式无效")
	}
	nonce := raw[:gcm.NonceSize()]
	payload := raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// getAISettingsResponse returns masked settings for UI consumption without exposing plain API keys.
func (s *Server) getAISettingsResponse() (aiSettingsResponse, error) {
	settings, err := s.loadAISettingsFile()
	if err != nil {
		return aiSettingsResponse{}, err
	}

	resp := aiSettingsResponse{
		BaseURL:   settings.BaseURL,
		Model:     settings.Model,
		HasAPIKey: strings.TrimSpace(settings.APIKeyEncrypted) != "",
		UpdatedAt: settings.UpdatedAt,
	}

	if !resp.HasAPIKey {
		return resp, nil
	}

	key, err := s.getAIMasterKey()
	if err != nil {
		resp.MaskedAPIKey = "已配置"
		return resp, nil
	}
	apiKey, err := decryptWithKey(key, settings.APIKeyEncrypted)
	if err != nil {
		resp.MaskedAPIKey = "已配置"
		return resp, nil
	}
	resp.MaskedAPIKey = maskAPIKey(apiKey)
	return resp, nil
}

// updateAISettings updates endpoint/model values and rotates encrypted API key when provided.
func (s *Server) updateAISettings(req aiSettingsUpdateRequest) (aiSettingsResponse, error) {
	settings, err := s.loadAISettingsFile()
	if err != nil {
		return aiSettingsResponse{}, err
	}

	baseURL := strings.TrimSpace(req.BaseURL)
	if baseURL == "" {
		baseURL = settings.BaseURL
		if baseURL == "" {
			baseURL = defaultAISettings().BaseURL
		}
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = settings.Model
		if model == "" {
			model = defaultAISettings().Model
		}
	}

	settings.BaseURL = strings.TrimRight(baseURL, "/")
	settings.Model = model

	if req.ClearAPIKey {
		settings.APIKeyEncrypted = ""
	} else if strings.TrimSpace(req.APIKey) != "" {
		key, err := s.getAIMasterKey()
		if err != nil {
			return aiSettingsResponse{}, err
		}
		encrypted, err := encryptWithKey(key, strings.TrimSpace(req.APIKey))
		if err != nil {
			return aiSettingsResponse{}, err
		}
		settings.APIKeyEncrypted = encrypted
	}

	settings.UpdatedAt = nowRFC3339()
	if err := s.saveAISettingsFile(settings); err != nil {
		return aiSettingsResponse{}, err
	}
	return s.getAISettingsResponse()
}

// getAIAPIKey decrypts and returns the current API key plus resolved settings.
func (s *Server) getAIAPIKey() (string, aiSettingsFile, error) {
	settings, err := s.loadAISettingsFile()
	if err != nil {
		return "", aiSettingsFile{}, err
	}
	if strings.TrimSpace(settings.APIKeyEncrypted) == "" {
		return "", aiSettingsFile{}, errAISettingsMissing
	}
	key, err := s.getAIMasterKey()
	if err != nil {
		return "", aiSettingsFile{}, err
	}
	apiKey, err := decryptWithKey(key, settings.APIKeyEncrypted)
	if err != nil {
		return "", aiSettingsFile{}, err
	}
	return apiKey, settings, nil
}

func (s *Server) loadAIRules() (string, error) {
	data, err := os.ReadFile(s.aiRulesPath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Server) saveAIRules(content string) error {
	if err := s.ensureAIDirs(); err != nil {
		return err
	}
	return os.WriteFile(s.aiRulesPath, []byte(content), 0644)
}

func (s *Server) sessionPath(sessionID string) string {
	return filepath.Join(s.aiSessionsDir, sessionID+".json")
}

func (s *Server) uploadDir(sessionID string) string {
	return filepath.Join(s.aiUploadsDir, sessionID)
}

func (s *Server) attachmentStoredPath(sessionID string, attachment aiSessionAttachment) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(attachment.Name)))
	filename := attachment.ID
	if ext != "" {
		filename += ext
	}
	return filepath.Join(s.uploadDir(sessionID), filename)
}

func (s *Server) populateAttachmentURLs(session *aiSession) {
	for idx := range session.Attachments {
		session.Attachments[idx].DownloadURL = fmt.Sprintf(
			"/api/ai/template/session/%s/attachment/%s",
			url.PathEscape(session.ID),
			url.PathEscape(session.Attachments[idx].ID),
		)
	}
}

// loadAISession loads a single persisted session and normalizes optional fields for backward compatibility.
func (s *Server) loadAISession(sessionID string) (*aiSession, error) {
	sessionPath := s.sessionPath(sessionID)
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, err
	}
	var session aiSession
	if err := json.Unmarshal(data, &session); err != nil {
		fmt.Printf("[AI] session load failed session=%s path=%s bytes=%d err=%v\n", sessionID, sessionPath, len(data), err)
		return nil, err
	}
	session.ConfigPatch = normalizeAIConfigPatch(session.ConfigPatch)
	session.PromptTrace = normalizeAIPromptTrace(session.PromptTrace)
	session.SelectedSkillIDs = normalizeSelectedSkillIDs(session.SelectedSkillIDs)
	session.SelectedModel = strings.TrimSpace(session.SelectedModel)
	if strings.TrimSpace(session.Title) == "" {
		session.Title = deriveAISessionTitle(&session)
	}
	for idx := range session.Attachments {
		if strings.TrimSpace(session.Attachments[idx].StoredPath) == "" {
			session.Attachments[idx].StoredPath = s.attachmentStoredPath(session.ID, session.Attachments[idx])
		}
	}
	for idx := range session.Messages {
		session.Messages[idx].PromptTrace = normalizeAIPromptTrace(session.Messages[idx].PromptTrace)
		session.Messages[idx].Model = strings.TrimSpace(session.Messages[idx].Model)
	}
	s.populateAttachmentURLs(&session)
	fmt.Printf("[AI] session loaded session=%s path=%s messages=%d drafts=%d attachments=%d bytes=%d\n",
		session.ID, sessionPath, len(session.Messages), len(session.DraftFiles), len(session.Attachments), len(data))
	return &session, nil
}

// saveAISession persists one AI session atomically with normalized metadata.
func (s *Server) saveAISession(session *aiSession) error {
	if err := s.ensureAIDirs(); err != nil {
		return err
	}
	session.Title = strings.TrimSpace(session.Title)
	if session.Title == "" {
		session.Title = deriveAISessionTitle(session)
	}
	session.UpdatedAt = nowRFC3339()
	session.SelectedSkillIDs = normalizeSelectedSkillIDs(session.SelectedSkillIDs)
	session.SelectedModel = strings.TrimSpace(session.SelectedModel)
	for idx := range session.Messages {
		session.Messages[idx].Model = strings.TrimSpace(session.Messages[idx].Model)
	}
	sort.SliceStable(session.Attachments, func(i, j int) bool {
		return session.Attachments[i].CreatedAt < session.Attachments[j].CreatedAt
	})
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	sessionPath := s.sessionPath(session.ID)
	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("[AI] session saved session=%s path=%s messages=%d drafts=%d attachments=%d bytes=%d\n",
		session.ID, sessionPath, len(session.Messages), len(session.DraftFiles), len(session.Attachments), len(data))
	return nil
}

// createAISession allocates a new empty conversation workspace with current default model and timestamps.
func (s *Server) createAISession() (*aiSession, error) {
	sessionID, err := generateID("session_")
	if err != nil {
		return nil, err
	}
	settings, err := s.loadAISettingsFile()
	if err != nil {
		return nil, err
	}
	session := &aiSession{
		ID:               sessionID,
		Title:            defaultAISessionTitle(),
		Messages:         []aiSessionMessage{},
		DraftFiles:       []aiDraftFile{},
		PlannedActions:   []aiPlannedAction{},
		ConfigPatch:      emptyAIConfigPatch(),
		ConfigIssues:     []aiConfigIssue{},
		Attachments:      []aiSessionAttachment{},
		PromptTrace:      emptyAIPromptTrace(),
		SelectedSkillIDs: []string{},
		SelectedModel:    strings.TrimSpace(settings.Model),
		CreatedAt:        nowRFC3339(),
		UpdatedAt:        nowRFC3339(),
	}
	if err := s.saveAISession(session); err != nil {
		return nil, err
	}
	s.populateAttachmentURLs(session)
	fmt.Printf("[AI] session initialized id=%s model=%s path=%s\n", session.ID, session.SelectedModel, s.sessionPath(session.ID))
	return session, nil
}

func deriveAISessionTitle(session *aiSession) string {
	if session == nil {
		return defaultAISessionTitle()
	}
	if title := strings.TrimSpace(session.Title); title != "" {
		if title != defaultAISessionTitle() {
			return title
		}
	}
	for _, message := range session.Messages {
		if strings.TrimSpace(message.Role) != "user" {
			continue
		}
		if title := summarizeSessionTitle(message.Content); title != "" {
			return title
		}
	}
	return defaultAISessionTitle()
}

func summarizeSessionTitle(raw string) string {
	title := strings.TrimSpace(raw)
	if title == "" {
		return ""
	}
	title = strings.Join(strings.Fields(title), " ")
	runes := []rune(title)
	if len(runes) > 32 {
		title = string(runes[:32]) + "..."
	}
	return title
}

func summarizeAISession(session *aiSession) aiSessionSummary {
	return aiSessionSummary{
		ID:           session.ID,
		Title:        deriveAISessionTitle(session),
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
		MessageCount: len(session.Messages),
		DraftCount:   len(session.DraftFiles),
	}
}

func (s *Server) listAISessions() ([]aiSessionSummary, error) {
	if err := s.ensureAIDirs(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.aiSessionsDir)
	if err != nil {
		return nil, err
	}

	sessions := make([]aiSessionSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(s.aiSessionsDir, entry.Name()))
		if readErr != nil {
			return nil, readErr
		}
		var session aiSession
		if err := json.Unmarshal(data, &session); err != nil {
			return nil, err
		}
		sessions = append(sessions, summarizeAISession(&session))
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].UpdatedAt == sessions[j].UpdatedAt {
			return sessions[i].CreatedAt > sessions[j].CreatedAt
		}
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})
	return sessions, nil
}

func (s *Server) updateAISessionTitle(session *aiSession, title string) error {
	session.Title = strings.TrimSpace(title)
	if session.Title == "" {
		session.Title = deriveAISessionTitle(session)
	}
	return s.saveAISession(session)
}

// deleteAISession removes the session record and its upload directory.
func (s *Server) deleteAISession(sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("session id 不能为空")
	}

	defer s.deleteAISessionLock(sessionID)
	sessionFile := s.sessionPath(sessionID)
	if err := os.Remove(sessionFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	uploadDir := s.uploadDir(sessionID)
	if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// normalizeTemplatePath validates that a target path stays inside templates/ and ends with .tmpl.
func (s *Server) normalizeTemplatePath(input string) (string, string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(input)))
	clean = strings.TrimPrefix(clean, "./")
	if !strings.HasPrefix(clean, "templates/") {
		return "", "", errors.New("模板路径必须位于 templates/ 目录下")
	}
	if !strings.HasSuffix(clean, ".tmpl") {
		return "", "", errors.New("仅允许保存 .tmpl 模板文件")
	}
	fullPath := filepath.Join(filepath.Dir(s.templatesDir), filepath.FromSlash(clean))
	templatesRoot := filepath.Clean(s.templatesDir)
	if !strings.HasPrefix(filepath.Clean(fullPath), templatesRoot) {
		return "", "", errors.New("模板路径无效")
	}
	return clean, fullPath, nil
}

// validateDraftTemplate parses draft content with the same template function map used by generation.
func (s *Server) validateDraftTemplate(content string) error {
	emptyCfg := &config.Config{
		Global:       config.Global{},
		Nodes:        config.Nodes{},
		ServiceTop:   config.ServiceTopos{},
		ServerConfig: config.ServiceConfigs{},
	}
	_, err := generator.ParseTemplateContent("draft", content, config.Context{}, emptyCfg)
	return err
}

// validatePlannedActions normalizes model-produced actions into the supported mkdir/write_file/remove set.
func validatePlannedActions(actions []aiPlannedAction) ([]aiPlannedAction, error) {
	validated := make([]aiPlannedAction, 0, len(actions))
	for _, action := range actions {
		rawType := strings.ToLower(strings.TrimSpace(action.Type))
		rawPath := strings.TrimSpace(action.Path)
		if isNoopPlannedActionType(rawType) {
			// Control/meta actions are not filesystem mutations; ignore instead of failing.
			continue
		}
		action.Path = filepath.ToSlash(filepath.Clean(rawPath))
		switch rawType {
		case "mkdir", "create_dir", "create_directory", "directory", "dir", "ensure_dir", "ensure_directory", "make_dir", "make_directory", "create_folder", "ensure_folder", "folder":
			action.Type = "mkdir"
		case "write_file", "create_file", "file", "write", "save", "save_file", "ensure_file", "template", "template_file", "create_template", "write_template", "save_template", "generate_template", "render_template", "create_script", "write_script", "update", "update_file", "edit", "edit_file", "modify", "modify_file", "rewrite", "overwrite", "append":
			action.Type = "write_file"
		case "remove", "delete", "delete_file", "remove_file", "unlink", "rm", "delete_template", "remove_template", "delete_script", "remove_script", "purge":
			action.Type = "remove"
		case "create":
			if strings.HasSuffix(action.Path, ".tmpl") || filepath.Ext(action.Path) != "" {
				action.Type = "write_file"
			} else {
				action.Type = "mkdir"
			}
		default:
			switch {
			case strings.Contains(rawType, "template"), strings.Contains(rawType, "file"), strings.Contains(rawType, "script"), strings.Contains(rawType, "render"):
				action.Type = "write_file"
			case strings.Contains(rawType, "remove"), strings.Contains(rawType, "delete"), strings.Contains(rawType, "unlink"):
				action.Type = "remove"
			case strings.Contains(rawType, "dir"), strings.Contains(rawType, "folder"), strings.Contains(rawType, "directory"):
				action.Type = "mkdir"
			default:
				if action.Path == "." || action.Path == "" {
					// Unknown control-like action without path target: ignore.
					continue
				}
				action.Type = rawType
			}
		}
		if action.Type != "mkdir" && action.Type != "write_file" && action.Type != "remove" {
			return nil, fmt.Errorf("不支持的计划动作类型: %s", action.Type)
		}
		if !strings.HasPrefix(action.Path, "templates/") {
			return nil, fmt.Errorf("计划动作路径必须位于 templates/ 目录下: %s", action.Path)
		}
		validated = append(validated, action)
	}
	return validated, nil
}

func isNoopPlannedActionType(rawType string) bool {
	switch rawType {
	case "", "noop", "none", "n/a", "ignore", "skip", "ack", "confirm", "confirm_draft", "review", "approval", "approve", "finalize", "complete", "finish":
		return true
	default:
		return false
	}
}

func detectAttachmentKind(contentType string) (string, bool) {
	switch contentType {
	case "image/png", "image/jpeg", "image/webp":
		return "image", true
	}

	if strings.HasPrefix(contentType, "text/") {
		return "text", true
	}

	switch contentType {
	case "application/json", "application/xml", "text/xml":
		return "text", true
	}

	return "", false
}

func allowedAttachmentByName(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".sh", ".tmpl", ".conf", ".yaml", ".yml", ".xml", ".properties", ".json", ".md", ".txt", ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

func loadTextPreview(path string, maxBytes int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(data) > maxBytes {
		data = data[:maxBytes]
	}
	return string(data)
}

func (s *Server) listPendingAttachments(session *aiSession) []aiSessionAttachment {
	pending := make([]aiSessionAttachment, 0)
	for _, attachment := range session.Attachments {
		if attachment.Pending {
			pending = append(pending, attachment)
		}
	}
	return pending
}

func (s *Server) countPendingKinds(session *aiSession) (int, int) {
	total := 0
	images := 0
	for _, attachment := range session.Attachments {
		if !attachment.Pending {
			continue
		}
		total++
		if attachment.Kind == "image" {
			images++
		}
	}
	return total, images
}

// saveUploadedAttachment validates limits/types, stores attachment binaries and appends pending metadata to session.
func (s *Server) saveUploadedAttachment(session *aiSession, originalName, contentType string, data []byte) (*aiSessionAttachment, error) {
	totalPending, imagePending := s.countPendingKinds(session)
	if totalPending >= aiMaxAttachments {
		return nil, fmt.Errorf("单次消息最多附加 %d 个文件", aiMaxAttachments)
	}

	attachmentKind, ok := detectAttachmentKind(contentType)
	if !ok {
		return nil, errors.New("仅支持文本文件或 png/jpg/jpeg/webp 图片")
	}
	if attachmentKind == "image" {
		if imagePending >= aiMaxImageAttachments {
			return nil, fmt.Errorf("单次消息最多附加 %d 张图片", aiMaxImageAttachments)
		}
		if len(data) > aiImageFileLimit {
			return nil, fmt.Errorf("图片大小不能超过 %d MB", aiImageFileLimit>>20)
		}
	} else if len(data) > aiTextFileLimit {
		return nil, fmt.Errorf("文本文件大小不能超过 %d MB", aiTextFileLimit>>20)
	}

	if !allowedAttachmentByName(originalName) {
		return nil, errors.New("文件类型不受支持")
	}

	attachmentID, err := generateID("att_")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.uploadDir(session.ID), 0755); err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(originalName))
	storedPath := filepath.Join(s.uploadDir(session.ID), attachmentID+ext)
	if err := os.WriteFile(storedPath, data, 0644); err != nil {
		return nil, err
	}

	attachment := aiSessionAttachment{
		ID:         attachmentID,
		Name:       originalName,
		Kind:       attachmentKind,
		MimeType:   contentType,
		Size:       int64(len(data)),
		StoredPath: storedPath,
		Pending:    true,
		CreatedAt:  nowRFC3339(),
	}
	if attachmentKind == "text" {
		attachment.PreviewText = string(data)
	}
	session.Attachments = append(session.Attachments, attachment)
	s.populateAttachmentURLs(session)
	return &session.Attachments[len(session.Attachments)-1], nil
}

func (s *Server) findAttachment(session *aiSession, attachmentID string) (*aiSessionAttachment, error) {
	for idx := range session.Attachments {
		if session.Attachments[idx].ID == attachmentID {
			return &session.Attachments[idx], nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *Server) deletePendingAttachment(session *aiSession, attachmentID string) error {
	filtered := make([]aiSessionAttachment, 0, len(session.Attachments))
	removed := false
	var target aiSessionAttachment

	for _, attachment := range session.Attachments {
		if attachment.ID != attachmentID {
			filtered = append(filtered, attachment)
			continue
		}
		if !attachment.Pending {
			return errors.New("仅支持取消当前待发送的附件")
		}
		removed = true
		target = attachment
	}

	if !removed {
		return os.ErrNotExist
	}

	session.Attachments = filtered
	if storedPath := strings.TrimSpace(target.StoredPath); storedPath != "" {
		if err := os.Remove(storedPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func consumePendingAttachments(session *aiSession) []string {
	ids := make([]string, 0)
	for idx := range session.Attachments {
		if !session.Attachments[idx].Pending {
			continue
		}
		session.Attachments[idx].Pending = false
		ids = append(ids, session.Attachments[idx].ID)
	}
	return ids
}

func (s *Server) buildMessageAttachmentPreview(session *aiSession, attachmentIDs []string) []string {
	previews := make([]string, 0, len(attachmentIDs))
	for _, attachmentID := range attachmentIDs {
		attachment, err := s.findAttachment(session, attachmentID)
		if err != nil {
			continue
		}
		previews = append(previews, attachment.Name)
	}
	return previews
}

func validateDraftTemplateContract(path string, content string) error {
	forbidden := []string{".Instance.NodeAlias", ".Instance.Node.HostName"}
	for _, token := range forbidden {
		if strings.Contains(content, token) {
			return fmt.Errorf("%s 包含禁用占位符 %s，请改为 {{ .Instance.Node.Hostname }}", path, token)
		}
	}
	return nil
}

// saveDraftFiles validates and writes reviewed drafts into templates/, creating parent directories as needed.
func (s *Server) saveDraftFiles(files []aiDraftFile) ([]string, []string, error) {
	savedFiles := make([]string, 0, len(files))
	createdDirSet := make(map[string]struct{})

	for _, file := range files {
		normalizedPath, fullPath, err := s.normalizeTemplatePath(file.Path)
		if err != nil {
			return nil, nil, err
		}
		if err := s.validateDraftTemplate(file.Content); err != nil {
			return nil, nil, fmt.Errorf("%s 模板语法错误: %w", normalizedPath, err)
		}
		if err := validateDraftTemplateContract(normalizedPath, file.Content); err != nil {
			fmt.Printf("[AI] save reject path=%s reason=%v\n", normalizedPath, err)
			return nil, nil, err
		}
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return nil, nil, err
		}
		relDir, err := filepath.Rel(s.templatesDir, parentDir)
		if err == nil && relDir != "." {
			createdDirSet[filepath.ToSlash(filepath.Join("templates", relDir))] = struct{}{}
		}
		if err := os.WriteFile(fullPath, []byte(file.Content), 0644); err != nil {
			return nil, nil, err
		}
		hash := sha256.Sum256([]byte(file.Content))
		fmt.Printf("[AI] save write path=%s abs=%s bytes=%d sha256=%s\n",
			normalizedPath, filepath.Clean(fullPath), len(file.Content), hex.EncodeToString(hash[:8]))
		savedFiles = append(savedFiles, normalizedPath)
	}

	createdDirs := make([]string, 0, len(createdDirSet))
	for dir := range createdDirSet {
		if dir == "." || dir == "templates" || dir == "" {
			continue
		}
		createdDirs = append(createdDirs, dir)
	}
	sort.Strings(createdDirs)
	sort.Strings(savedFiles)
	return savedFiles, createdDirs, nil
}

// deleteDraftFiles removes drafts from session state and optionally deletes persisted template files.
func (s *Server) deleteDraftFiles(session *aiSession, paths []string, removeFromDisk bool) ([]string, []string, error) {
	if len(paths) == 0 {
		return nil, nil, errors.New("paths 不能为空")
	}

	targets := make(map[string]struct{}, len(paths))
	for _, rawPath := range paths {
		normalizedPath, fullPath, err := s.normalizeTemplatePath(rawPath)
		if err != nil {
			return nil, nil, err
		}
		targets[normalizedPath] = struct{}{}
		if removeFromDisk {
			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				return nil, nil, err
			}
			s.cleanupEmptyTemplateDirs(filepath.Dir(fullPath))
		}
	}

	deletedDrafts := make([]string, 0, len(targets))
	for path := range targets {
		deletedDrafts = append(deletedDrafts, path)
	}
	sort.Strings(deletedDrafts)

	filterDrafts := func(drafts []aiDraftFile) []aiDraftFile {
		result := make([]aiDraftFile, 0, len(drafts))
		for _, draft := range drafts {
			if _, ok := targets[draft.Path]; ok {
				continue
			}
			result = append(result, draft)
		}
		return result
	}

	filterActions := func(actions []aiPlannedAction) []aiPlannedAction {
		result := make([]aiPlannedAction, 0, len(actions))
		for _, action := range actions {
			if _, ok := targets[action.Path]; ok {
				continue
			}
			result = append(result, action)
		}
		return result
	}

	session.DraftFiles = filterDrafts(session.DraftFiles)
	session.PlannedActions = filterActions(session.PlannedActions)
	for idx := range session.Messages {
		session.Messages[idx].DraftFiles = filterDrafts(session.Messages[idx].DraftFiles)
		session.Messages[idx].PlannedActions = filterActions(session.Messages[idx].PlannedActions)
	}

	deletedTemplates := []string{}
	if removeFromDisk {
		deletedTemplates = append(deletedTemplates, deletedDrafts...)
	}
	if err := s.saveAISession(session); err != nil {
		return nil, nil, err
	}
	return deletedDrafts, deletedTemplates, nil
}

func (s *Server) cleanupEmptyTemplateDirs(dir string) {
	root := filepath.Clean(s.templatesDir)
	current := filepath.Clean(dir)
	for current != root && strings.HasPrefix(current, root) {
		entries, err := os.ReadDir(current)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(current); err != nil {
			return
		}
		current = filepath.Dir(current)
	}
}

func templateRulesSummary() string {
	return strings.TrimSpace(`
你正在为一个基于 Go Template 的大数据平台离线交付系统生成 templates/*.tmpl 文件。

必须遵守的项目规范：
1. 目录路径优先使用全局变量拼接，例如 {{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }} 和 {{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}。
2. 用户、用户组、JAVA_HOME 优先使用 {{ .Global.user }}、{{ .Global.group }}、{{ .Global.java_home }}，不要重复定义服务级 run_user/run_group/java_home。
3. shell 脚本必须尽量幂等：mkdir -p、覆盖前备份、软链接先删后建、重复执行结果一致。
4. 非必要不要使用 root 直接启动服务；需要 root 权限的操作通过 run_as_root 包装，普通运行逻辑优先使用 {{ .Global.user }}。
5. systemd 服务中 User/Group 也应优先使用全局用户和用户组。
6. 模板文件路径必须位于 templates/<service>/ 下，并以 .tmpl 结尾。
7. 你只能输出草稿计划，不要假设文件已经落盘。

可用模板上下文：
- .Global
- .Nodes
- .Instance
- .AllInstances

常用模板函数：
- toUpper, toLower, trim, replace, join, split, default
- add, sub, mul, div
- serviceNodes, serviceEndpoints, serviceEndpointsJoin
- serviceIPs, serviceHostnames, getServiceNodes
- serviceVars, serviceVar, serviceConfig, nodeInfo

图片只用于识别截图或拍照中的可见文字；如果有识别不清、被截断、遮挡、低分辨率等风险，必须在 warnings 中明确指出。

响应必须是单个 JSON 对象，字段如下：
{
  "assistantMessage": "给用户的简洁说明",
  "draftFiles": [{"path":"templates/<service>/install.sh.tmpl","content":"...","reason":"为什么生成该文件"}],
  "plannedActions": [{"type":"mkdir","path":"templates/<service>","reason":"为什么需要此目录"}],
  "warnings": ["需要人工复核的风险"],
  "followUpQuestions": ["如果还缺信息，提简短问题"]
}
`)
}
