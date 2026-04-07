package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
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

		// 保存到文件
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

	// 构建服务实例
	instances := generator.BuildServiceInstances(s.cfg)

	// 生成配置
	if err := generator.GenerateOutputs(s.cfg, instances, s.outputDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 统计生成的文件数
	fileCount := s.countGeneratedFiles()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "配置生成成功",
		"instances": len(instances),
		"files":     fileCount,
	})
}

// handleGetOutput 处理获取输出文件列表请求
func (s *Server) handleGetOutput(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var files []string

	filepath.Walk(s.outputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(s.outputDir, path)
			files = append(files, relPath)
		}
		return nil
	})

	json.NewEncoder(w).Encode(files)
}

// handleDownloadOutput 处理下载输出文件请求
func (s *Server) handleDownloadOutput(w http.ResponseWriter, r *http.Request) {
	// 创建 ZIP 文件
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

	// 安全检查
	cleanPath := filepath.Clean(path)
	fullPath := filepath.Join(s.outputDir, cleanPath)

	// 确保路径在输出目录内
	if !strings.HasPrefix(fullPath, s.outputDir) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(content)
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
				"path":        relPath,
				"service":     serviceName,
				"name":        filepath.Base(path),
				"content":     string(content),
				"size":        info.Size(),
				"modified":    info.ModTime(),
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已重新加载",
	})
}

// handleDescriptions 处理服务描述的 GET 和 POST 请求
func (s *Server) handleDescriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 从路径中提取服务名
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	serviceName := pathParts[3]

	descPath := getDescriptionPath(serviceName, s.webDir)

	switch r.Method {
	case "GET":
		// 读取描述文件
		if _, err := os.Stat(descPath); os.IsNotExist(err) {
			json.NewEncoder(w).Encode(map[string]string{})
			return
		}

		data, err := os.ReadFile(descPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var descriptions map[string]string
		if err := json.Unmarshal(data, &descriptions); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(descriptions)

	case "POST":
		var descriptions map[string]string
		if err := json.NewDecoder(r.Body).Decode(&descriptions); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 保存到文件
		data, err := json.MarshalIndent(descriptions, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(descPath, data, 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 合并到配置
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

	descPath := getDescriptionPath("global", s.webDir)

	switch r.Method {
	case "GET":
		if _, err := os.Stat(descPath); os.IsNotExist(err) {
			json.NewEncoder(w).Encode(map[string]string{})
			return
		}

		data, err := os.ReadFile(descPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var descriptions map[string]string
		if err := json.Unmarshal(data, &descriptions); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(descriptions)

	case "POST":
		var descriptions map[string]string
		if err := json.NewDecoder(r.Body).Decode(&descriptions); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		data, err := json.MarshalIndent(descriptions, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(descPath, data, 0644); err != nil {
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

	// 解析请求
	var req checker.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := checker.CheckReferences(req, s.templatesDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

// countGeneratedFiles 统计生成的文件数量
func (s *Server) countGeneratedFiles() int {
	count := 0
	filepath.Walk(s.outputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	return count
}
