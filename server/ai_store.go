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
	return nil
}

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

func (s *Server) populateAttachmentURLs(session *aiSession) {
	for idx := range session.Attachments {
		session.Attachments[idx].DownloadURL = fmt.Sprintf(
			"/api/ai/template/session/%s/attachment/%s",
			url.PathEscape(session.ID),
			url.PathEscape(session.Attachments[idx].ID),
		)
	}
}

func (s *Server) loadAISession(sessionID string) (*aiSession, error) {
	data, err := os.ReadFile(s.sessionPath(sessionID))
	if err != nil {
		return nil, err
	}
	var session aiSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	session.ConfigPatch = normalizeAIConfigPatch(session.ConfigPatch)
	s.populateAttachmentURLs(&session)
	return &session, nil
}

func (s *Server) saveAISession(session *aiSession) error {
	if err := s.ensureAIDirs(); err != nil {
		return err
	}
	session.UpdatedAt = nowRFC3339()
	sort.SliceStable(session.Attachments, func(i, j int) bool {
		return session.Attachments[i].CreatedAt < session.Attachments[j].CreatedAt
	})
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.sessionPath(session.ID), data, 0644)
}

func (s *Server) createAISession() (*aiSession, error) {
	sessionID, err := generateID("session_")
	if err != nil {
		return nil, err
	}
	session := &aiSession{
		ID:             sessionID,
		Messages:       []aiSessionMessage{},
		DraftFiles:     []aiDraftFile{},
		PlannedActions: []aiPlannedAction{},
		ConfigPatch:    emptyAIConfigPatch(),
		ConfigIssues:   []aiConfigIssue{},
		Attachments:    []aiSessionAttachment{},
		CreatedAt:      nowRFC3339(),
		UpdatedAt:      nowRFC3339(),
	}
	if err := s.saveAISession(session); err != nil {
		return nil, err
	}
	s.populateAttachmentURLs(session)
	return session, nil
}

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

func validatePlannedActions(actions []aiPlannedAction) ([]aiPlannedAction, error) {
	validated := make([]aiPlannedAction, 0, len(actions))
	for _, action := range actions {
		action.Type = strings.TrimSpace(action.Type)
		action.Path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(action.Path)))
		if action.Type != "mkdir" && action.Type != "write_file" {
			return nil, fmt.Errorf("不支持的计划动作类型: %s", action.Type)
		}
		if !strings.HasPrefix(action.Path, "templates/") {
			return nil, fmt.Errorf("计划动作路径必须位于 templates/ 目录下: %s", action.Path)
		}
		validated = append(validated, action)
	}
	return validated, nil
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
- toUpper, toLower, trim, replace, default
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
