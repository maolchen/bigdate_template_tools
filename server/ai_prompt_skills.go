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
			DefaultBody: strings.TrimSpace(`
浠呯敓鎴愬彲瀹¤鐨勬ā鏉胯崏绋?(auditable draft templates only)銆?- 浼樺厛浣跨敤 .Global 鎻愪緵 user/group/java/path 绛夊叏灞€鍊笺€?- Shell 鑴氭湰灏介噺淇濊瘉骞傜瓑 (idempotent where possible)銆?- 浠呭湪鐗规潈鎿嶄綔浣跨敤 run_as_root銆?- 浼樺厛澶嶇敤鐜版湁妯℃澘 helper 鍑芥暟銆?- 涓嶈澹扮О鏂囦欢鈥滃凡缁忓啓鍏ョ鐩樷€濄€?`),
		},
		{
			RelativePath: "core/output_json.md",
			Scope:        "core",
			Title:        "Structured JSON Output",
			Tags:         []string{"json", "output"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
璇峰彧杩斿洖涓€涓?JSON object锛屽繀椤诲寘鍚細
- assistantMessage
- draftFiles[]: path, content, reason
- plannedActions[]: type, path, reason
- warnings[]
- followUpQuestions[]
- configPatch: serviceTop, serverConfig
`),
		},
		{
			RelativePath: "core/path_guard.md",
			Scope:        "core",
			Title:        "Template Path Guard",
			Tags:         []string{"path", "templates"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
璺緞绾︽潫 (path constraints):
- 鎵€鏈?draftFiles 璺緞蹇呴』鍦?templates/<service>/...
- 妯℃澘鏂囦欢蹇呴』浠?.tmpl 缁撳熬
- plannedActions.path 蹇呴』浣嶄簬 templates/ 涓?`),
		},
		{
			RelativePath: "core/template_contract.md",
			Scope:        "core",
			Title:        "Template Contract",
			Tags:         []string{"contract", "hostname", "placeholder"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
妯℃澘濂戠害 (strict template contract):
- 鑺傜偣鍗犱綅绗﹀繀椤讳娇鐢?{{ .Instance.Node.Hostname }}銆?- 绂佹鍗犱綅绗︼細.Instance.NodeAlias, .Instance.Node.HostName銆?- 妯℃澘璺緞蹇呴』鍦?templates/<service>/ 涓嬶紝涓斿悗缂€涓?.tmpl銆?- 榛樿杈撳嚭缁撴瀯鏄?output/<ip>/<service>/ 骞抽摵鐩綍銆?- 闄ら潪鐢ㄦ埛鏄庣‘瑕佹眰锛屽惁鍒欎笉瑕佸垱寤?templates/<service>/config/...銆?- 鑻ユ棫鑽夌涓庡绾﹀啿绐侊紝浼樺厛閬靛惊鏈绾﹀苟鍦?warnings 璇存槑銆?`),
		},
		{
			RelativePath: "core/variable_contract.md",
			Scope:        "core",
			Title:        "Variable Contract",
			Tags:         []string{"variables", "context", "schema"},
			Required:     true,
			DefaultBody: strings.TrimSpace(`
妯℃澘涓婁笅鏂囧绾?(template context contract):
- .Global.<key>锛堟潵鑷?config.yaml global锛屽姩鎬?key锛?- .Nodes.<alias>.IP / .Nodes.<alias>.Hostname锛堝浐瀹氳妭鐐瑰瓧娈碉級
- .Instance.ServiceName / .Instance.NodeName / .Instance.AutoID
- .Instance.Node.IP / .Instance.Node.Hostname / .Instance.Node.NodeName
- .Instance.Vars.<key>锛堟潵鑷?serverConfig.<service>.vars锛屽姩鎬?key锛?- .AllInstances[]锛堝瓧娈电粨鏋勪笌 .Instance 涓€鑷达級

绂佹鍋囪 (forbidden assumptions):
- Do not use .Instance.NodeAlias
- Do not use .Instance.Node.HostName
- Do not invent .Instance.<unknownField>
`),
		},
		{
			RelativePath: "task/install_script.md",
			Scope:        "task",
			Title:        "Install Script Rules",
			Tags:         []string{"shell", "install"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
鐢ㄤ簬 install/start/stop/check 鑴氭湰锛?- 缁撴瀯娓呮櫚锛歷ariables/helpers/main flow銆?- 浼樺厛浣跨敤 .Global + .Instance.Vars 缁勫悎銆?- 鑻ヤ娇鐢?.Instance.Vars.xxx锛屽簲鎻愪緵鍖归厤鐨?serverConfig.vars 寤鸿銆?`),
		},
		{
			RelativePath: "task/config_file.md",
			Scope:        "task",
			Title:        "Config File Rules",
			Tags:         []string{"config"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
鐢ㄤ簬閰嶇疆鏂囦欢妯℃澘锛?- 鐜鐩稿叧鍊间紭鍏堝弬鏁板寲鍒?.Global 鍜?.Instance.Vars銆?- 鑳藉弬鏁板寲鐨?host/IP/user/path 涓嶈纭紪鐮併€?- 鏂囦欢鍛藉悕搴斾笌鐪熷疄閰嶇疆鏂囦欢鍚嶄繚鎸佷竴鑷淬€?`),
		},
		{
			RelativePath: "task/systemd_service.md",
			Scope:        "task",
			Title:        "Systemd Rules",
			Tags:         []string{"systemd", "service"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
鐢ㄤ簬 systemd unit 妯℃澘锛?- 浼樺厛浣跨敤 {{ .Global.user }} 涓?{{ .Global.group }}銆?- unit 鏂囦欢淇濇寔鑱氱劍锛屼笉瑕佸祵鍏ュ畬鏁村畨瑁呴€昏緫銆?`),
		},
		{
			RelativePath: "task/cluster_service.md",
			Scope:        "task",
			Title:        "Cluster Rules",
			Tags:         []string{"cluster"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
鐢ㄤ簬闆嗙兢鏈嶅姟妯℃澘锛?- 涓嶈缂栭€犻厤缃鐨?node aliases銆?- 鎷撴墤绾︽潫涓嶆槑纭椂鍏堟彁 follow-up questions銆?`),
		},
		{
			RelativePath: "task/image_rewrite.md",
			Scope:        "task",
			Title:        "Image Rewrite Rules",
			Tags:         []string{"image", "ocr"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
褰撴彁渚涘浘鐗囪緭鍏ユ椂锛?- 鍏堟彁鍙栧彲瑙佹枃瀛楋紝鍐嶉噸鍐欎负妯℃澘鑽夌鏍煎紡銆?- OCR 涓嶇‘瀹氭€ф垨鎴柇椋庨櫓鍐欏叆 warnings銆?`),
		},
		{
			RelativePath: "task/config_patch.md",
			Scope:        "task",
			Title:        "Config Patch Rules",
			Tags:         []string{"configPatch"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
褰撴ā鏉垮紩鍏ユ柊 vars 鏃讹細
- 杩斿洖鍖归厤鐨?configPatch.serverConfig vars 寤鸿銆?- 鏃犲厖鍒嗙悊鐢辨椂涓嶈瑕嗙洊鏃犲叧鐜版湁閰嶇疆銆?`),
		},
		{
			RelativePath: "service/docker.md",
			Scope:        "service",
			Title:        "Docker Rules",
			Tags:         []string{"docker"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
Docker 妯℃澘鍙秹鍙婄郴缁熺洰褰曡矾寰勶紙濡?/etc/docker锛夈€?浣?user/group/path 浠嶅簲浼樺厛澶嶇敤鍏ㄥ眬鍙橀噺 (.Global)銆?`),
		},
		{
			RelativePath: "service/mysql.md",
			Scope:        "service",
			Title:        "MySQL Rules",
			Tags:         []string{"mysql"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
MySQL 妯℃澘搴斾繚鎸佺洰褰曞弬鏁板寲骞朵繚鎸佺粨鏋勬竻鏅般€?`),
		},
		{
			RelativePath: "service/kerberos.md",
			Scope:        "service",
			Title:        "Kerberos Rules",
			Tags:         []string{"kerberos"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
Kerberos 妯℃澘涓殑 realm/hostname 蹇呴』娓呮櫚銆佸彲瀹¤銆?`),
		},
		{
			RelativePath: "service/elasticsearch.md",
			Scope:        "service",
			Title:        "Elasticsearch Contract",
			Tags:         []string{"elasticsearch", "hostname", "output-layout"},
			Required:     false,
			DefaultBody: strings.TrimSpace(`
Elasticsearch 涓撻」绾︽潫锛?- 鑺傜偣鍗犱綅绗﹀繀椤绘槸 {{ .Instance.Node.Hostname }}銆?- 绂佹 .Instance.NodeAlias / .Instance.Node.HostName銆?- 榛樿浣跨敤 templates/elasticsearch/*.tmpl 骞抽摵缁撴瀯銆?- 鏈槑纭姹傛椂锛屼笉瑕佹斁鍒?templates/elasticsearch/config/銆?- 鑻ョ敤鎴疯姹傗€滀慨鏀瑰叏閮ㄦā鏉库€濓紝蹇呴』鏇存柊鐩稿叧鍏ㄩ儴 draft files锛屼笉鍙敼涓€涓€?`),
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

var templateIntentKeywords = []string{
	"模板", "生成模板", "改模板", "改写模板", "template", ".tmpl", "生成草稿", "draft",
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
	for _, serviceName := range serviceNames {
		if strings.Contains(lower, serviceName) {
			reasons = append(reasons, "service:"+serviceName)
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
func (s *Server) detectSpecialServices(message string, selectedDrafts []aiDraftFile) map[string]bool {
	services := make(map[string]bool)
	messageLower := strings.ToLower(message)
	for _, serviceName := range s.listTemplateServiceNames() {
		if strings.Contains(messageLower, serviceName) {
			services[serviceName] = true
		}
	}
	for _, draft := range selectedDrafts {
		serviceName := strings.ToLower(serviceNameFromTemplatePath(draft.Path))
		if serviceName == "" {
			continue
		}
		services[serviceName] = true
	}
	return services
}
func (s *Server) selectPromptSkills(session *aiSession, req aiSessionMessageRequest, selectedDrafts []aiDraftFile, attachmentIDs []string) ([]aiPromptSkill, error) {
	if err := s.ensureAIDirs(); err != nil {
		return nil, err
	}

	inlineSkillIDs := extractInlineSkillIDs(req.Message)
	explicitSkillIDs := make([]string, 0, len(req.SelectedSkillIDs)+len(inlineSkillIDs))
	explicitSkillIDs = append(explicitSkillIDs, req.SelectedSkillIDs...)
	explicitSkillIDs = append(explicitSkillIDs, inlineSkillIDs...)

	templateIntent, intentReasons := detectTemplateIntent(req.Message, selectedDrafts, s.listTemplateServiceNames())
	hasExplicitSkill := len(explicitSkillIDs) > 0

	specMap := defaultAISkillSpecMap()
	selectedSpecs := []aiPromptSkillSpec{
		specMap["core/output_json.md"],
	}
	if templateIntent || hasExplicitSkill {
		selectedSpecs = append(selectedSpecs,
			specMap["core/base.md"],
			specMap["core/path_guard.md"],
			specMap["core/template_contract.md"],
			specMap["core/variable_contract.md"],
		)
	}
	fmt.Printf("[AI] skill trigger templateIntent=%t explicit=%t reasons=%s\n",
		templateIntent, hasExplicitSkill, summarizePathsForLog(intentReasons, 12))

	kinds := s.detectPromptKinds(req.Message, selectedDrafts, attachmentIDs, session, templateIntent)
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

	for serviceName := range s.detectSpecialServices(req.Message, selectedDrafts) {
		if spec, exists := specMap["service/"+serviceName+".md"]; exists {
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

func (s *Server) buildAIPromptBundle(session *aiSession, req aiSessionMessageRequest, selectedDrafts []aiDraftFile, attachmentIDs []string) (string, aiPromptTrace, error) {
	customRules, err := s.loadAIRules()
	if err != nil {
		return "", aiPromptTrace{}, err
	}

	skills, err := s.selectPromptSkills(session, req, selectedDrafts, attachmentIDs)
	if err != nil {
		return "", aiPromptTrace{}, err
	}

	trace := emptyAIPromptTrace()
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
			return "", aiPromptTrace{}, selectErr
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
