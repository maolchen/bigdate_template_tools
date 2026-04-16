package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetWorkDirRespectsEnv(t *testing.T) {
	want := t.TempDir()
	t.Setenv("CONFIG_GENERATOR_WORK_DIR", want)

	got := GetWorkDir()
	if got != want {
		t.Fatalf("GetWorkDir() = %q, want %q", got, want)
	}
}

func TestHasWorkDirMarkers(t *testing.T) {
	root := t.TempDir()
	if hasWorkDirMarkers(root) {
		t.Fatalf("expected false for empty dir: %s", root)
	}

	configDir := filepath.Join(root, "with_config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("global: {}\n"), 0o644); err != nil {
		t.Fatalf("write config.yaml failed: %v", err)
	}
	if !hasWorkDirMarkers(configDir) {
		t.Fatalf("expected true when config.yaml exists")
	}

	templatesDir := filepath.Join(root, "with_templates")
	if err := os.MkdirAll(filepath.Join(templatesDir, "templates"), 0o755); err != nil {
		t.Fatalf("mkdir templates dir failed: %v", err)
	}
	if !hasWorkDirMarkers(templatesDir) {
		t.Fatalf("expected true when templates dir exists")
	}
}
