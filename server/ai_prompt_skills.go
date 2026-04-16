package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type aiPromptSkill struct {
	Ref     aiSkillRef
	Path    string
	Content string
}

type aiPromptSkillSpec struct {
	RelativePath string
	Scope        string
	Title        string
	Tags         []string
	Required     bool
	DefaultBody  string
}

type aiPromptBundle struct {
	SystemPrompt string
	PromptTrace  aiPromptTrace
	Structured   bool
}

func emptyAIPromptTrace() aiPromptTrace {
	return aiPromptTrace{
		SkillRefs:   []aiSkillRef{},
		ExampleRefs: []aiExampleRef{},
		Mode:        "layered-skills",
		Provider:    "chat-completions",
	}
}

func normalizeAIPromptTrace(trace aiPromptTrace) aiPromptTrace {
	if trace.SkillRefs == nil {
		trace.SkillRefs = []aiSkillRef{}
	}
	if trace.ExampleRefs == nil {
		trace.ExampleRefs = []aiExampleRef{}
	}
	if strings.TrimSpace(trace.Mode) == "" {
		trace.Mode = "layered-skills"
	}
	if strings.TrimSpace(trace.Provider) == "" {
		trace.Provider = "chat-completions"
	}
	return trace
}

func defaultAISkillSpecs() []aiPromptSkillSpec {
	return []aiPromptSkillSpec{
		{
			RelativePath: "core/base.md",
			Scope:        "core",
			Title:        "Base Project Rules",
			Tags:         []string{"global", "idempotent"},
			Required:     true,
			DefaultBody:  "AI 助手主用途：生成符合项目规范、支持 Go Template 语法、可审核、可保存的 .tmpl 模板草稿。",
		},
		{
			RelativePath: "core/output_json.md",
			Scope:        "core",
			Title:        "Structured JSON Output",
			Tags:         []string{"json", "output"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
请只返回一个 JSON object，不要输出任何 JSON 之外的解释文本。`),
		},
		{
			RelativePath: "core/path_guard.md",
			Scope:        "core",
			Title:        "Template Path Guard",
			Tags:         []string{"path", "templates"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
路径约束：
- 所有草稿文件必须位于 templates/<service>/...
- 所有模板文件必须以 .tmpl 结尾`),
		},
		{
			RelativePath: "core/template_contract.md",
			Scope:        "core",
			Title:        "Template Contract",
			Tags:         []string{"contract", "hostname", "placeholder"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
模板契约：
- 草稿文件路径必须位于 templates/<service>/
- 节点主机名必须使用 {{ .Instance.Node.Hostname }}
- 节点 IP 必须使用 {{ .Instance.Node.IP }}`),
		},
		{
			RelativePath: "core/variable_contract.md",
			Scope:        "core",
			Title:        "Variable Contract",
			Tags:         []string{"variables", "context", "schema"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
模板变量契约：
- .Global.<key> 来自 config.yaml.global
- .Instance.Vars.<key> 来自 serverConfig.<service>.vars
- 禁止使用 .Instance.NodeAlias 和 .Instance.Node.HostName`),
		},
		{
			RelativePath: "core/template_functions.md",
			Scope:        "core",
			Title:        "Template Function Reference",
			Tags:         []string{"functions", "helpers", "examples"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
模板函数参考：
- serviceNodes("service") 返回服务实例列表
- serviceEndpoints("service","port") 返回 ip:port 列表
- default("fallback", value) 用于空值兜底`),
		},
		{
			RelativePath: "task/install_script.md",
			Scope:        "task",
			Title:        "Install Script Rules",
			Tags:         []string{"shell", "install"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
用于 install.sh.tmpl 及少量必要的 start/stop/check 脚本。`),
		},
		{
			RelativePath: "task/config_file.md",
			Scope:        "task",
			Title:        "Config File Rules",
			Tags:         []string{"config"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
用于服务配置文件模板。`),
		},
		{
			RelativePath: "task/systemd_service.md",
			Scope:        "task",
			Title:        "Systemd Rules",
			Tags:         []string{"systemd", "service"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
用于 systemd unit 模板。`),
		},
		{
			RelativePath: "task/cluster_service.md",
			Scope:        "task",
			Title:        "Cluster Rules",
			Tags:         []string{"cluster"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
用于集群服务模板。`),
		},
		{
			RelativePath: "task/image_rewrite.md",
			Scope:        "task",
			Title:        "Image Rewrite Rules",
			Tags:         []string{"image", "ocr"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
当提供图片输入时，先提取可见文字，再按模板规范生成草稿。`),
		},
		{
			RelativePath: "task/config_patch.md",
			Scope:        "task",
			Title:        "Config Patch Rules",
			Tags:         []string{"configPatch"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
当模板变更引入或依赖变量时，返回最小可用的配置补丁建议。`),
		},
	}
}

func defaultAISkillSpecMap() map[string]aiPromptSkillSpec {
	specMap := make(map[string]aiPromptSkillSpec)
	for _, spec := range defaultAISkillSpecs() {
		specMap[filepath.ToSlash(spec.RelativePath)] = spec
	}
	return specMap
}

func (s *Server) ensureDefaultAISkillFiles() error {
	for _, spec := range defaultAISkillSpecs() {
		fullPath := filepath.Join(s.aiSkillsDir, filepath.FromSlash(spec.RelativePath))
		if _, err := os.Stat(fullPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(spec.DefaultBody+"\n"), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) loadPromptSkill(spec aiPromptSkillSpec) (aiPromptSkill, error) {
	relativePath := filepath.ToSlash(spec.RelativePath)
	fullPath := filepath.Join(s.aiSkillsDir, filepath.FromSlash(relativePath))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return aiPromptSkill{}, err
	}

	id := strings.TrimSuffix(filepath.ToSlash(relativePath), filepath.Ext(relativePath))
	return aiPromptSkill{
		Ref: aiSkillRef{
			ID:       id,
			Scope:    spec.Scope,
			Title:    spec.Title,
			Tags:     append([]string(nil), spec.Tags...),
			Required: spec.Required,
		},
		Path:    fullPath,
		Content: strings.TrimSpace(string(data)),
	}, nil
}

func humanizeSkillTitle(relativePath string) string {
	base := strings.TrimSuffix(filepath.Base(relativePath), filepath.Ext(relativePath))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	parts := strings.Fields(base)
	for idx := range parts {
		runes := []rune(parts[idx])
		if len(runes) == 0 {
			continue
		}
		if runes[0] >= 'a' && runes[0] <= 'z' {
			runes[0] = runes[0] - 'a' + 'A'
		}
		parts[idx] = string(runes)
	}
	return strings.Join(parts, " ")
}

func scopeFromRelativePath(relativePath string) string {
	parts := strings.Split(filepath.ToSlash(relativePath), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "custom"
	}
	return parts[0]
}

func buildSkillCatalogItem(spec aiPromptSkillSpec, source string, content string) aiSkillCatalogItem {
	return aiSkillCatalogItem{
		ID:       strings.TrimSuffix(filepath.ToSlash(spec.RelativePath), filepath.Ext(spec.RelativePath)),
		Scope:    spec.Scope,
		Title:    spec.Title,
		Tags:     append([]string(nil), spec.Tags...),
		Required: spec.Required,
		Path:     filepath.ToSlash(spec.RelativePath),
		Source:   source,
		Content:  strings.TrimSpace(content),
	}
}

func normalizeAISkillRelativePath(raw string) (string, error) {
	value := filepath.ToSlash(strings.TrimSpace(raw))
	value = strings.TrimPrefix(value, "/")
	if value == "" {
		return "", errors.New("skill path 涓嶈兘涓虹┖")
	}

	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == "" {
		return "", errors.New("skill path 鏃犳晥")
	}
	if strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", errors.New("skill path 瓒呭嚭鍏佽鑼冨洿")
	}
	if strings.ToLower(filepath.Ext(clean)) != ".md" {
		return "", errors.New("skill 鏂囦欢蹇呴』鏄?.md")
	}
	return clean, nil
}

func (s *Server) loadAISkillCatalogItem(relativePath string) (aiSkillCatalogItem, error) {
	if err := s.ensureAIDirs(); err != nil {
		return aiSkillCatalogItem{}, err
	}

	relativePath, err := normalizeAISkillRelativePath(relativePath)
	if err != nil {
		return aiSkillCatalogItem{}, err
	}

	if strings.HasPrefix(relativePath, "custom/") {
		skillID := strings.TrimSuffix(strings.TrimPrefix(relativePath, "custom/"), ".md")
		custom, err := s.loadCustomSkill(skillID)
		if err != nil {
			return aiSkillCatalogItem{}, err
		}
		return aiSkillCatalogItem{
			ID:       "custom/" + custom.ID,
			Scope:    custom.Scope,
			Title:    custom.Title,
			Tags:     append([]string(nil), custom.Tags...),
			Required: false,
			Path:     custom.Path,
			Source:   "custom",
			Content:  custom.Content,
		}, nil
	}

	fullPath := filepath.Join(s.aiSkillsDir, filepath.FromSlash(relativePath))
	cleanRoot := filepath.Clean(s.aiSkillsDir)
	cleanPath := filepath.Clean(fullPath)
	if cleanPath != cleanRoot && !strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator)) {
		return aiSkillCatalogItem{}, errors.New("skill path 瓒呭嚭鍏佽鑼冨洿")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return aiSkillCatalogItem{}, err
	}

	if spec, exists := defaultAISkillSpecMap()[relativePath]; exists {
		return buildSkillCatalogItem(spec, "builtin", string(data)), nil
	}

	customSpec := aiPromptSkillSpec{
		RelativePath: relativePath,
		Scope:        scopeFromRelativePath(relativePath),
		Title:        humanizeSkillTitle(relativePath),
		Tags:         []string{},
		Required:     false,
	}
	return buildSkillCatalogItem(customSpec, "custom", string(data)), nil
}

func (s *Server) saveAISkillContent(relativePath string, content string) (aiSkillCatalogItem, error) {
	if err := s.ensureAIDirs(); err != nil {
		return aiSkillCatalogItem{}, err
	}

	relativePath, err := normalizeAISkillRelativePath(relativePath)
	if err != nil {
		return aiSkillCatalogItem{}, err
	}

	if strings.HasPrefix(relativePath, "custom/") {
		skillID := strings.TrimSuffix(strings.TrimPrefix(relativePath, "custom/"), ".md")
		custom, err := s.loadCustomSkill(skillID)
		if err != nil {
			return aiSkillCatalogItem{}, err
		}
		if _, err := s.saveCustomSkill(aiCustomSkillUpsertRequest{
			ID:         custom.ID,
			Title:      custom.Title,
			Scope:      custom.Scope,
			Tags:       append([]string(nil), custom.Tags...),
			AutoAttach: custom.AutoAttach,
			Content:    content,
		}); err != nil {
			return aiSkillCatalogItem{}, err
		}
		return s.loadAISkillCatalogItem(relativePath)
	}

	fullPath := filepath.Join(s.aiSkillsDir, filepath.FromSlash(relativePath))
	cleanRoot := filepath.Clean(s.aiSkillsDir)
	cleanPath := filepath.Clean(fullPath)
	if cleanPath != cleanRoot && !strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator)) {
		return aiSkillCatalogItem{}, errors.New("skill path 瓒呭嚭鍏佽鑼冨洿")
	}
	if _, err := os.Stat(fullPath); err != nil {
		return aiSkillCatalogItem{}, err
	}
	if err := os.WriteFile(fullPath, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
		return aiSkillCatalogItem{}, err
	}
	return s.loadAISkillCatalogItem(relativePath)
}

func (s *Server) listAISkillCatalog() ([]aiSkillCatalogItem, error) {
	if err := s.ensureAIDirs(); err != nil {
		return nil, err
	}

	specMap := defaultAISkillSpecMap()
	items := make([]aiSkillCatalogItem, 0, len(specMap))
	seen := make(map[string]struct{}, len(specMap))
	customRoot := filepath.Clean(s.customSkillsDir())

	err := filepath.WalkDir(s.aiSkillsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			return nil
		}
		if strings.HasPrefix(filepath.Clean(path), customRoot) {
			return nil
		}

		relativePath, err := filepath.Rel(s.aiSkillsDir, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if spec, exists := specMap[relativePath]; exists {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			items = append(items, buildSkillCatalogItem(spec, "builtin", string(data)))
			seen[relativePath] = struct{}{}
			return nil
		}

		customSpec := aiPromptSkillSpec{
			RelativePath: relativePath,
			Scope:        scopeFromRelativePath(relativePath),
			Title:        humanizeSkillTitle(relativePath),
			Tags:         []string{},
			Required:     false,
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		items = append(items, buildSkillCatalogItem(customSpec, "custom", string(data)))
		seen[relativePath] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}

	customSkills, err := s.listCustomAISkills()
	if err != nil {
		return nil, err
	}
	for _, custom := range customSkills {
		items = append(items, aiSkillCatalogItem{
			ID:       "custom/" + custom.ID,
			Scope:    custom.Scope,
			Title:    custom.Title,
			Tags:     append([]string(nil), custom.Tags...),
			Required: false,
			Path:     custom.Path,
			Source:   "custom",
			Content:  custom.Content,
		})
	}

	for relativePath, spec := range specMap {
		if _, exists := seen[relativePath]; exists {
			continue
		}
		items = append(items, buildSkillCatalogItem(spec, "builtin", spec.DefaultBody))
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Scope == items[j].Scope {
			return items[i].Path < items[j].Path
		}
		return items[i].Scope < items[j].Scope
	})
	return items, nil
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func isTemplateEntrySkillID(raw string) bool {
	switch normalizePromptSkillToken(raw) {
	case "template":
		return true
	default:
		return false
	}
}

var templateIntentKeywords = []string{
	"模板", "生成模板", "改模板", "改写模板", "template", ".tmpl", "生成草稿", "draft",
	"安装脚本", "部署脚本", "配置文件模板", "systemd", "unit file", "install.sh",
	"start.sh", "stop.sh", "check.sh", "config patch", "serviceTop", "serverConfig",
}

func (s *Server) listTemplateServiceNames() []string {
	entries, err := os.ReadDir(s.templatesDir)
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(entry.Name()))
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func detectTemplateIntent(message string, selectedDrafts []aiDraftFile, serviceNames []string) (bool, []string) {
	reasons := make([]string, 0)
	lower := strings.ToLower(message)
	for _, keyword := range templateIntentKeywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			reasons = append(reasons, "keyword:"+keyword)
		}
	}
	if len(selectedDrafts) > 0 {
		reasons = append(reasons, "selectedDrafts")
	}
	if len(reasons) == 0 {
		return false, []string{}
	}
	return true, uniqueSortedStrings(reasons)
}

func shouldUseTemplateSkillMode(req aiSessionMessageRequest, selectedDrafts []aiDraftFile, attachmentIDs []string, serviceNames []string) (bool, []string) {
	reasons := make([]string, 0, 6)
	if ok, intentReasons := detectTemplateIntent(req.Message, selectedDrafts, serviceNames); ok {
		reasons = append(reasons, intentReasons...)
	}
	if len(req.SelectedSkillIDs) > 0 {
		reasons = append(reasons, "explicit-skills")
	}
	if len(selectedDrafts) > 0 {
		reasons = append(reasons, "selectedDrafts")
	}
	if len(attachmentIDs) > 0 {
		reasons = append(reasons, "attachments")
	}
	if strings.TrimSpace(req.SessionRules) != "" {
		reasons = append(reasons, "sessionRules")
	}
	if len(reasons) == 0 {
		return false, []string{}
	}
	return true, uniqueSortedStrings(reasons)
}

func normalizePromptSkillToken(raw string) string {
	token := strings.TrimSpace(strings.ToLower(raw))
	if token == "" {
		return ""
	}
	token = strings.Trim(token, " \t\r\n\"'`[](){}<>.,;")
	token = strings.ReplaceAll(token, "\\", "/")
	token = strings.TrimPrefix(token, "./")
	token = strings.TrimPrefix(token, "/")
	token = strings.TrimPrefix(token, "data/ai/skills/")
	token = strings.TrimPrefix(token, "skills/")
	token = strings.TrimPrefix(token, "builtin/")
	return strings.TrimSpace(token)
}

func extractInlineSkillIDs(message string) []string {
	if strings.TrimSpace(message) == "" {
		return []string{}
	}
	collected := make([]string, 0)

	for _, token := range strings.Fields(message) {
		lowerToken := strings.ToLower(strings.TrimSpace(token))
		if strings.HasPrefix(lowerToken, "#skill:") {
			collected = append(collected, token[len("#skill:"):])
			continue
		}
		if strings.HasPrefix(lowerToken, "#") {
			alias := normalizePromptSkillToken(strings.TrimPrefix(token, "#"))
			if isTemplateEntrySkillID(alias) {
				collected = append(collected, alias)
			}
		}
	}

	lowerMessage := strings.ToLower(message)
	for offset := 0; offset < len(lowerMessage); {
		idx := strings.Index(lowerMessage[offset:], "@skill(")
		if idx < 0 {
			break
		}
		start := offset + idx + len("@skill(")
		endRel := strings.Index(lowerMessage[start:], ")")
		if endRel < 0 {
			break
		}
		segment := message[start : start+endRel]
		for _, part := range strings.Split(segment, ",") {
			collected = append(collected, part)
		}
		offset = start + endRel + 1
	}

	normalized := make([]string, 0, len(collected))
	seen := make(map[string]struct{}, len(collected))
	for _, skillID := range collected {
		id := normalizePromptSkillToken(skillID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}

func resolveBuiltinPromptSkillSpec(skillID string) (aiPromptSkillSpec, bool) {
	specMap := defaultAISkillSpecMap()
	normalized := normalizePromptSkillToken(skillID)
	if normalized == "" || strings.HasPrefix(normalized, "custom/") {
		return aiPromptSkillSpec{}, false
	}
	if isTemplateEntrySkillID(normalized) {
		return aiPromptSkillSpec{}, false
	}
	candidates := []string{normalized}
	if strings.HasSuffix(normalized, ".md") {
		candidates = append(candidates, strings.TrimSuffix(normalized, ".md"))
	} else {
		candidates = append(candidates, normalized+".md")
	}
	for _, candidate := range candidates {
		clean := filepath.ToSlash(strings.TrimSpace(candidate))
		if clean == "" {
			continue
		}
		if !strings.HasSuffix(clean, ".md") {
			clean += ".md"
		}
		if spec, ok := specMap[clean]; ok {
			return spec, true
		}
	}
	return aiPromptSkillSpec{}, false
}

func (s *Server) selectExplicitPromptSkills(skillIDs []string) ([]aiPromptSkill, error) {
	normalized := make([]string, 0, len(skillIDs))
	seen := make(map[string]struct{}, len(skillIDs))
	for _, raw := range skillIDs {
		id := normalizePromptSkillToken(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}

	selected := make([]aiPromptSkill, 0, len(normalized))
	for _, skillID := range normalized {
		if isTemplateEntrySkillID(skillID) {
			continue
		}
		if spec, ok := resolveBuiltinPromptSkillSpec(skillID); ok {
			skill, err := s.loadPromptSkill(spec)
			if err != nil {
				return nil, err
			}
			selected = append(selected, skill)
			continue
		}

		customID := strings.TrimPrefix(skillID, "custom/")
		customID = normalizeCustomSkillID(customID)
		if customID == "" || strings.Contains(customID, "/") {
			fmt.Printf("[AI] explicit skill ignored id=%s reason=unknown-skill\n", skillID)
			continue
		}
		if err := validateCustomSkillID(customID); err != nil {
			fmt.Printf("[AI] explicit skill ignored id=%s reason=invalid-custom-id err=%v\n", skillID, err)
			continue
		}

		custom, err := s.loadCustomSkill(customID)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("[AI] explicit skill ignored id=%s reason=not-found\n", skillID)
				continue
			}
			return nil, err
		}
		selected = append(selected, aiPromptSkill{
			Ref: aiSkillRef{
				ID:       "custom/" + custom.ID,
				Scope:    custom.Scope,
				Title:    custom.Title,
				Tags:     append([]string(nil), custom.Tags...),
				Required: false,
			},
			Path:    filepath.Join(s.customSkillsDir(), custom.ID+".md"),
			Content: custom.Content,
		})
	}

	sort.SliceStable(selected, func(i, j int) bool {
		if selected[i].Ref.Scope == selected[j].Ref.Scope {
			return selected[i].Ref.ID < selected[j].Ref.ID
		}
		return selected[i].Ref.Scope < selected[j].Ref.Scope
	})
	return selected, nil
}

func (s *Server) detectPromptKinds(message string, selectedDrafts []aiDraftFile, attachmentIDs []string, session *aiSession, templateIntent bool) map[string]bool {
	kinds := map[string]bool{
		"install_script":  false,
		"config_file":     false,
		"systemd_service": false,
		"cluster_service": false,
		"image_rewrite":   false,
		"config_patch":    false,
	}

	messageLower := strings.ToLower(message)
	if containsAny(messageLower, "install", "setup", "deploy", "start", "stop", "check", ".sh.tmpl", "脚本", "安装", "启动", "停止", "检查") {
		kinds["install_script"] = true
		kinds["config_patch"] = true
	}
	if containsAny(messageLower, ".conf", ".yaml", ".yml", ".xml", ".json", ".properties", ".service", "配置", "config", "service file") {
		kinds["config_file"] = true
		kinds["config_patch"] = true
	}
	if containsAny(messageLower, "systemd", ".service", "unit file", "服务单元") {
		kinds["systemd_service"] = true
		kinds["config_file"] = true
		kinds["config_patch"] = true
	}
	if containsAny(messageLower, "cluster", "ha", "多节点", "3节点", "三节点", "副本") {
		kinds["cluster_service"] = true
		kinds["config_patch"] = true
	}
	if containsAny(messageLower, "vars", "变量", "patch", "同步", "servicetop", "serverconfig") {
		kinds["config_patch"] = true
	}
	if templateIntent {
		kinds["config_patch"] = true
	}

	for _, draft := range selectedDrafts {
		lowerPath := strings.ToLower(filepath.ToSlash(draft.Path))
		switch detectExampleKind(lowerPath) {
		case "shell_script":
			kinds["install_script"] = true
			kinds["config_patch"] = true
		case "config_file":
			kinds["config_file"] = true
			kinds["config_patch"] = true
		case "systemd_service":
			kinds["systemd_service"] = true
			kinds["config_file"] = true
			kinds["config_patch"] = true
		}
		if looksClusterTemplate(draft.Content, lowerPath) {
			kinds["cluster_service"] = true
			kinds["config_patch"] = true
		}
	}

	for _, attachmentID := range attachmentIDs {
		attachment, err := s.findAttachment(session, attachmentID)
		if err != nil {
			continue
		}
		if attachment.Kind == "image" {
			kinds["image_rewrite"] = true
		}
	}

	return kinds
}
func (s *Server) selectPromptSkills(session *aiSession, req aiSessionMessageRequest, selectedDrafts []aiDraftFile, attachmentIDs []string) ([]aiPromptSkill, error) {
	if err := s.ensureAIDirs(); err != nil {
		return nil, err
	}

	inlineSkillIDs := extractInlineSkillIDs(req.Message)
	explicitSkillIDs := make([]string, 0, len(req.SelectedSkillIDs)+len(inlineSkillIDs))
	explicitSkillIDs = append(explicitSkillIDs, req.SelectedSkillIDs...)
	explicitSkillIDs = append(explicitSkillIDs, inlineSkillIDs...)
	req.SelectedSkillIDs = explicitSkillIDs

	templateMode, modeReasons := shouldUseTemplateSkillMode(req, selectedDrafts, attachmentIDs, s.listTemplateServiceNames())
	hasExplicitSkill := len(explicitSkillIDs) > 0

	specMap := defaultAISkillSpecMap()
	selectedSpecs := []aiPromptSkillSpec{}
	if templateMode {
		selectedSpecs = append(selectedSpecs,
			specMap["core/output_json.md"],
			specMap["core/base.md"],
			specMap["core/path_guard.md"],
			specMap["core/template_contract.md"],
			specMap["core/variable_contract.md"],
			specMap["core/template_functions.md"],
		)
	}
	fmt.Printf("[AI] skill trigger templateMode=%t explicit=%t reasons=%s\n",
		templateMode, hasExplicitSkill, summarizePathsForLog(modeReasons, 12))
	if !templateMode && !hasExplicitSkill {
		return []aiPromptSkill{}, nil
	}

	kinds := s.detectPromptKinds(req.Message, selectedDrafts, attachmentIDs, session, templateMode)
	if kinds["install_script"] {
		selectedSpecs = append(selectedSpecs, specMap["task/install_script.md"])
	}
	if kinds["config_file"] {
		selectedSpecs = append(selectedSpecs, specMap["task/config_file.md"])
	}
	if kinds["systemd_service"] {
		selectedSpecs = append(selectedSpecs, specMap["task/systemd_service.md"])
	}
	if kinds["cluster_service"] {
		selectedSpecs = append(selectedSpecs, specMap["task/cluster_service.md"])
	}
	if kinds["image_rewrite"] {
		selectedSpecs = append(selectedSpecs, specMap["task/image_rewrite.md"])
	}
	if kinds["config_patch"] {
		selectedSpecs = append(selectedSpecs, specMap["task/config_patch.md"])
	}

	dedup := make(map[string]struct{}, len(selectedSpecs))
	skills := make([]aiPromptSkill, 0, len(selectedSpecs))
	for _, spec := range selectedSpecs {
		relativePath := filepath.ToSlash(spec.RelativePath)
		if _, exists := dedup[relativePath]; exists {
			continue
		}
		dedup[relativePath] = struct{}{}

		skill, err := s.loadPromptSkill(spec)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}

	sort.SliceStable(skills, func(i, j int) bool {
		if skills[i].Ref.Scope == skills[j].Ref.Scope {
			return skills[i].Ref.ID < skills[j].Ref.ID
		}
		return skills[i].Ref.Scope < skills[j].Ref.Scope
	})

	customSkills, err := s.selectMatchingCustomPromptSkills(req.Message, selectedDrafts)
	if err != nil {
		return nil, err
	}
	for _, customSkill := range customSkills {
		duplicate := false
		for _, existing := range skills {
			if existing.Ref.ID == customSkill.Ref.ID {
				duplicate = true
				break
			}
		}
		if !duplicate {
			skills = append(skills, customSkill)
		}
	}

	explicitSkills, err := s.selectExplicitPromptSkills(explicitSkillIDs)
	if err != nil {
		return nil, err
	}
	for _, explicitSkill := range explicitSkills {
		duplicate := false
		for _, existing := range skills {
			if existing.Ref.ID == explicitSkill.Ref.ID {
				duplicate = true
				break
			}
		}
		if !duplicate {
			skills = append(skills, explicitSkill)
		}
	}

	sort.SliceStable(skills, func(i, j int) bool {
		if skills[i].Ref.Scope == skills[j].Ref.Scope {
			return skills[i].Ref.ID < skills[j].Ref.ID
		}
		return skills[i].Ref.Scope < skills[j].Ref.Scope
	})
	finalIDs := make([]string, 0, len(skills))
	for _, skill := range skills {
		finalIDs = append(finalIDs, skill.Ref.ID)
	}
	fmt.Printf("[AI] prompt skills resolved autoMatched=%d explicitRequested=%s final=%s\n",
		len(skills)-len(explicitSkills),
		summarizePathsForLog(explicitSkillIDs, 12),
		summarizePathsForLog(finalIDs, 24))
	return skills, nil
}

func (s *Server) buildAIPromptBundle(session *aiSession, req aiSessionMessageRequest, selectedDrafts []aiDraftFile, attachmentIDs []string) (aiPromptBundle, error) {
	customRules, err := s.loadAIRules()
	if err != nil {
		return aiPromptBundle{}, fmt.Errorf("load AI rules failed: %w", err)
	}

	skills, err := s.selectPromptSkills(session, req, selectedDrafts, attachmentIDs)
	if err != nil {
		return aiPromptBundle{}, fmt.Errorf("select prompt skills failed: %w", err)
	}

	trace := emptyAIPromptTrace()
	if len(skills) == 0 {
		trace.Mode = "general-chat"
		builder := strings.Builder{}
		builder.WriteString("你是该内部工具的中文技术助手。直接回答用户问题。\n")
		builder.WriteString("只有当用户明确要求生成/修改模板、显式调用 skill、选择草稿或上传附件时，才进入模板草稿模式。\n")
		builder.WriteString("不要虚构已经生成模板、已经修改文件、已经保存成功。\n")
		if strings.TrimSpace(customRules) != "" {
			builder.WriteString("\n[Global Extra Rules]\n")
			builder.WriteString(strings.TrimSpace(customRules))
			builder.WriteString("\n")
		}
		return aiPromptBundle{
			SystemPrompt: strings.TrimSpace(builder.String()),
			PromptTrace:  normalizeAIPromptTrace(trace),
			Structured:   false,
		}, nil
	}

	builder := strings.Builder{}
	builder.WriteString("Follow the selected skill modules and produce one valid JSON response.\n")

	for _, skill := range skills {
		trace.SkillRefs = append(trace.SkillRefs, skill.Ref)
		builder.WriteString("\n[Skill] ")
		builder.WriteString(skill.Ref.Title)
		builder.WriteString(" (")
		builder.WriteString(skill.Ref.ID)
		builder.WriteString(")\n")
		builder.WriteString(skill.Content)
		builder.WriteString("\n")
	}

	if strings.TrimSpace(customRules) != "" {
		builder.WriteString("\n[Global Extra Rules]\n")
		builder.WriteString(strings.TrimSpace(customRules))
		builder.WriteString("\n")
	}
	if strings.TrimSpace(req.SessionRules) != "" {
		builder.WriteString("\n[Session Extra Rules]\n")
		builder.WriteString(strings.TrimSpace(req.SessionRules))
		builder.WriteString("\n")
	}

	if configContext := buildConfigPromptContext(s.cfg); strings.TrimSpace(configContext) != "" {
		builder.WriteString("\n[Current Config Context]\n")
		builder.WriteString(configContext)
		builder.WriteString("\n")
	}

	inlineSkillIDs := extractInlineSkillIDs(req.Message)
	templateIntent, _ := detectTemplateIntent(req.Message, selectedDrafts, s.listTemplateServiceNames())
	includeExamples := len(req.SelectedSkillIDs) > 0 || len(inlineSkillIDs) > 0 || templateIntent
	if includeExamples {
		examples, selectErr := s.selectPromptExamples(req.Message, selectedDrafts, skills)
		if selectErr != nil {
			return aiPromptBundle{}, fmt.Errorf("select prompt examples failed: %w", selectErr)
		}
		if len(examples) > 0 {
			builder.WriteString("\n[Reference Examples]\nUse style/structure for reference only, do not copy blindly.\n")
			for _, example := range examples {
				trace.ExampleRefs = append(trace.ExampleRefs, example.Ref)
				builder.WriteString("\nExample Path: ")
				builder.WriteString(example.Ref.Path)
				if example.Ref.Reason != "" {
					builder.WriteString("\nReason: ")
					builder.WriteString(example.Ref.Reason)
				}
				builder.WriteString("\nSnippet:\n")
				builder.WriteString(example.Snippet)
				builder.WriteString("\n")
			}
		}
	}

	builder.WriteString("\n[Template Function Whitelist]\nUse project-registered helpers only: ")
	builder.WriteString(templateFunctionWhitelistSummary())
	builder.WriteString(". Go-template builtins are allowed. If extra helper is needed, rewrite and explain in warnings.\n")

	builder.WriteString("\n[Output Requirement]\nReturn configPatch for serviceTop/serverConfig when possible; if info is missing, ask in followUpQuestions.\n")
	return aiPromptBundle{
		SystemPrompt: strings.TrimSpace(builder.String()),
		PromptTrace:  normalizeAIPromptTrace(trace),
		Structured:   true,
	}, nil
}

func formatPromptTraceSummary(trace aiPromptTrace) string {
	trace = normalizeAIPromptTrace(trace)
	parts := make([]string, 0, len(trace.SkillRefs)+len(trace.ExampleRefs))
	if len(trace.SkillRefs) > 0 {
		skillIDs := make([]string, 0, len(trace.SkillRefs))
		for _, skill := range trace.SkillRefs {
			skillIDs = append(skillIDs, skill.ID)
		}
		parts = append(parts, "Skills: "+strings.Join(skillIDs, ", "))
	}
	if len(trace.ExampleRefs) > 0 {
		paths := make([]string, 0, len(trace.ExampleRefs))
		for _, example := range trace.ExampleRefs {
			paths = append(paths, example.Path)
		}
		parts = append(parts, "Examples: "+strings.Join(paths, ", "))
	}
	return strings.Join(parts, "\n")
}

func summarizePromptTrace(trace aiPromptTrace) string {
	summary := formatPromptTraceSummary(trace)
	if strings.TrimSpace(summary) == "" {
		return ""
	}
	return "\n\nPromptTrace:\n" + summary
}
