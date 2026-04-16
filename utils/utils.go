package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetNestedPort extracts an integer port from nested maps.
// Supported format: "zookeeper.client_port" or "server.port".
func GetNestedPort(vars map[string]interface{}, fieldPath string) int {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 1 {
		if p, ok := vars[parts[0]].(int); ok {
			return p
		}
		return 0
	}

	current := vars
	for i, part := range parts {
		if i == len(parts)-1 {
			if p, ok := current[part].(int); ok {
				return p
			}
			return 0
		}
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return 0
		}
		current = next
	}
	return 0
}

// FormatID formats service instance IDs by template string.
// Supported placeholders: {index}, {index+1}.
func FormatID(format string, index int) interface{} {
	indexPlusOne := index + 1

	result := strings.ReplaceAll(format, "{index}", fmt.Sprintf("%d", index))
	result = strings.ReplaceAll(result, "{index+1}", fmt.Sprintf("%d", indexPlusOne))

	if result == "index" {
		return index
	}
	if result == "index+1" {
		return indexPlusOne
	}
	return result
}

// GetWorkDir returns project runtime root.
// Priority:
// 1) CONFIG_GENERATOR_WORK_DIR
// 2) current working directory if it already has config/templates markers
// 3) executable directory if it has markers
// 4) fallback cwd/executable directory
func GetWorkDir() string {
	if workDir := strings.TrimSpace(os.Getenv("CONFIG_GENERATOR_WORK_DIR")); workDir != "" {
		return workDir
	}

	if cwd, err := os.Getwd(); err == nil && hasWorkDirMarkers(cwd) {
		return cwd
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		if hasWorkDirMarkers(exeDir) {
			return exeDir
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	if exePath, err := os.Executable(); err == nil {
		return filepath.Dir(exePath)
	}

	return "."
}

func hasWorkDirMarkers(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	configPath := filepath.Join(dir, "config.yaml")
	if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
		return true
	}
	templatesPath := filepath.Join(dir, "templates")
	if info, err := os.Stat(templatesPath); err == nil && info.IsDir() {
		return true
	}
	return false
}
