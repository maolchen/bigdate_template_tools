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

	"gopkg.in/yaml.v3"
)

// handleConfig 处理配置的 GET 和 PUT 请求
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(s.cfg)

	case "PUT":
		var newConfig config.Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.cfg = &newConfig
		fmt.Printf("[Config] save requested: nodes=%d services=%d\n", len(s.cfg.Nodes), len(s.cfg.ServiceTop))

		data, err := yaml.Marshal(s.cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(s.configPath, data, 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "配置已保存",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGenerate 处理配置生成请求
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Printf("[Generate] start nodes=%d services=%d output=%s templates=%s\n", len(s.cfg.Nodes), len(s.cfg.ServiceTop), s.outputDir, s.templatesDir)
	instances := generator.BuildServiceInstances(s.cfg)

	summary, err := generator.GenerateOutputs(s.cfg, instances, s.outputDir, s.templatesDir)
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
			"nodes":     len(s.cfg.Nodes),
			"services":  len(s.cfg.ServiceTop),
			"generated": summary.Generated,
			"skipped":   len(summary.SkippedServices),
		},
	})
}

// handleGetOutput 处理获取输出文件列表请求
func (s *Server) handleGetOutput(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	filesByNode := make(map[string][]string)
	total := 0

	filepath.Walk(s.outputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(s.outputDir, path)
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

// handleDownloadOutput 处理下载输出文件请求
func (s *Server) handleDownloadOutput(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	filepath.Walk(s.outputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(s.outputDir, path)
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

	zipWriter.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=output.zip")
	w.Write(buf.Bytes())
}

// handleGetOutputFile 处理获取单个输出文件请求
func (s *Server) handleGetOutputFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	cleanPath := filepath.Clean(path)
	fullPath := filepath.Join(s.outputDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.outputDir) {
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

// handleGetTemplates 处理获取模板列表请求
func (s *Server) handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var templates []map[string]interface{}

	filepath.Walk(s.templatesDir, func(path string, info fs.FileInfo, err error) error {
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

// handleReloadConfig 处理重新加载配置请求
func (s *Server) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.loadConfig(); err != nil {
		fmt.Printf("[Config] reload failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[Config] reload ok: nodes=%d services=%d\n", len(s.cfg.Nodes), len(s.cfg.ServiceTop))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已重新加载",
		"config":  s.cfg,
	})
}

// handleDescriptions 处理服务描述的 GET 和 POST 请求
func (s *Server) handleDescriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	switch r.Method {
	case "GET":
		descriptions, err := s.getFilteredDescriptions(serviceName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(descriptions)

	case "POST":
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		descriptions, err := s.saveDescriptionUpdates(serviceName, updates)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		mergeDescriptions(serviceName, descriptions, s.cfg)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "描述已保存",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGlobalDescriptions 处理全局描述的 GET 和 POST 请求
func (s *Server) handleGlobalDescriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		descriptions, err := s.getFilteredDescriptions("global")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(descriptions)

	case "POST":
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if _, err := s.saveDescriptionUpdates("global", updates); err != nil {
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

// handleCheckReferences 处理引用检查请求
func (s *Server) handleCheckReferences(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
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
