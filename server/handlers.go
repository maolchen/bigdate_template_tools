package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"config-generator/checker"
	"config-generator/config"
	"config-generator/generator"
)

// handleConfig handles current user's active config get/save.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	s.configMu.Lock()
	defer s.configMu.Unlock()

	switch r.Method {
	case http.MethodGet:
		cfg, _, _, err := s.loadUserActiveConfig(principal.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.cfg = cfg
		json.NewEncoder(w).Encode(cfg)

	case http.MethodPut:
		_, activePath, activeTemplateID, err := s.loadUserActiveConfig(principal.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var newConfig config.Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := config.SaveConfig(activePath, &newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.cfg = &newConfig
		fmt.Printf("[Config] save requested user=%s template=%s nodes=%d services=%d\n", principal.Username, activeTemplateID, len(newConfig.Nodes), len(newConfig.ServiceTop))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "配置已保存",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGenerate generates outputs to current user's output directory.
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	s.configMu.Lock()
	cfg, _, activeTemplateID, err := s.loadUserActiveConfig(principal.Username)
	s.configMu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userOutputDir := s.userOutputDir(principal.Username)
	if err := os.MkdirAll(userOutputDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[Generate] start user=%s template=%s nodes=%d services=%d output=%s templates=%s\n", principal.Username, activeTemplateID, len(cfg.Nodes), len(cfg.ServiceTop), userOutputDir, s.templatesDir)
	instances := generator.BuildServiceInstances(cfg)

	summary, err := generator.GenerateOutputs(cfg, instances, userOutputDir, s.templatesDir)
	if err != nil {
		fmt.Printf("[Generate] failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[Generate] done generated=%d skipped=%d warnings=%d\n", summary.Generated, len(summary.SkippedServices), len(summary.Warnings))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"message":         "配置生成成功",
		"results":         summary.Results,
		"errors":          []string{},
		"warnings":        summary.Warnings,
		"skippedServices": summary.SkippedServices,
		"stats": map[string]int{
			"nodes":     len(cfg.Nodes),
			"services":  len(cfg.ServiceTop),
			"generated": summary.Generated,
			"skipped":   len(summary.SkippedServices),
		},
	})
}

// handleGetOutput returns file list from current user's output directory.
func (s *Server) handleGetOutput(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	userOutputDir := s.userOutputDir(principal.Username)
	filesByNode := make(map[string][]string)
	total := 0

	_ = filepath.Walk(userOutputDir, func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(userOutputDir, path)
			relPath = filepath.ToSlash(relPath)
			parts := strings.SplitN(relPath, "/", 2)
			if len(parts) == 2 {
				filesByNode[parts[0]] = append(filesByNode[parts[0]], parts[1])
			} else if len(parts) == 1 {
				filesByNode["output"] = append(filesByNode["output"], parts[0])
			}
			total++
		}
		return nil
	})

	for nodeIP := range filesByNode {
		sort.Strings(filesByNode[nodeIP])
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": filesByNode,
		"total": total,
	})
}

// handleDownloadOutput zips current user's output directory.
func (s *Server) handleDownloadOutput(w http.ResponseWriter, r *http.Request) {
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	userOutputDir := s.userOutputDir(principal.Username)
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	_ = filepath.Walk(userOutputDir, func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(userOutputDir, path)
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = writer.Write(content)
		return err
	})

	_ = zipWriter.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=output.zip")
	_, _ = w.Write(buf.Bytes())
}

// handleGetOutputFile returns a single file in current user's output directory.
func (s *Server) handleGetOutputFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	userOutputDir := s.userOutputDir(principal.Username)
	cleanPath := filepath.Clean(path)
	fullPath := filepath.Clean(filepath.Join(userOutputDir, cleanPath))
	absRoot, _ := filepath.Abs(userOutputDir)
	absPath, _ := filepath.Abs(fullPath)
	if absPath != absRoot && !strings.HasPrefix(absPath, absRoot+string(os.PathSeparator)) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"path":    filepath.ToSlash(cleanPath),
		"content": string(content),
	})
}

// handleGetTemplates returns template list from templates directory.
func (s *Server) handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var templates []map[string]interface{}

	_ = filepath.Walk(s.templatesDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tmpl") {
			relPath, _ := filepath.Rel(s.templatesDir, path)
			content, _ := os.ReadFile(path)

			serviceName := filepath.Dir(relPath)
			if serviceName == "." {
				serviceName = strings.TrimSuffix(filepath.Base(path), ".tmpl")
			}

			templates = append(templates, map[string]interface{}{
				"path":     relPath,
				"service":  serviceName,
				"name":     filepath.Base(path),
				"content":  string(content),
				"size":     info.Size(),
				"modified": info.ModTime(),
			})
		}
		return nil
	})

	json.NewEncoder(w).Encode(templates)
}

// handleReloadConfig reloads current user's active config from file.
func (s *Server) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	s.configMu.Lock()
	cfg, _, _, err := s.loadUserActiveConfig(principal.Username)
	s.configMu.Unlock()
	if err != nil {
		fmt.Printf("[Config] reload failed user=%s: %v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.cfg = cfg
	fmt.Printf("[Config] reload ok user=%s nodes=%d services=%d\n", principal.Username, len(cfg.Nodes), len(cfg.ServiceTop))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已重新加载",
		"config":  cfg,
	})
}

// handleDescriptions handles service description get/save.
func (s *Server) handleDescriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	escapedServiceName := strings.TrimPrefix(r.URL.EscapedPath(), "/api/descriptions/")
	if escapedServiceName == "" || escapedServiceName == r.URL.EscapedPath() {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	serviceName, err := url.PathUnescape(escapedServiceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg, _, _, err := s.loadUserActiveConfig(principal.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		descriptions, err := s.getFilteredDescriptions(serviceName, cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(descriptions)

	case http.MethodPost:
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := s.saveDescriptionUpdates(serviceName, updates, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "描述已保存",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGlobalDescriptions handles global description get/save.
func (s *Server) handleGlobalDescriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	cfg, _, _, err := s.loadUserActiveConfig(principal.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		descriptions, err := s.getFilteredDescriptions("global", cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(descriptions)

	case http.MethodPost:
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := s.saveDescriptionUpdates("global", updates, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "全局描述已保存",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCheckReferences performs template reference checking.
func (s *Server) handleCheckReferences(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req checker.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("[CheckReferences] type=%s key=%s service=%s\n", req.Type, req.Key, req.Service)

	result, err := checker.CheckReferences(req, s.templatesDir)
	if err != nil {
		fmt.Printf("[CheckReferences] failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[CheckReferences] done hasReferences=%v refs=%d\n", result.HasReferences, len(result.References))

	json.NewEncoder(w).Encode(result)
}
