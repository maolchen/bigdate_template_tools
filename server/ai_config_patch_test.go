package server

import (
	"testing"

	"config-generator/config"
)

func TestExtractInstanceVars(t *testing.T) {
	content := `
PORT={{ .Instance.Vars.http_port }}
DATA_DIR={{ .Instance.Vars.data_subdir }}
ALT={{ index .Instance.Vars "cluster_name" }}
`

	vars := extractInstanceVars(content)
	expected := []string{"cluster_name", "data_subdir", "http_port"}
	if len(vars) != len(expected) {
		t.Fatalf("unexpected vars length: %#v", vars)
	}
	for idx, item := range expected {
		if vars[idx] != item {
			t.Fatalf("unexpected vars: %#v", vars)
		}
	}
}

func TestEnrichConfigPatchForDrafts(t *testing.T) {
	server := newTestServer(t)
	server.cfg = &config.Config{
		Nodes: config.Nodes{
			"node-a": {IP: "10.0.0.1", Hostname: "node-a"},
			"node-b": {IP: "10.0.0.2", Hostname: "node-b"},
		},
		ServiceTop:   config.ServiceTopos{},
		ServerConfig: config.ServiceConfigs{},
	}

	drafts := []aiDraftFile{{
		Path:    "templates/elasticsearch/install.sh.tmpl",
		Content: "PORT={{ .Instance.Vars.http_port }}\nDATA={{ .Instance.Vars.data_subdir }}\n",
	}}
	patch, issues := server.enrichConfigPatchForDrafts(drafts, aiConfigPatch{
		ServiceTop: map[string]config.ServiceTopo{
			"elasticsearch": {
				Nodes:        []string{"node-a", "node-b"},
				IDAutoDerive: false,
			},
		},
	})

	serviceCfg, ok := patch.ServerConfig["elasticsearch"]
	if !ok {
		t.Fatal("expected serverConfig patch for elasticsearch")
	}
	if _, exists := serviceCfg.Vars["http_port"]; !exists {
		t.Fatalf("expected inferred var http_port, got %#v", serviceCfg.Vars)
	}
	if _, exists := serviceCfg.Vars["data_subdir"]; !exists {
		t.Fatalf("expected inferred var data_subdir, got %#v", serviceCfg.Vars)
	}
	if len(issues) == 0 {
		t.Fatal("expected placeholder issues for inferred vars")
	}
}

func TestEnrichConfigPatchForDraftsRejectsUnknownNodes(t *testing.T) {
	server := newTestServer(t)
	server.cfg = &config.Config{
		Nodes: config.Nodes{
			"node-a": {IP: "10.0.0.1", Hostname: "node-a"},
		},
		ServiceTop:   config.ServiceTopos{},
		ServerConfig: config.ServiceConfigs{},
	}

	_, issues := server.enrichConfigPatchForDrafts([]aiDraftFile{{
		Path:    "templates/elasticsearch/install.sh.tmpl",
		Content: "PORT={{ .Instance.Vars.http_port }}",
	}}, aiConfigPatch{
		ServiceTop: map[string]config.ServiceTopo{
			"elasticsearch": {
				Nodes: []string{"node-x"},
			},
		},
	})

	if !hasBlockingConfigIssues(issues) {
		t.Fatalf("expected blocking issues, got %#v", issues)
	}
}

func TestApplyConfigPatchToConfig(t *testing.T) {
	cfg := &config.Config{
		ServiceTop:   config.ServiceTopos{},
		ServerConfig: config.ServiceConfigs{},
	}

	applied := applyConfigPatchToConfig(cfg, aiConfigPatch{
		ServiceTop: map[string]config.ServiceTopo{
			"elasticsearch": {
				Nodes:        []string{"node-a", "node-b"},
				IDAutoDerive: true,
			},
		},
		ServerConfig: map[string]config.ServiceConfig{
			"elasticsearch": {
				Description: "ElasticSearch cluster",
				Vars: map[string]interface{}{
					"http_port": 9200,
				},
			},
		},
	})

	if len(applied) != 1 || applied[0] != "elasticsearch" {
		t.Fatalf("unexpected applied services: %#v", applied)
	}
	if cfg.ServiceTop["elasticsearch"].IDAutoDerive != true {
		t.Fatalf("expected serviceTop to be applied: %#v", cfg.ServiceTop["elasticsearch"])
	}
	if cfg.ServerConfig["elasticsearch"].Vars["http_port"] != 9200 {
		t.Fatalf("expected serverConfig vars to be applied: %#v", cfg.ServerConfig["elasticsearch"].Vars)
	}
}
