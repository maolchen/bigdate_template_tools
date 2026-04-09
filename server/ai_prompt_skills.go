package server

import (
	"errors"
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
			Title:        "基础项目约束",
			Tags:         []string{"global", "idempotent"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
你正在为本项目生成可审核、可保存、可渲染的 templates/**/*.tmpl 草稿。

强约束：
- 路径、用户、组、JAVA_HOME 优先复用 .Global，不要重复定义 run_user、run_group、java_home、install_base_dir、data_base_dir。
- shell 脚本必须优先考虑幂等性：mkdir -p、软链接先删后建、覆盖前备份、重复执行不应产生脏状态。
- 涉及 root 权限的操作通过 run_as_root 包装；非必要不要直接使用 root 运行服务。
- 模板上下文优先使用 .Global、.Nodes、.Instance、.AllInstances。
- 优先使用已有模板函数：serviceNodes、serviceEndpoints、serviceEndpointsJoin、serviceIPs、serviceHostnames、getServiceNodes、serviceVars、serviceVar、serviceConfig、nodeInfo。
- 你只能输出草稿计划，不能假设文件已经真正写入。`),
		},
		{
			RelativePath: "core/output_json.md",
			Scope:        "core",
			Title:        "结构化输出",
			Tags:         []string{"json", "output"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
响应必须是单个 JSON 对象，字段固定为：
- assistantMessage
- draftFiles[]: path、content、reason
- plannedActions[]: type、path、reason
- warnings[]
- followUpQuestions[]
- configPatch: serviceTop、serverConfig

如果信息不足，不要伪造配置，把问题写到 followUpQuestions。`),
		},
		{
			RelativePath: "core/path_guard.md",
			Scope:        "core",
			Title:        "模板路径限制",
			Tags:         []string{"path", "templates"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
路径约束：
- 模板文件路径必须位于 templates/<service>/... 下。
- 模板文件必须以 .tmpl 结尾。
- 新服务可以先规划 mkdir 动作，再生成具体草稿文件。
- 目录计划和文件计划都只能落在 templates/ 下。`),
		},
		{
			RelativePath: "task/install_script.md",
			Scope:        "task",
			Title:        "脚本模板规则",
			Tags:         []string{"shell", "install"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
安装/脚本模板规则：
- install.sh.tmpl、install_binary.sh.tmpl、setup_dirs.sh.tmpl、start.sh.tmpl、stop.sh.tmpl、check.sh.tmpl 都属于脚本类模板。
- 脚本里优先集中定义变量区，再定义辅助函数，再执行安装/配置/启动步骤。
- 涉及数据目录、安装目录、软件包目录时，优先分别使用：
  - {{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}
  - {{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}
  - {{ .Global.software_dir }}/{{ .Instance.Vars.package_subdir }}
- 推荐包含 log_info、log_warn、log_error、log_success 这类基础日志函数。
- 如果脚本引用了 .Instance.Vars.xxx，必须同步给出 serverConfig.vars.xxx 草案。`),
		},
		{
			RelativePath: "task/config_file.md",
			Scope:        "task",
			Title:        "配置模板规则",
			Tags:         []string{"config"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
配置文件模板规则：
- 优先把环境相关值参数化成 .Global 或 .Instance.Vars，不要硬编码 IP、主机名、目录、用户、JDK 路径。
- 配置模板命名尽量对齐真实文件名，例如 my.cnf.tmpl、dnsmasq.conf.tmpl、docker.service.tmpl。
- 允许保留服务本身要求的固定键名，但取值应参数化。
- 如果某些固定路径不可避免，必须在 reason 或 warnings 中说明原因。`),
		},
		{
			RelativePath: "task/systemd_service.md",
			Scope:        "task",
			Title:        "systemd 规则",
			Tags:         []string{"systemd", "service"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
systemd 模板规则：
- systemd unit 文件通常属于配置模板，不要把整套安装逻辑写进 unit。
- User 和 Group 优先使用 {{ .Global.user }} 和 {{ .Global.group }}。
- 如果 unit 文件需要引用脚本或目录，优先使用全局目录和实例变量拼接。
- 涉及 /etc/systemd/system 这类固定系统路径时，要在说明里标记为系统目录约束。`),
		},
		{
			RelativePath: "task/cluster_service.md",
			Scope:        "task",
			Title:        "集群模板规则",
			Tags:         []string{"cluster"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
集群模板规则：
- 不允许新增节点；serviceTop.nodes 只能从当前已配置节点别名中选择，或使用 ["*"]。
- 需要节点列表时，优先使用模板函数和当前上下文，不要手工硬编码节点集合。
- 如果用户要求 3 节点或多节点，但当前配置节点数量不足，必须在 followUpQuestions 里要求确认，不要编造节点。
- 如果需要实例唯一标识，优先考虑 {{ .Instance.AutoID }}。`),
		},
		{
			RelativePath: "task/image_rewrite.md",
			Scope:        "task",
			Title:        "图片改写规则",
			Tags:         []string{"image", "ocr"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
图片改写规则：
- 图片只用于识别文字截图、脚本截图、配置截图、文档拍照。
- 先提取可见文字，再按本项目规则改写成模板草稿。
- 对识别不清、被裁切、被遮挡、低分辨率、含糊字段，必须写入 warnings。
- 图片来源的草稿要保守处理，宁可提示人工补齐，也不要伪造细节。`),
		},
		{
			RelativePath: "task/config_patch.md",
			Scope:        "task",
			Title:        "配置同步规则",
			Tags:         []string{"configPatch"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
配置同步规则：
- 新服务模板通常需要同时给出 configPatch.serviceTop 和 configPatch.serverConfig。
- 如果模板引用了 .Instance.Vars.xxx，就必须补对应的 serverConfig.vars.xxx。
- 不要把与模板无关的变量塞进 vars。
- 已有服务如果只是补模板，优先补齐缺失项，不要无理由覆盖现有配置。`),
		},
		{
			RelativePath: "service/docker.md",
			Scope:        "service",
			Title:        "DOCKER 特殊规则",
			Tags:         []string{"docker"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
Docker 特殊规则：
- 允许涉及 /etc/docker、/usr/lib/systemd/system、docker.service 这类系统路径。
- 但仍需优先复用全局用户、组和软件目录，并明确说明哪些路径是 Docker 的固定系统约束。`),
		},
		{
			RelativePath: "service/mysql.md",
			Scope:        "service",
			Title:        "MYSQL 特殊规则",
			Tags:         []string{"mysql"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
MySQL 特殊规则：
- 配置文件通常命名为 my.cnf.tmpl。
- 数据目录、日志目录优先参数化为 .Instance.Vars，再与 .Global.data_base_dir 拼接。
- 如果需要初始化库或用户，优先拆分成额外脚本模板，不要把所有逻辑塞进一个大脚本。`),
		},
		{
			RelativePath: "service/kerberos.md",
			Scope:        "service",
			Title:        "KERBEROS 特殊规则",
			Tags:         []string{"kerberos"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
Kerberos 特殊规则：
- krb5.conf、kdc.conf 等配置项较多时，优先保持结构清晰，减少无关注释。
- realm、域名、主机名这类值不要编造，信息不足时必须追问。`),
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
		return "", errors.New("skill path 不能为空")
	}

	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == "" {
		return "", errors.New("skill path 无效")
	}
	if strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", errors.New("skill path 超出允许范围")
	}
	if strings.ToLower(filepath.Ext(clean)) != ".md" {
		return "", errors.New("skill 文件必须是 .md")
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
		return aiSkillCatalogItem{}, errors.New("skill path 超出允许范围")
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
		return aiSkillCatalogItem{}, errors.New("skill path 超出允许范围")
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

func (s *Server) detectPromptKinds(message string, selectedDrafts []aiDraftFile, attachmentIDs []string, session *aiSession) map[string]bool {
	kinds := map[string]bool{
		"install_script":  false,
		"config_file":     false,
		"systemd_service": false,
		"cluster_service": false,
		"image_rewrite":   false,
		"config_patch":    true,
	}

	messageLower := strings.ToLower(message)
	if containsAny(messageLower, "install", "setup", "deploy", "start", "stop", "check", ".sh.tmpl", "安装", "部署", "启动", "停止", "检查") {
		kinds["install_script"] = true
	}
	if containsAny(messageLower, ".conf", ".yaml", ".yml", ".xml", ".json", ".properties", ".service", "配置", "config", "service 文件") {
		kinds["config_file"] = true
	}
	if containsAny(messageLower, "systemd", ".service", "unit 文件", "服务单元") {
		kinds["systemd_service"] = true
		kinds["config_file"] = true
	}
	if containsAny(messageLower, "cluster", "集群", "ha", "多节点", "3节点", "三节点", "多副本") {
		kinds["cluster_service"] = true
	}

	for _, draft := range selectedDrafts {
		lowerPath := strings.ToLower(filepath.ToSlash(draft.Path))
		switch detectExampleKind(lowerPath) {
		case "shell_script":
			kinds["install_script"] = true
		case "config_file":
			kinds["config_file"] = true
		case "systemd_service":
			kinds["systemd_service"] = true
			kinds["config_file"] = true
		}
		if looksClusterTemplate(draft.Content, lowerPath) {
			kinds["cluster_service"] = true
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

func detectSpecialServices(message string, selectedDrafts []aiDraftFile) map[string]bool {
	services := make(map[string]bool)
	messageLower := strings.ToLower(message)
	for _, special := range []string{"docker", "mysql", "kerberos"} {
		if strings.Contains(messageLower, special) {
			services[special] = true
		}
	}
	for _, draft := range selectedDrafts {
		serviceName := strings.ToLower(serviceNameFromTemplatePath(draft.Path))
		if serviceName == "" {
			continue
		}
		if serviceName == "docker" || serviceName == "mysql" || serviceName == "kerberos" {
			services[serviceName] = true
		}
	}
	return services
}

func (s *Server) selectPromptSkills(session *aiSession, req aiSessionMessageRequest, selectedDrafts []aiDraftFile, attachmentIDs []string) ([]aiPromptSkill, error) {
	if err := s.ensureAIDirs(); err != nil {
		return nil, err
	}

	selectedSpecs := []aiPromptSkillSpec{
		defaultAISkillSpecMap()["core/base.md"],
		defaultAISkillSpecMap()["core/output_json.md"],
		defaultAISkillSpecMap()["core/path_guard.md"],
	}

	kinds := s.detectPromptKinds(req.Message, selectedDrafts, attachmentIDs, session)
	if kinds["install_script"] {
		selectedSpecs = append(selectedSpecs, defaultAISkillSpecMap()["task/install_script.md"])
	}
	if kinds["config_file"] {
		selectedSpecs = append(selectedSpecs, defaultAISkillSpecMap()["task/config_file.md"])
	}
	if kinds["systemd_service"] {
		selectedSpecs = append(selectedSpecs, defaultAISkillSpecMap()["task/systemd_service.md"])
	}
	if kinds["cluster_service"] {
		selectedSpecs = append(selectedSpecs, defaultAISkillSpecMap()["task/cluster_service.md"])
	}
	if kinds["image_rewrite"] {
		selectedSpecs = append(selectedSpecs, defaultAISkillSpecMap()["task/image_rewrite.md"])
	}
	if kinds["config_patch"] {
		selectedSpecs = append(selectedSpecs, defaultAISkillSpecMap()["task/config_patch.md"])
	}

	for serviceName := range detectSpecialServices(req.Message, selectedDrafts) {
		if spec, exists := defaultAISkillSpecMap()["service/"+serviceName+".md"]; exists {
			selectedSpecs = append(selectedSpecs, spec)
		}
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

	explicitSkills, err := s.selectExplicitCustomPromptSkills(req.SelectedSkillIDs)
	if err != nil {
		return nil, err
	}
	for _, customSkill := range explicitSkills {
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

	sort.SliceStable(skills, func(i, j int) bool {
		if skills[i].Ref.Scope == skills[j].Ref.Scope {
			return skills[i].Ref.ID < skills[j].Ref.ID
		}
		return skills[i].Ref.Scope < skills[j].Ref.Scope
	})
	return skills, nil
}

func (s *Server) buildAIPromptBundle(session *aiSession, req aiSessionMessageRequest, selectedDrafts []aiDraftFile, attachmentIDs []string) (string, aiPromptTrace, error) {
	customRules, err := s.loadAIRules()
	if err != nil {
		return "", aiPromptTrace{}, err
	}

	skills, err := s.selectPromptSkills(session, req, selectedDrafts, attachmentIDs)
	if err != nil {
		return "", aiPromptTrace{}, err
	}

	examples, err := s.selectPromptExamples(req.Message, selectedDrafts, skills)
	if err != nil {
		return "", aiPromptTrace{}, err
	}

	trace := emptyAIPromptTrace()
	builder := strings.Builder{}
	builder.WriteString("请同时遵守以下规则模块，生成适合本项目的模板草稿。\n")

	for _, skill := range skills {
		trace.SkillRefs = append(trace.SkillRefs, skill.Ref)
		builder.WriteString("\n[规则模块] ")
		builder.WriteString(skill.Ref.Title)
		builder.WriteString(" (")
		builder.WriteString(skill.Ref.ID)
		builder.WriteString(")\n")
		builder.WriteString(skill.Content)
		builder.WriteString("\n")
	}

	if strings.TrimSpace(customRules) != "" {
		builder.WriteString("\n[自定义全局补充规则]\n")
		builder.WriteString(strings.TrimSpace(customRules))
		builder.WriteString("\n")
	}
	if strings.TrimSpace(req.SessionRules) != "" {
		builder.WriteString("\n[当前会话补充规则]\n")
		builder.WriteString(strings.TrimSpace(req.SessionRules))
		builder.WriteString("\n")
	}

	if configContext := buildConfigPromptContext(s.cfg); strings.TrimSpace(configContext) != "" {
		builder.WriteString("\n[当前配置上下文]\n")
		builder.WriteString(configContext)
		builder.WriteString("\n")
	}

	if len(examples) > 0 {
		builder.WriteString("\n[参考模板示例]\n以下示例只用于结构、风格和变量组织参考，不要机械复制；上面的规则优先级更高。\n")
		for _, example := range examples {
			trace.ExampleRefs = append(trace.ExampleRefs, example.Ref)
			builder.WriteString("\n示例路径: ")
			builder.WriteString(example.Ref.Path)
			if example.Ref.Reason != "" {
				builder.WriteString("\n选择原因: ")
				builder.WriteString(example.Ref.Reason)
			}
			builder.WriteString("\n示例片段:\n")
			builder.WriteString(example.Snippet)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n[模板函数白名单]\n")
	builder.WriteString("请仅使用项目已注册函数：")
	builder.WriteString(templateFunctionWhitelistSummary())
	builder.WriteString("。除此之外，只允许使用 Go Template 内建函数（如 and/or/not/len/index/printf 等）。若需要白名单外函数，请改写实现并在 warnings 中说明。\n")

	builder.WriteString("\n[输出要求补充]\n请尽量返回 configPatch，用于同步 serviceTop 和 serverConfig.vars；如果信息不足，请在 followUpQuestions 里明确提出。")
	return strings.TrimSpace(builder.String()), normalizeAIPromptTrace(trace), nil
}

func formatPromptTraceSummary(trace aiPromptTrace) string {
	trace = normalizeAIPromptTrace(trace)
	parts := make([]string, 0, len(trace.SkillRefs)+len(trace.ExampleRefs))
	if len(trace.SkillRefs) > 0 {
		skillIDs := make([]string, 0, len(trace.SkillRefs))
		for _, skill := range trace.SkillRefs {
			skillIDs = append(skillIDs, skill.ID)
		}
		parts = append(parts, "命中规则: "+strings.Join(skillIDs, ", "))
	}
	if len(trace.ExampleRefs) > 0 {
		paths := make([]string, 0, len(trace.ExampleRefs))
		for _, example := range trace.ExampleRefs {
			paths = append(paths, example.Path)
		}
		parts = append(parts, "参考示例: "+strings.Join(paths, ", "))
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
