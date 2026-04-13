package server

import (
	"testing"

	"config-generator/config"
)

func TestSyncMissingServicesFromMainAddsOnlyMissingAndFiltersNodes(t *testing.T) {
	mainCfg := &config.Config{
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
