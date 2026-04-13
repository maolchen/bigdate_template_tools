package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"config-generator/config"
)

type userConfigSyncLog struct {
	AddedServices     []string `json:"addedServices"`
	ServiceTopAdded   int      `json:"serviceTopAdded"`
	ServerConfigAdded int      `json:"serverConfigAdded"`
}

type userMetaFile struct {
	ActiveTemplateID string `json:"activeTemplateId"`
	UpdatedAt        string `json:"updatedAt"`
}

type userTemplateEntry struct {
	ID          string `json:"id"`
	FileName    string `json:"fileName"`
	Remark      string `json:"remark"`
	NodeSummary string `json:"nodeSummary"`
	UpdatedAt   string `json:"updatedAt"`
	IsActive    bool   `json:"isActive"`
}

type userTemplateIndexFile struct {
	Templates []userTemplateEntry `json:"templates"`
}

type createConfigTemplateRequest struct {
	ID     string `json:"id"`
	Remark string `json:"remark"`
}

type switchConfigTemplateResponse struct {
	Success          bool              `json:"success"`
	ActiveTemplateID string            `json:"activeTemplateId"`
	Sync             userConfigSyncLog `json:"sync"`
	Config           *config.Config    `json:"config"`
}

func (s *Server) userDir(username string) string {
	return filepath.Join(s.usersRootDir, username)
}

func (s *Server) userConfigsDir(username string) string {
	return filepath.Join(s.userDir(username), "configs")
}

func (s *Server) userMetaPath(username string) string {
	return filepath.Join(s.userDir(username), "meta.json")
}

func (s *Server) userTemplateIndexPath(username string) string {
	return filepath.Join(s.userConfigsDir(username), "templates_index.json")
}

func (s *Server) userOutputDir(username string) string {
	return filepath.Join(s.outputDir, "users", username)
}

func sanitizeTemplateID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("template id 不能为空")
	}
	if len(value) > 64 {
		return "", errors.New("template id 长度不能超过 64")
	}
	for _, ch := range value {
		isAlpha := ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
		isDigit := ch >= '0' && ch <= '9'
		if !isAlpha && !isDigit && ch != '_' && ch != '-' {
			return "", errors.New("template id 仅允许字母、数字、_、-")
		}
	}
	return value, nil
}

func configTemplateFileName(templateID string) string {
	if templateID == "config" {
		return "config.yaml"
	}
	return templateID + "_config.yaml"
}

func (s *Server) configTemplatePath(username, templateID string) string {
	return filepath.Join(s.userConfigsDir(username), configTemplateFileName(templateID))
}

func (s *Server) ensureUserBaseConfig(username string) error {
	configsDir := s.userConfigsDir(username)
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return err
	}

	targetPath := s.configTemplatePath(username, "config")
	if _, err := os.Stat(targetPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	rootData, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) && s.cfg != nil {
			if err := config.SaveConfig(s.configPath, s.cfg); err != nil {
				return err
			}
			rootData, err = os.ReadFile(s.configPath)
		}
		if err != nil {
			return err
		}
	}

	if err := os.WriteFile(targetPath, rootData, 0644); err != nil {
		return err
	}
	fmt.Printf("[UserConfig] bootstrap user config from root: user=%s path=%s\n", username, targetPath)
	return nil
}

func (s *Server) loadUserMeta(username string) (userMetaFile, error) {
	defaultMeta := userMetaFile{
		ActiveTemplateID: "config",
		UpdatedAt:        time.Now().Format(time.RFC3339),
	}
	data, err := os.ReadFile(s.userMetaPath(username))
	if os.IsNotExist(err) {
		return defaultMeta, nil
	}
	if err != nil {
		return userMetaFile{}, err
	}
	if len(data) == 0 {
		return defaultMeta, nil
	}

	var meta userMetaFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return userMetaFile{}, err
	}
	if strings.TrimSpace(meta.ActiveTemplateID) == "" {
		meta.ActiveTemplateID = "config"
	}
	return meta, nil
}

func (s *Server) saveUserMeta(username string, meta userMetaFile) error {
	meta.ActiveTemplateID = strings.TrimSpace(meta.ActiveTemplateID)
	if meta.ActiveTemplateID == "" {
		meta.ActiveTemplateID = "config"
	}
	meta.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := os.MkdirAll(s.userDir(username), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.userMetaPath(username), data, 0644)
}

func (s *Server) resolveUserActiveTemplate(username string) (string, string, error) {
	meta, err := s.loadUserMeta(username)
	if err != nil {
		return "", "", err
	}
	activeID := meta.ActiveTemplateID
	activePath := s.configTemplatePath(username, activeID)
	if _, err := os.Stat(activePath); err == nil {
		return activeID, activePath, nil
	}

	activeID = "config"
	activePath = s.configTemplatePath(username, activeID)
	meta.ActiveTemplateID = activeID
	if err := s.saveUserMeta(username, meta); err != nil {
		return "", "", err
	}
	return activeID, activePath, nil
}

func (s *Server) loadUserConfigByTemplateID(username, templateID string) (*config.Config, string, error) {
	templatePath := s.configTemplatePath(username, templateID)
	cfg, err := config.LoadConfig(templatePath)
	if err != nil {
		return nil, "", err
	}
	return cfg, templatePath, nil
}

func (s *Server) loadUserActiveConfig(username string) (*config.Config, string, string, error) {
	if err := s.ensureUserBaseConfig(username); err != nil {
		return nil, "", "", err
	}
	activeID, activePath, err := s.resolveUserActiveTemplate(username)
	if err != nil {
		return nil, "", "", err
	}
	cfg, err := config.LoadConfig(activePath)
	if err != nil {
		return nil, "", "", err
	}
	return cfg, activePath, activeID, nil
}

func syncServiceTopNodesForUser(mainTopo config.ServiceTopo, userNodes config.Nodes) config.ServiceTopo {
	if len(mainTopo.Nodes) == 0 {
		mainTopo.Nodes = []string{}
		return mainTopo
	}
	filtered := make([]string, 0, len(mainTopo.Nodes))
	for _, nodeName := range mainTopo.Nodes {
		if nodeName == "*" {
			filtered = []string{"*"}
			break
		}
		if _, exists := userNodes[nodeName]; exists {
			filtered = append(filtered, nodeName)
		}
	}
	mainTopo.Nodes = filtered
	return mainTopo
}

func syncMissingServicesFromMain(mainCfg, userCfg *config.Config) (bool, userConfigSyncLog) {
	if mainCfg == nil || userCfg == nil {
		return false, userConfigSyncLog{}
	}
	if userCfg.ServiceTop == nil {
		userCfg.ServiceTop = make(config.ServiceTopos)
	}
	if userCfg.ServerConfig == nil {
		userCfg.ServerConfig = make(config.ServiceConfigs)
	}
	if userCfg.Nodes == nil {
		userCfg.Nodes = make(config.Nodes)
	}

	addedServices := make(map[string]struct{})
	log := userConfigSyncLog{
		AddedServices: []string{},
	}
	changed := false

	for serviceName, mainTopo := range mainCfg.ServiceTop {
		if _, exists := userCfg.ServiceTop[serviceName]; exists {
			continue
		}
		userCfg.ServiceTop[serviceName] = syncServiceTopNodesForUser(mainTopo, userCfg.Nodes)
		addedServices[serviceName] = struct{}{}
		log.ServiceTopAdded++
		changed = true
	}

	for serviceName, mainServiceCfg := range mainCfg.ServerConfig {
		if _, exists := userCfg.ServerConfig[serviceName]; exists {
			continue
		}
		userCfg.ServerConfig[serviceName] = mainServiceCfg
		addedServices[serviceName] = struct{}{}
		log.ServerConfigAdded++
		changed = true
	}

	for serviceName := range addedServices {
		log.AddedServices = append(log.AddedServices, serviceName)
	}
	sort.Strings(log.AddedServices)
	return changed, log
}

func configSummaryFromPath(path string) string {
	cfg, err := config.LoadConfig(path)
	if err != nil || cfg == nil {
		return "-"
	}
	return fmt.Sprintf("%d节点/%d服务", len(cfg.Nodes), len(cfg.ServiceTop))
}

func readUserTemplateIndex(path string) (map[string]userTemplateEntry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]userTemplateEntry), nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return make(map[string]userTemplateEntry), nil
	}
	var indexFile userTemplateIndexFile
	if err := json.Unmarshal(data, &indexFile); err != nil {
		return nil, err
	}
	result := make(map[string]userTemplateEntry, len(indexFile.Templates))
	for _, item := range indexFile.Templates {
		result[item.ID] = item
	}
	return result, nil
}

func (s *Server) refreshUserTemplateIndex(username, activeTemplateID string, remarkOverride map[string]string) ([]userTemplateEntry, error) {
	configsDir := s.userConfigsDir(username)
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return nil, err
	}
	oldMap, err := readUserTemplateIndex(s.userTemplateIndexPath(username))
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return nil, err
	}
	templates := make([]userTemplateEntry, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".yaml") {
			continue
		}
		templateID := ""
		if name == "config.yaml" {
			templateID = "config"
		} else if strings.HasSuffix(name, "_config.yaml") {
			templateID = strings.TrimSuffix(name, "_config.yaml")
		} else {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		item := userTemplateEntry{
			ID:          templateID,
			FileName:    name,
			NodeSummary: configSummaryFromPath(filepath.Join(configsDir, name)),
			UpdatedAt:   info.ModTime().Format(time.RFC3339),
			IsActive:    templateID == activeTemplateID,
		}
		if oldItem, exists := oldMap[templateID]; exists {
			item.Remark = oldItem.Remark
		}
		if remarkOverride != nil {
			if overrideValue, exists := remarkOverride[templateID]; exists {
				item.Remark = strings.TrimSpace(overrideValue)
			}
		}
		templates = append(templates, item)
	}

	sort.SliceStable(templates, func(i, j int) bool {
		if templates[i].ID == "config" {
			return true
		}
		if templates[j].ID == "config" {
			return false
		}
		return templates[i].ID < templates[j].ID
	})

	data, err := json.MarshalIndent(userTemplateIndexFile{Templates: templates}, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(s.userTemplateIndexPath(username), data, 0644); err != nil {
		return nil, err
	}
	return templates, nil
}

// ensureUserConfigForLogin guarantees user workspace config exists and runs incremental service sync.
func (s *Server) ensureUserConfigForLogin(username string) (string, userConfigSyncLog, error) {
	if err := s.ensureUserBaseConfig(username); err != nil {
		return "", userConfigSyncLog{}, err
	}
	userCfg, activePath, activeTemplateID, err := s.loadUserActiveConfig(username)
	if err != nil {
		return "", userConfigSyncLog{}, err
	}
	mainCfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		return "", userConfigSyncLog{}, err
	}
	changed, syncLog := syncMissingServicesFromMain(mainCfg, userCfg)
	if changed {
		if err := config.SaveConfig(activePath, userCfg); err != nil {
			return "", userConfigSyncLog{}, err
		}
	}
	if err := s.saveUserMeta(username, userMetaFile{ActiveTemplateID: activeTemplateID}); err != nil {
		return "", userConfigSyncLog{}, err
	}
	if _, err := s.refreshUserTemplateIndex(username, activeTemplateID, nil); err != nil {
		return "", userConfigSyncLog{}, err
	}
	fmt.Printf("[UserConfig] login sync user=%s activeTemplate=%s serviceTopAdded=%d serverConfigAdded=%d\n",
		username, activeTemplateID, syncLog.ServiceTopAdded, syncLog.ServerConfigAdded)
	return activeTemplateID, syncLog, nil
}

func (s *Server) applyMainServiceSyncToUserTemplate(username, templateID string) (userConfigSyncLog, *config.Config, error) {
	cfg, templatePath, err := s.loadUserConfigByTemplateID(username, templateID)
	if err != nil {
		return userConfigSyncLog{}, nil, err
	}
	mainCfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		return userConfigSyncLog{}, nil, err
	}
	changed, syncLog := syncMissingServicesFromMain(mainCfg, cfg)
	if changed {
		if err := config.SaveConfig(templatePath, cfg); err != nil {
			return userConfigSyncLog{}, nil, err
		}
	}
	return syncLog, cfg, nil
}

func requestPrincipal(r *http.Request) (authPrincipal, error) {
	principal, ok := authPrincipalFromContext(r.Context())
	if !ok {
		return authPrincipal{}, errors.New("unauthorized")
	}
	return principal, nil
}

// handleConfigTemplatesCollection supports list/create user config templates.
func (s *Server) handleConfigTemplatesCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		meta, err := s.loadUserMeta(principal.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		templates, err := s.refreshUserTemplateIndex(principal.Username, meta.ActiveTemplateID, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(templates)
	case http.MethodPost:
		var req createConfigTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		templateID, err := sanitizeTemplateID(req.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if templateID == "config" {
			http.Error(w, "config 是保留模板ID", http.StatusBadRequest)
			return
		}

		_, activePath, activeTemplateID, err := s.loadUserActiveConfig(principal.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		targetPath := s.configTemplatePath(principal.Username, templateID)
		if _, err := os.Stat(targetPath); err == nil {
			http.Error(w, "模板ID已存在", http.StatusBadRequest)
			return
		}
		in, err := os.Open(activePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer in.Close()
		out, err := os.Create(targetPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = out.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := out.Close(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		templates, err := s.refreshUserTemplateIndex(principal.Username, activeTemplateID, map[string]string{
			templateID: strings.TrimSpace(req.Remark),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("[UserConfig] template created user=%s templateID=%s\n", principal.Username, templateID)
		json.NewEncoder(w).Encode(map[string]any{
			"success":   true,
			"templates": templates,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConfigTemplatesDetail supports switch/delete by /api/config-templates/:id/{switch}.
func (s *Server) handleConfigTemplatesDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/config-templates/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.Error(w, "template id is required", http.StatusBadRequest)
		return
	}
	parts := strings.Split(path, "/")
	templateID := parts[0]
	if _, err := sanitizeTemplateID(templateID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(parts) == 2 && parts[1] == "switch" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		templatePath := s.configTemplatePath(principal.Username, templateID)
		if _, err := os.Stat(templatePath); err != nil {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}

		syncLog, cfg, err := s.applyMainServiceSyncToUserTemplate(principal.Username, templateID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.saveUserMeta(principal.Username, userMetaFile{ActiveTemplateID: templateID}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := s.refreshUserTemplateIndex(principal.Username, templateID, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("[UserConfig] template switched user=%s templateID=%s\n", principal.Username, templateID)
		json.NewEncoder(w).Encode(switchConfigTemplateResponse{
			Success:          true,
			ActiveTemplateID: templateID,
			Sync:             syncLog,
			Config:           cfg,
		})
		return
	}

	if len(parts) == 1 && r.Method == http.MethodDelete {
		meta, err := s.loadUserMeta(principal.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if templateID == "config" {
			http.Error(w, "默认模板不可删除", http.StatusBadRequest)
			return
		}
		if meta.ActiveTemplateID == templateID {
			http.Error(w, "当前激活模板不可删除", http.StatusBadRequest)
			return
		}
		templatePath := s.configTemplatePath(principal.Username, templateID)
		if err := os.Remove(templatePath); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "template not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		templates, err := s.refreshUserTemplateIndex(principal.Username, meta.ActiveTemplateID, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("[UserConfig] template deleted user=%s templateID=%s\n", principal.Username, templateID)
		json.NewEncoder(w).Encode(map[string]any{
			"success":   true,
			"templates": templates,
		})
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}
