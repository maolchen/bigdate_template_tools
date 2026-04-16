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
	AddedGlobalKeys             []string `json:"addedGlobalKeys"`
	RemovedGlobalKeys           []string `json:"removedGlobalKeys"`
	GlobalAdded                 int      `json:"globalAdded"`
	GlobalRemoved               int      `json:"globalRemoved"`
	AddedServices               []string `json:"addedServices"`
	RemovedServices             []string `json:"removedServices"`
	AddedServiceTopServices     []string `json:"addedServiceTopServices"`
	RemovedServiceTopServices   []string `json:"removedServiceTopServices"`
	AddedServerConfigServices   []string `json:"addedServerConfigServices"`
	RemovedServerConfigServices []string `json:"removedServerConfigServices"`
	ServiceTopAdded             int      `json:"serviceTopAdded"`
	ServiceTopRemoved           int      `json:"serviceTopRemoved"`
	ServerConfigAdded           int      `json:"serverConfigAdded"`
	ServerConfigRemoved         int      `json:"serverConfigRemoved"`
}

type userMetaFile struct {
	ActiveTemplateID string `json:"activeTemplateId"`
	UpdatedAt        string `json:"updatedAt"`
}

const (
	userMainTemplateID   = "config"
	userMainTemplateFile = "config.yaml"
)

type userTemplateEntry struct {
	ID          string `json:"id"`
	FileName    string `json:"fileName"`
	Kind        string `json:"kind,omitempty"`
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
		return "", errors.New("template id cannot be empty")
	}
	if len(value) > 64 {
		return "", errors.New("template id length cannot exceed 64")
	}
	for _, ch := range value {
		isAlpha := ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
		isDigit := ch >= '0' && ch <= '9'
		if !isAlpha && !isDigit && ch != '_' && ch != '-' {
			return "", errors.New("template id only allows letters, digits, _ and -")
		}
	}
	return value, nil
}

func configTemplateFileName(templateID string) string {
	if templateID == userMainTemplateID {
		return userMainTemplateFile
	}
	return templateID + "_config.yaml"
}

func (s *Server) configTemplatePath(username, templateID string) string {
	return filepath.Join(s.userConfigsDir(username), configTemplateFileName(templateID))
}

func (s *Server) userMainTemplatePath(username string) string {
	return s.configTemplatePath(username, userMainTemplateID)
}

func (s *Server) ensureUserBaseConfig(username string) error {
	configsDir := s.userConfigsDir(username)
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return err
	}

	targetPath := s.userMainTemplatePath(username)
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
		ActiveTemplateID: userMainTemplateID,
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
		meta.ActiveTemplateID = userMainTemplateID
	}
	return meta, nil
}

func (s *Server) saveUserMeta(username string, meta userMetaFile) error {
	meta.ActiveTemplateID = strings.TrimSpace(meta.ActiveTemplateID)
	if meta.ActiveTemplateID == "" {
		meta.ActiveTemplateID = userMainTemplateID
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
	mainPath := s.userMainTemplatePath(username)
	if _, err := os.Stat(mainPath); err != nil {
		return "", "", err
	}
	return userMainTemplateID, mainPath, nil
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

// loadConfigForPrincipal resolves editable config by role.
// Admin always uses root config.yaml; normal user uses user workspace main config.
func (s *Server) loadConfigForPrincipal(principal authPrincipal) (*config.Config, string, string, error) {
	if principal.Role == roleAdmin {
		cfg, err := config.LoadConfig(s.configPath)
		if err != nil {
			return nil, "", "", err
		}
		return cfg, s.configPath, userMainTemplateID, nil
	}
	return s.loadUserActiveConfig(principal.Username)
}

func mergeUserSyncLog(dst *userConfigSyncLog, part userConfigSyncLog) {
	if dst == nil {
		return
	}
	dst.GlobalAdded += part.GlobalAdded
	dst.GlobalRemoved += part.GlobalRemoved
	dst.ServiceTopAdded += part.ServiceTopAdded
	dst.ServiceTopRemoved += part.ServiceTopRemoved
	dst.ServerConfigAdded += part.ServerConfigAdded
	dst.ServerConfigRemoved += part.ServerConfigRemoved

	mergeUniqueSorted := func(base []string, extra []string) []string {
		set := make(map[string]struct{}, len(base)+len(extra))
		for _, item := range base {
			set[item] = struct{}{}
		}
		for _, item := range extra {
			set[item] = struct{}{}
		}
		merged := make([]string, 0, len(set))
		for item := range set {
			merged = append(merged, item)
		}
		sort.Strings(merged)
		return merged
	}

	dst.AddedGlobalKeys = mergeUniqueSorted(dst.AddedGlobalKeys, part.AddedGlobalKeys)
	dst.RemovedGlobalKeys = mergeUniqueSorted(dst.RemovedGlobalKeys, part.RemovedGlobalKeys)
	dst.AddedServices = mergeUniqueSorted(dst.AddedServices, part.AddedServices)
	dst.RemovedServices = mergeUniqueSorted(dst.RemovedServices, part.RemovedServices)
	dst.AddedServiceTopServices = mergeUniqueSorted(dst.AddedServiceTopServices, part.AddedServiceTopServices)
	dst.RemovedServiceTopServices = mergeUniqueSorted(dst.RemovedServiceTopServices, part.RemovedServiceTopServices)
	dst.AddedServerConfigServices = mergeUniqueSorted(dst.AddedServerConfigServices, part.AddedServerConfigServices)
	dst.RemovedServerConfigServices = mergeUniqueSorted(dst.RemovedServerConfigServices, part.RemovedServerConfigServices)
}

func newUserConfigSyncLog() userConfigSyncLog {
	return userConfigSyncLog{
		AddedGlobalKeys:             []string{},
		RemovedGlobalKeys:           []string{},
		AddedServices:               []string{},
		RemovedServices:             []string{},
		AddedServiceTopServices:     []string{},
		RemovedServiceTopServices:   []string{},
		AddedServerConfigServices:   []string{},
		RemovedServerConfigServices: []string{},
	}
}

func formatSyncLog(log userConfigSyncLog) string {
	return fmt.Sprintf(
		"globalAdded=%d removed=%d addedGlobalKeys=%v removedGlobalKeys=%v serviceTopAdded=%d removed=%d addedServiceTopServices=%v removedServiceTopServices=%v serverConfigAdded=%d removed=%d addedServerConfigServices=%v removedServerConfigServices=%v addedServices=%v removedServices=%v",
		log.GlobalAdded,
		log.GlobalRemoved,
		log.AddedGlobalKeys,
		log.RemovedGlobalKeys,
		log.ServiceTopAdded,
		log.ServiceTopRemoved,
		log.AddedServiceTopServices,
		log.RemovedServiceTopServices,
		log.ServerConfigAdded,
		log.ServerConfigRemoved,
		log.AddedServerConfigServices,
		log.RemovedServerConfigServices,
		log.AddedServices,
		log.RemovedServices,
	)
}

func (s *Server) listUserBackupTemplateIDs(username string) ([]string, error) {
	configsDir := s.userConfigsDir(username)
	entries, err := os.ReadDir(configsDir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_config.yaml") {
			result = append(result, strings.TrimSuffix(name, "_config.yaml"))
		}
	}
	sort.Strings(result)
	return result, nil
}

func syncTemplateFromMain(templatePath string, mainCfg *config.Config) (bool, userConfigSyncLog, error) {
	cfg, err := config.LoadConfig(templatePath)
	if err != nil {
		return false, newUserConfigSyncLog(), err
	}
	changed, syncLog := syncMissingServicesFromMain(mainCfg, cfg)
	if changed {
		if err := config.SaveConfig(templatePath, cfg); err != nil {
			return false, newUserConfigSyncLog(), err
		}
	}
	return changed, syncLog, nil
}

// syncUserBackupsFromMainConfig applies incremental sync from mainCfg into all backup templates.
func (s *Server) syncUserBackupsFromMainConfig(username string, mainCfg *config.Config) (userConfigSyncLog, error) {
	backupIDs, err := s.listUserBackupTemplateIDs(username)
	if err != nil {
		return newUserConfigSyncLog(), err
	}
	total := newUserConfigSyncLog()
	for _, templateID := range backupIDs {
		templatePath := s.configTemplatePath(username, templateID)
		changed, logPart, err := syncTemplateFromMain(templatePath, mainCfg)
		if err != nil {
			fmt.Printf("[UserConfig] backup sync failed user=%s template=%s path=%s err=%v\n", username, templateID, templatePath, err)
			return newUserConfigSyncLog(), err
		}
		mergeUserSyncLog(&total, logPart)
		fmt.Printf("[UserConfig] backup sync user=%s template=%s path=%s changed=%t %s\n",
			username, templateID, templatePath, changed, formatSyncLog(logPart))
	}
	return total, nil
}

// syncUserTemplatesFromRoot synchronizes root config additions to one user's main and backups.
func (s *Server) syncUserTemplatesFromRoot(username string, rootCfg *config.Config) (userConfigSyncLog, error) {
	if err := s.ensureUserBaseConfig(username); err != nil {
		return newUserConfigSyncLog(), err
	}
	if err := s.migrateLegacyActiveTemplateToMain(username); err != nil {
		return newUserConfigSyncLog(), err
	}
	mainPath := s.userMainTemplatePath(username)
	mainChanged, mainLog, err := syncTemplateFromMain(mainPath, rootCfg)
	if err != nil {
		fmt.Printf("[UserConfig] root sync main failed user=%s path=%s err=%v\n", username, mainPath, err)
		return newUserConfigSyncLog(), err
	}
	backupLog, err := s.syncUserBackupsFromMainConfig(username, rootCfg)
	if err != nil {
		return newUserConfigSyncLog(), err
	}
	total := newUserConfigSyncLog()
	mergeUserSyncLog(&total, mainLog)
	mergeUserSyncLog(&total, backupLog)
	if err := s.saveUserMeta(username, userMetaFile{ActiveTemplateID: userMainTemplateID}); err != nil {
		return newUserConfigSyncLog(), err
	}
	if _, err := s.refreshUserTemplateIndex(username, userMainTemplateID, nil); err != nil {
		return newUserConfigSyncLog(), err
	}
	fmt.Printf("[UserConfig] root sync user=%s mainPath=%s mainChanged=%t %s\n",
		username, mainPath, mainChanged, formatSyncLog(total))
	return total, nil
}

func (s *Server) listNonAdminUsers() []string {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	result := make([]string, 0, len(s.authUsers))
	for username, user := range s.authUsers {
		if user.Role == roleAdmin {
			continue
		}
		result = append(result, username)
	}
	sort.Strings(result)
	return result
}

// syncRootConfigToAllUsers propagates root config additions to every non-admin user workspace.
func (s *Server) syncRootConfigToAllUsers(rootCfg *config.Config) (int, userConfigSyncLog) {
	users := s.listNonAdminUsers()
	total := newUserConfigSyncLog()
	for _, username := range users {
		logPart, err := s.syncUserTemplatesFromRoot(username, rootCfg)
		if err != nil {
			fmt.Printf("[UserConfig] root sync user failed user=%s err=%v\n", username, err)
			continue
		}
		mergeUserSyncLog(&total, logPart)
	}
	return len(users), total
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// swapTemplateFiles swaps two template files by rename to avoid intermediate overwrite.
func swapTemplateFiles(pathA, pathB string) error {
	tmpPath := pathA + fmt.Sprintf(".swap.%d", time.Now().UnixNano())
	if err := os.Rename(pathA, tmpPath); err != nil {
		return err
	}
	if err := os.Rename(pathB, pathA); err != nil {
		_ = os.Rename(tmpPath, pathA)
		return err
	}
	if err := os.Rename(tmpPath, pathB); err != nil {
		// Best-effort rollback.
		_ = os.Rename(pathA, pathB)
		_ = os.Rename(tmpPath, pathA)
		return err
	}
	return nil
}

// migrateLegacyActiveTemplateToMain keeps old active template behavior compatible:
// if historical activeTemplateId was not "config", swap it into config.yaml once.
func (s *Server) migrateLegacyActiveTemplateToMain(username string) error {
	meta, err := s.loadUserMeta(username)
	if err != nil {
		return err
	}
	legacyID := strings.TrimSpace(meta.ActiveTemplateID)
	if legacyID == "" || legacyID == userMainTemplateID {
		return s.saveUserMeta(username, userMetaFile{ActiveTemplateID: userMainTemplateID})
	}

	mainPath := s.userMainTemplatePath(username)
	legacyPath := s.configTemplatePath(username, legacyID)
	ok, err := fileExists(legacyPath)
	if err != nil {
		return err
	}
	if ok {
		if err := swapTemplateFiles(mainPath, legacyPath); err != nil {
			return err
		}
		fmt.Printf("[UserConfig] migrated legacy active template user=%s legacyTemplate=%s\n", username, legacyID)
	}
	return s.saveUserMeta(username, userMetaFile{ActiveTemplateID: userMainTemplateID})
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

// syncMissingServicesFromMain applies incremental additions from root config into target config.
// Only missing global keys and missing service entries are added; existing values are kept as-is.
func syncMissingServicesFromMain(mainCfg, userCfg *config.Config) (bool, userConfigSyncLog) {
	if mainCfg == nil || userCfg == nil {
		return false, newUserConfigSyncLog()
	}
	if userCfg.Global == nil {
		userCfg.Global = make(config.Global)
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

	addedGlobalKeys := make(map[string]struct{})
	addedServices := make(map[string]struct{})
	addedServiceTopServices := make(map[string]struct{})
	addedServerConfigServices := make(map[string]struct{})
	log := newUserConfigSyncLog()
	changed := false

	for key, value := range mainCfg.Global {
		if _, exists := userCfg.Global[key]; exists {
			continue
		}
		userCfg.Global[key] = value
		addedGlobalKeys[key] = struct{}{}
		log.GlobalAdded++
		changed = true
	}

	for serviceName, mainTopo := range mainCfg.ServiceTop {
		if _, exists := userCfg.ServiceTop[serviceName]; exists {
			continue
		}
		userCfg.ServiceTop[serviceName] = syncServiceTopNodesForUser(mainTopo, userCfg.Nodes)
		addedServices[serviceName] = struct{}{}
		addedServiceTopServices[serviceName] = struct{}{}
		log.ServiceTopAdded++
		changed = true
	}

	for serviceName, mainServiceCfg := range mainCfg.ServerConfig {
		if _, exists := userCfg.ServerConfig[serviceName]; exists {
			continue
		}
		userCfg.ServerConfig[serviceName] = mainServiceCfg
		addedServices[serviceName] = struct{}{}
		addedServerConfigServices[serviceName] = struct{}{}
		log.ServerConfigAdded++
		changed = true
	}

	for key := range addedGlobalKeys {
		log.AddedGlobalKeys = append(log.AddedGlobalKeys, key)
	}
	sort.Strings(log.AddedGlobalKeys)

	for serviceName := range addedServices {
		log.AddedServices = append(log.AddedServices, serviceName)
	}
	sort.Strings(log.AddedServices)
	for serviceName := range addedServiceTopServices {
		log.AddedServiceTopServices = append(log.AddedServiceTopServices, serviceName)
	}
	sort.Strings(log.AddedServiceTopServices)
	for serviceName := range addedServerConfigServices {
		log.AddedServerConfigServices = append(log.AddedServerConfigServices, serviceName)
	}
	sort.Strings(log.AddedServerConfigServices)
	return changed, log
}

// diffConfigChangesFromMain computes a non-destructive summary of root-vs-user differences.
// It is used for user-facing reminders (showing what was added/removed in root),
// while actual sync behavior still follows incremental-add-only policy.
func diffConfigChangesFromMain(mainCfg, userCfg *config.Config) userConfigSyncLog {
	log := newUserConfigSyncLog()
	if mainCfg == nil || userCfg == nil {
		return log
	}
	if mainCfg.Global == nil {
		mainCfg.Global = make(config.Global)
	}
	if userCfg.Global == nil {
		userCfg.Global = make(config.Global)
	}
	if mainCfg.ServiceTop == nil {
		mainCfg.ServiceTop = make(config.ServiceTopos)
	}
	if userCfg.ServiceTop == nil {
		userCfg.ServiceTop = make(config.ServiceTopos)
	}
	if mainCfg.ServerConfig == nil {
		mainCfg.ServerConfig = make(config.ServiceConfigs)
	}
	if userCfg.ServerConfig == nil {
		userCfg.ServerConfig = make(config.ServiceConfigs)
	}

	addedGlobalKeys := make(map[string]struct{})
	removedGlobalKeys := make(map[string]struct{})
	addedTop := make(map[string]struct{})
	removedTop := make(map[string]struct{})
	addedCfg := make(map[string]struct{})
	removedCfg := make(map[string]struct{})

	for key := range mainCfg.Global {
		if _, exists := userCfg.Global[key]; !exists {
			addedGlobalKeys[key] = struct{}{}
		}
	}
	for key := range userCfg.Global {
		if _, exists := mainCfg.Global[key]; !exists {
			removedGlobalKeys[key] = struct{}{}
		}
	}

	for serviceName := range mainCfg.ServiceTop {
		if _, exists := userCfg.ServiceTop[serviceName]; !exists {
			addedTop[serviceName] = struct{}{}
		}
	}
	for serviceName := range userCfg.ServiceTop {
		if _, exists := mainCfg.ServiceTop[serviceName]; !exists {
			removedTop[serviceName] = struct{}{}
		}
	}

	for serviceName := range mainCfg.ServerConfig {
		if _, exists := userCfg.ServerConfig[serviceName]; !exists {
			addedCfg[serviceName] = struct{}{}
		}
	}
	for serviceName := range userCfg.ServerConfig {
		if _, exists := mainCfg.ServerConfig[serviceName]; !exists {
			removedCfg[serviceName] = struct{}{}
		}
	}

	appendSorted := func(dst []string, set map[string]struct{}) []string {
		for item := range set {
			dst = append(dst, item)
		}
		sort.Strings(dst)
		return dst
	}
	log.AddedGlobalKeys = appendSorted(log.AddedGlobalKeys, addedGlobalKeys)
	log.RemovedGlobalKeys = appendSorted(log.RemovedGlobalKeys, removedGlobalKeys)
	log.AddedServiceTopServices = appendSorted(log.AddedServiceTopServices, addedTop)
	log.RemovedServiceTopServices = appendSorted(log.RemovedServiceTopServices, removedTop)
	log.AddedServerConfigServices = appendSorted(log.AddedServerConfigServices, addedCfg)
	log.RemovedServerConfigServices = appendSorted(log.RemovedServerConfigServices, removedCfg)

	log.GlobalAdded = len(log.AddedGlobalKeys)
	log.GlobalRemoved = len(log.RemovedGlobalKeys)
	log.ServiceTopAdded = len(log.AddedServiceTopServices)
	log.ServiceTopRemoved = len(log.RemovedServiceTopServices)
	log.ServerConfigAdded = len(log.AddedServerConfigServices)
	log.ServerConfigRemoved = len(log.RemovedServerConfigServices)
	log.AddedServices = appendSorted(log.AddedServices, unionStringSets(addedTop, addedCfg))
	log.RemovedServices = appendSorted(log.RemovedServices, unionStringSets(removedTop, removedCfg))
	return log
}

func unionStringSets(a, b map[string]struct{}) map[string]struct{} {
	merged := make(map[string]struct{}, len(a)+len(b))
	for key := range a {
		merged[key] = struct{}{}
	}
	for key := range b {
		merged[key] = struct{}{}
	}
	return merged
}

// removeDeletedFromMain deletes items from userCfg when they no longer exist in mainCfg.
// This is an explicit dangerous operation and should only run after user confirmation.
func removeDeletedFromMain(mainCfg, userCfg *config.Config) (bool, userConfigSyncLog) {
	log := newUserConfigSyncLog()
	if mainCfg == nil || userCfg == nil {
		return false, log
	}
	if mainCfg.Global == nil {
		mainCfg.Global = make(config.Global)
	}
	if userCfg.Global == nil {
		userCfg.Global = make(config.Global)
	}
	if mainCfg.ServiceTop == nil {
		mainCfg.ServiceTop = make(config.ServiceTopos)
	}
	if userCfg.ServiceTop == nil {
		userCfg.ServiceTop = make(config.ServiceTopos)
	}
	if mainCfg.ServerConfig == nil {
		mainCfg.ServerConfig = make(config.ServiceConfigs)
	}
	if userCfg.ServerConfig == nil {
		userCfg.ServerConfig = make(config.ServiceConfigs)
	}

	removedGlobal := make(map[string]struct{})
	removedTop := make(map[string]struct{})
	removedCfg := make(map[string]struct{})
	changed := false

	for key := range userCfg.Global {
		if _, exists := mainCfg.Global[key]; exists {
			continue
		}
		delete(userCfg.Global, key)
		removedGlobal[key] = struct{}{}
		log.GlobalRemoved++
		changed = true
	}
	for serviceName := range userCfg.ServiceTop {
		if _, exists := mainCfg.ServiceTop[serviceName]; exists {
			continue
		}
		delete(userCfg.ServiceTop, serviceName)
		removedTop[serviceName] = struct{}{}
		log.ServiceTopRemoved++
		changed = true
	}
	for serviceName := range userCfg.ServerConfig {
		if _, exists := mainCfg.ServerConfig[serviceName]; exists {
			continue
		}
		delete(userCfg.ServerConfig, serviceName)
		removedCfg[serviceName] = struct{}{}
		log.ServerConfigRemoved++
		changed = true
	}

	appendSorted := func(dst []string, set map[string]struct{}) []string {
		for item := range set {
			dst = append(dst, item)
		}
		sort.Strings(dst)
		return dst
	}
	log.RemovedGlobalKeys = appendSorted(log.RemovedGlobalKeys, removedGlobal)
	log.RemovedServiceTopServices = appendSorted(log.RemovedServiceTopServices, removedTop)
	log.RemovedServerConfigServices = appendSorted(log.RemovedServerConfigServices, removedCfg)
	log.RemovedServices = appendSorted(log.RemovedServices, unionStringSets(removedTop, removedCfg))
	return changed, log
}

func configSummaryFromPath(path string) string {
	cfg, err := config.LoadConfig(path)
	if err != nil || cfg == nil {
		return "-"
	}
	return fmt.Sprintf("%d nodes / %d services", len(cfg.Nodes), len(cfg.ServiceTop))
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
		if name == userMainTemplateFile {
			templateID = userMainTemplateID
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
		if templateID == userMainTemplateID {
			item.Kind = "main"
		} else {
			item.Kind = "backup"
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
		if templates[i].ID == userMainTemplateID {
			return true
		}
		if templates[j].ID == userMainTemplateID {
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

// swapUserTemplateRemarks keeps remarks attached to template content during a
// main/backup file swap. Without this, remarks stay with template IDs while the
// underlying YAML content has already moved to another file.
func (s *Server) swapUserTemplateRemarks(username, backupTemplateID string) (map[string]string, error) {
	oldMap, err := readUserTemplateIndex(s.userTemplateIndexPath(username))
	if err != nil {
		return nil, err
	}
	mainRemark := oldMap[userMainTemplateID].Remark
	backupRemark := oldMap[backupTemplateID].Remark
	return map[string]string{
		userMainTemplateID: backupRemark,
		backupTemplateID:   mainRemark,
	}, nil
}

// ensureUserConfigForLogin guarantees user workspace config exists and runs incremental service sync.
func (s *Server) ensureUserConfigForLogin(username string) (string, userConfigSyncLog, error) {
	mainCfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		return "", newUserConfigSyncLog(), err
	}
	syncLog, err := s.syncUserTemplatesFromRoot(username, mainCfg)
	if err != nil {
		return "", newUserConfigSyncLog(), err
	}
	fmt.Printf("[UserConfig] login sync user=%s activeTemplate=%s %s\n", username, userMainTemplateID, formatSyncLog(syncLog))
	return userMainTemplateID, syncLog, nil
}

func (s *Server) applyMainServiceSyncToUserTemplate(username, templateID string) (userConfigSyncLog, *config.Config, error) {
	cfg, templatePath, err := s.loadUserConfigByTemplateID(username, templateID)
	if err != nil {
		return newUserConfigSyncLog(), nil, err
	}
	mainCfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		return newUserConfigSyncLog(), nil, err
	}
	changed, syncLog := syncMissingServicesFromMain(mainCfg, cfg)
	if changed {
		if err := config.SaveConfig(templatePath, cfg); err != nil {
			return newUserConfigSyncLog(), nil, err
		}
	}
	fmt.Printf("[UserConfig] switch pre-sync user=%s template=%s changed=%t %s\n", username, templateID, changed, formatSyncLog(syncLog))
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
		templates, err := s.refreshUserTemplateIndex(principal.Username, userMainTemplateID, nil)
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
		if templateID == userMainTemplateID {
			http.Error(w, "config is reserved as main template", http.StatusBadRequest)
			return
		}

		mainPath := s.userMainTemplatePath(principal.Username)
		activeTemplateID := userMainTemplateID
		targetPath := s.configTemplatePath(principal.Username, templateID)
		if _, err := os.Stat(targetPath); err == nil {
			http.Error(w, "template id already exists", http.StatusBadRequest)
			return
		}
		in, err := os.Open(mainPath)
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
		if templateID == userMainTemplateID {
			http.Error(w, "config is already main template", http.StatusBadRequest)
			return
		}
		templatePath := s.configTemplatePath(principal.Username, templateID)
		if _, err := os.Stat(templatePath); err != nil {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}

		// Keep target backup updated with newly added services before switching.
		syncLog, _, err := s.applyMainServiceSyncToUserTemplate(principal.Username, templateID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		mainPath := s.userMainTemplatePath(principal.Username)
		if err := swapTemplateFiles(mainPath, templatePath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.saveUserMeta(principal.Username, userMetaFile{ActiveTemplateID: userMainTemplateID}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		remarkSwap, err := s.swapUserTemplateRemarks(principal.Username, templateID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := s.refreshUserTemplateIndex(principal.Username, userMainTemplateID, remarkSwap); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cfg, _, _, err := s.loadUserActiveConfig(principal.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("[UserConfig] template switched by swap user=%s targetBackup=%s main=%s\n", principal.Username, templateID, userMainTemplateID)
		json.NewEncoder(w).Encode(switchConfigTemplateResponse{
			Success:          true,
			ActiveTemplateID: userMainTemplateID,
			Sync:             syncLog,
			Config:           cfg,
		})
		return
	}

	if len(parts) == 1 && r.Method == http.MethodDelete {
		if templateID == userMainTemplateID {
			http.Error(w, "main template cannot be deleted", http.StatusBadRequest)
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
		templates, err := s.refreshUserTemplateIndex(principal.Username, userMainTemplateID, nil)
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
