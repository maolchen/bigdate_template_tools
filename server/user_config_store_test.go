package server

import (
	"os"
	"path/filepath"
	"testing"

	"config-generator/config"
)

func testCfg(nodeName string, port int) *config.Config {
	return &config.Config{
		Global: config.Global{
			"user": "tester",
		},
		Nodes: config.Nodes{
			nodeName: {IP: "10.0.0.1", Hostname: nodeName},
		},
		ServiceTop: config.ServiceTopos{
			"zookeeper": {Nodes: []string{nodeName}},
		},
		ServerConfig: config.ServiceConfigs{
			"zookeeper": {Vars: map[string]any{"port": port}},
		},
	}
}

func TestSyncMissingServicesFromMainAddsOnlyMissingAndFiltersNodes(t *testing.T) {
	mainCfg := &config.Config{
		Global: config.Global{
			"user":             "root-admin",
			"install_base_dir": "/opt/bigdata",
		},
		Nodes: config.Nodes{
			"master": {IP: "10.0.0.1"},
			"node2":  {IP: "10.0.0.2"},
		},
		ServiceTop: config.ServiceTopos{
			"zookeeper": {Nodes: []string{"master", "node2"}},
			"hdfs":      {Nodes: []string{"*"}},
		},
		ServerConfig: config.ServiceConfigs{
			"zookeeper": {Vars: map[string]any{"port": 2181}},
			"hdfs":      {Vars: map[string]any{"nn": "on"}},
		},
	}
	userCfg := &config.Config{
		Global: config.Global{
			"user": "workspace-user",
		},
		Nodes: config.Nodes{
			"master-a": {IP: "10.1.0.1"},
		},
		ServiceTop: config.ServiceTopos{},
		ServerConfig: config.ServiceConfigs{
			"zookeeper": {Vars: map[string]any{"port": 2281}},
		},
	}

	changed, syncLog := syncMissingServicesFromMain(mainCfg, userCfg)
	if !changed {
		t.Fatal("expected changed=true")
	}
	if syncLog.ServiceTopAdded != 2 {
		t.Fatalf("unexpected serviceTopAdded: %d", syncLog.ServiceTopAdded)
	}
	if syncLog.ServerConfigAdded != 1 {
		t.Fatalf("unexpected serverConfigAdded: %d", syncLog.ServerConfigAdded)
	}
	if syncLog.GlobalAdded != 1 {
		t.Fatalf("unexpected globalAdded: %d", syncLog.GlobalAdded)
	}

	if got := userCfg.ServiceTop["zookeeper"].Nodes; len(got) != 0 {
		t.Fatalf("expected zookeeper nodes to be filtered empty, got: %#v", got)
	}
	if got := userCfg.ServiceTop["hdfs"].Nodes; len(got) != 1 || got[0] != "*" {
		t.Fatalf("expected hdfs nodes to keep wildcard, got: %#v", got)
	}
	if got := userCfg.ServerConfig["zookeeper"].Vars["port"]; got != 2281 {
		t.Fatalf("existing service config should not be overwritten, got: %v", got)
	}
	if got := userCfg.ServerConfig["hdfs"].Vars["nn"]; got != "on" {
		t.Fatalf("expected hdfs copied from main, got: %v", got)
	}
	if got := userCfg.Global["install_base_dir"]; got != "/opt/bigdata" {
		t.Fatalf("expected missing global key copied from main, got: %v", got)
	}
	if got := userCfg.Global["user"]; got != "workspace-user" {
		t.Fatalf("expected existing global key unchanged, got: %v", got)
	}
}

func TestSyncMissingServicesFromMainReportsAddedGlobalKey(t *testing.T) {
	mainCfg := testCfg("n1", 2181)
	mainCfg.Global["test"] = "test"
	userCfg := testCfg("n1", 2181)
	delete(userCfg.Global, "test")

	changed, syncLog := syncMissingServicesFromMain(mainCfg, userCfg)
	if !changed {
		t.Fatal("expected changed=true")
	}
	if syncLog.GlobalAdded != 1 {
		t.Fatalf("expected one global key added, got %+v", syncLog)
	}
	if len(syncLog.AddedGlobalKeys) != 1 || syncLog.AddedGlobalKeys[0] != "test" {
		t.Fatalf("expected added global key test, got %#v", syncLog.AddedGlobalKeys)
	}
	if got := userCfg.Global["test"]; got != "test" {
		t.Fatalf("expected test global copied to user config, got %v", got)
	}
}

func TestDiffConfigChangesFromMainReportsAddedAndRemoved(t *testing.T) {
	mainCfg := &config.Config{
		Global: config.Global{
			"user":             "root-admin",
			"install_base_dir": "/opt/bigdata",
		},
		ServiceTop: config.ServiceTopos{
			"zookeeper": {Nodes: []string{"master"}},
			"spark3":    {Nodes: []string{"master"}},
		},
		ServerConfig: config.ServiceConfigs{
			"zookeeper": {Vars: map[string]any{"port": 2181}},
			"spark3":    {Vars: map[string]any{"spark_port": 7077}},
		},
	}
	userCfg := &config.Config{
		Global: config.Global{
			"user":      "workspace-user",
			"legacyKey": "legacy-value",
		},
		ServiceTop: config.ServiceTopos{
			"zookeeper": {Nodes: []string{"master-a"}},
			"oldsvc":    {Nodes: []string{"master-a"}},
		},
		ServerConfig: config.ServiceConfigs{
			"zookeeper": {Vars: map[string]any{"port": 2281}},
			"oldsvc":    {Vars: map[string]any{"x": 1}},
		},
	}

	diff := diffConfigChangesFromMain(mainCfg, userCfg)
	if diff.GlobalAdded != 1 || diff.GlobalRemoved != 1 {
		t.Fatalf("unexpected global diff: %+v", diff)
	}
	if diff.ServiceTopAdded != 1 || diff.ServiceTopRemoved != 1 {
		t.Fatalf("unexpected serviceTop diff: %+v", diff)
	}
	if diff.ServerConfigAdded != 1 || diff.ServerConfigRemoved != 1 {
		t.Fatalf("unexpected serverConfig diff: %+v", diff)
	}
	if len(diff.AddedGlobalKeys) != 1 || diff.AddedGlobalKeys[0] != "install_base_dir" {
		t.Fatalf("unexpected added global keys: %#v", diff.AddedGlobalKeys)
	}
	if len(diff.RemovedGlobalKeys) != 1 || diff.RemovedGlobalKeys[0] != "legacyKey" {
		t.Fatalf("unexpected removed global keys: %#v", diff.RemovedGlobalKeys)
	}
	if len(diff.AddedServiceTopServices) != 1 || diff.AddedServiceTopServices[0] != "spark3" {
		t.Fatalf("unexpected added serviceTop services: %#v", diff.AddedServiceTopServices)
	}
	if len(diff.RemovedServiceTopServices) != 1 || diff.RemovedServiceTopServices[0] != "oldsvc" {
		t.Fatalf("unexpected removed serviceTop services: %#v", diff.RemovedServiceTopServices)
	}
	if len(diff.AddedServerConfigServices) != 1 || diff.AddedServerConfigServices[0] != "spark3" {
		t.Fatalf("unexpected added serverConfig services: %#v", diff.AddedServerConfigServices)
	}
	if len(diff.RemovedServerConfigServices) != 1 || diff.RemovedServerConfigServices[0] != "oldsvc" {
		t.Fatalf("unexpected removed serverConfig services: %#v", diff.RemovedServerConfigServices)
	}
}

func TestSanitizeTemplateID(t *testing.T) {
	if _, err := sanitizeTemplateID("abc_1"); err != nil {
		t.Fatalf("expected valid id, got err=%v", err)
	}
	if _, err := sanitizeTemplateID("bad id"); err == nil {
		t.Fatal("expected invalid id with spaces")
	}
	if _, err := sanitizeTemplateID("../bad"); err == nil {
		t.Fatal("expected invalid id with slash")
	}
}

func TestLoadUserActiveConfigAlwaysUsesMainTemplate(t *testing.T) {
	workDir := t.TempDir()
	s := NewServer(workDir)
	username := "alice"

	if err := config.SaveConfig(s.configPath, testCfg("root-node", 2181)); err != nil {
		t.Fatalf("save root config: %v", err)
	}
	if err := s.ensureUserBaseConfig(username); err != nil {
		t.Fatalf("ensure base config: %v", err)
	}
	if err := config.SaveConfig(s.configTemplatePath(username, "legacy"), testCfg("legacy-node", 3181)); err != nil {
		t.Fatalf("save legacy template: %v", err)
	}
	if err := s.saveUserMeta(username, userMetaFile{ActiveTemplateID: "legacy"}); err != nil {
		t.Fatalf("save meta: %v", err)
	}

	cfg, path, activeID, err := s.loadUserActiveConfig(username)
	if err != nil {
		t.Fatalf("load user active config: %v", err)
	}
	if activeID != userMainTemplateID {
		t.Fatalf("expected active template id %q, got %q", userMainTemplateID, activeID)
	}
	if filepath.Base(path) != userMainTemplateFile {
		t.Fatalf("expected main template file %q, got %q", userMainTemplateFile, filepath.Base(path))
	}
	if _, ok := cfg.Nodes["root-node"]; !ok {
		t.Fatalf("expected to load main template content")
	}
}

func TestMigrateLegacyActiveTemplateToMainSwapsContent(t *testing.T) {
	workDir := t.TempDir()
	s := NewServer(workDir)
	username := "bob"

	if err := config.SaveConfig(s.configPath, testCfg("root-node", 2181)); err != nil {
		t.Fatalf("save root config: %v", err)
	}
	if err := s.ensureUserBaseConfig(username); err != nil {
		t.Fatalf("ensure base config: %v", err)
	}
	mainPath := s.userMainTemplatePath(username)
	legacyPath := s.configTemplatePath(username, "project_4node")

	mainCfg := testCfg("main-node", 2181)
	legacyCfg := testCfg("legacy-node", 3181)
	if err := config.SaveConfig(mainPath, mainCfg); err != nil {
		t.Fatalf("save main cfg: %v", err)
	}
	if err := config.SaveConfig(legacyPath, legacyCfg); err != nil {
		t.Fatalf("save legacy cfg: %v", err)
	}
	if err := s.saveUserMeta(username, userMetaFile{ActiveTemplateID: "project_4node"}); err != nil {
		t.Fatalf("save meta: %v", err)
	}

	if err := s.migrateLegacyActiveTemplateToMain(username); err != nil {
		t.Fatalf("migrate legacy active template: %v", err)
	}

	mainAfter, err := config.LoadConfig(mainPath)
	if err != nil {
		t.Fatalf("load main after migration: %v", err)
	}
	legacyAfter, err := config.LoadConfig(legacyPath)
	if err != nil {
		t.Fatalf("load legacy after migration: %v", err)
	}
	if _, ok := mainAfter.Nodes["legacy-node"]; !ok {
		t.Fatalf("expected main template to contain legacy content after swap")
	}
	if _, ok := legacyAfter.Nodes["main-node"]; !ok {
		t.Fatalf("expected legacy template to contain old main content after swap")
	}

	meta, err := s.loadUserMeta(username)
	if err != nil {
		t.Fatalf("load meta after migration: %v", err)
	}
	if meta.ActiveTemplateID != userMainTemplateID {
		t.Fatalf("expected meta activeTemplateId=%q, got %q", userMainTemplateID, meta.ActiveTemplateID)
	}
}

func TestSwapUserTemplateRemarksKeepsRemarkWithContent(t *testing.T) {
	workDir := t.TempDir()
	s := NewServer(workDir)
	username := "alice"

	if err := os.MkdirAll(s.userConfigsDir(username), 0o755); err != nil {
		t.Fatalf("mkdir user configs dir: %v", err)
	}
	mainPath := s.userMainTemplatePath(username)
	backupPath := s.configTemplatePath(username, "project7")
	if err := config.SaveConfig(mainPath, testCfg("main-node", 2181)); err != nil {
		t.Fatalf("save main cfg: %v", err)
	}
	if err := config.SaveConfig(backupPath, testCfg("seven-node", 3181)); err != nil {
		t.Fatalf("save backup cfg: %v", err)
	}
	if _, err := s.refreshUserTemplateIndex(username, userMainTemplateID, map[string]string{
		userMainTemplateID: "4nodes/23services",
		"project7":         "7节点服务",
	}); err != nil {
		t.Fatalf("refresh initial index: %v", err)
	}

	if err := swapTemplateFiles(mainPath, backupPath); err != nil {
		t.Fatalf("swap template files: %v", err)
	}
	remarkSwap, err := s.swapUserTemplateRemarks(username, "project7")
	if err != nil {
		t.Fatalf("swap user template remarks: %v", err)
	}
	templates, err := s.refreshUserTemplateIndex(username, userMainTemplateID, remarkSwap)
	if err != nil {
		t.Fatalf("refresh swapped index: %v", err)
	}

	got := map[string]string{}
	for _, item := range templates {
		got[item.ID] = item.Remark
	}
	if got[userMainTemplateID] != "7节点服务" {
		t.Fatalf("expected main remark to follow promoted backup content, got %q", got[userMainTemplateID])
	}
	if got["project7"] != "4nodes/23services" {
		t.Fatalf("expected backup remark to follow demoted main content, got %q", got["project7"])
	}
}

func TestLoadConfigForPrincipalAdminUsesRoot(t *testing.T) {
	workDir := t.TempDir()
	s := NewServer(workDir)

	rootCfg := testCfg("root-node", 2181)
	if err := config.SaveConfig(s.configPath, rootCfg); err != nil {
		t.Fatalf("save root config: %v", err)
	}
	if err := s.ensureUserBaseConfig("admin"); err != nil {
		t.Fatalf("ensure admin user base config: %v", err)
	}
	if err := config.SaveConfig(s.userMainTemplatePath("admin"), testCfg("admin-node", 3181)); err != nil {
		t.Fatalf("save admin user-space config: %v", err)
	}

	cfg, path, _, err := s.loadConfigForPrincipal(authPrincipal{Username: "admin", Role: roleAdmin})
	if err != nil {
		t.Fatalf("load config for admin principal: %v", err)
	}
	if path != s.configPath {
		t.Fatalf("expected admin path %q, got %q", s.configPath, path)
	}
	if _, ok := cfg.Nodes["root-node"]; !ok {
		t.Fatalf("expected admin principal to load root config content")
	}
	if _, ok := cfg.Nodes["admin-node"]; ok {
		t.Fatalf("admin principal should not load user-space config")
	}
}

func TestSyncUserBackupsFromMainConfigAddsMissingServices(t *testing.T) {
	workDir := t.TempDir()
	s := NewServer(workDir)
	username := "alice"

	if err := os.MkdirAll(s.userConfigsDir(username), 0o755); err != nil {
		t.Fatalf("mkdir user configs dir: %v", err)
	}
	backupPath := s.configTemplatePath(username, "bk1")
	backupCfg := &config.Config{
		Global: config.Global{
			"user": "alice",
		},
		Nodes: config.Nodes{
			"n1": {IP: "10.0.0.10"},
		},
		ServiceTop: config.ServiceTopos{
			"zookeeper": {Nodes: []string{"n1"}},
		},
		ServerConfig: config.ServiceConfigs{
			"zookeeper": {Vars: map[string]any{"port": 2181}},
		},
	}
	if err := config.SaveConfig(backupPath, backupCfg); err != nil {
		t.Fatalf("save backup cfg: %v", err)
	}

	mainCfg := &config.Config{
		Global: config.Global{
			"user":             "root-user",
			"install_base_dir": "/opt/platform",
		},
		Nodes: config.Nodes{
			"n1": {IP: "10.0.0.10"},
			"n2": {IP: "10.0.0.11"},
		},
		ServiceTop: config.ServiceTopos{
			"zookeeper": {Nodes: []string{"n1"}},
			"hdfs":      {Nodes: []string{"n2"}},
		},
		ServerConfig: config.ServiceConfigs{
			"zookeeper": {Vars: map[string]any{"port": 2181}},
			"hdfs":      {Vars: map[string]any{"nn": "on"}},
		},
	}
	log, err := s.syncUserBackupsFromMainConfig(username, mainCfg)
	if err != nil {
		t.Fatalf("sync user backups from main: %v", err)
	}
	if log.ServerConfigAdded == 0 || log.ServiceTopAdded == 0 {
		t.Fatalf("expected backup sync to add missing services, got %+v", log)
	}
	if log.GlobalAdded == 0 {
		t.Fatalf("expected backup sync to add missing global keys, got %+v", log)
	}
	updated, err := config.LoadConfig(backupPath)
	if err != nil {
		t.Fatalf("load updated backup: %v", err)
	}
	if _, ok := updated.ServiceTop["hdfs"]; !ok {
		t.Fatalf("expected hdfs serviceTop copied to backup")
	}
	if _, ok := updated.ServerConfig["hdfs"]; !ok {
		t.Fatalf("expected hdfs serverConfig copied to backup")
	}
	if got := updated.Global["install_base_dir"]; got != "/opt/platform" {
		t.Fatalf("expected missing global key copied to backup, got: %v", got)
	}
	if got := updated.Global["user"]; got != "alice" {
		t.Fatalf("expected existing backup global key unchanged, got: %v", got)
	}
}
