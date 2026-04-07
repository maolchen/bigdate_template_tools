package web

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	bigdata "bigdata-deploy-generator/config"
	"bigdata-deploy-generator/generator"
	"gopkg.in/yaml.v3"
)

// WebServer 管理 Web 模式的配置和状态
type WebServer struct {
	ConfigPath   string
	Config       *bigdata.Config
	OutputDir    string
	TemplatesDir string
	WebDir       string

	// 检查节点是否有服务分配的回调函数
	CheckNodeUsage func(nodeName string) (bool, []string)
}

// NewWebServer 创建新的 WebServer 实例
func NewWebServer(configPath, outputDir, templatesDir, webDir string) *WebServer {
	return &WebServer{
		ConfigPath:   configPath,
		OutputDir:    outputDir,
		TemplatesDir: templatesDir,
		WebDir:       webDir,
	}
}

// LoadConfig 加载配置文件
func (ws *WebServer) LoadConfig() error {
	data, err := os.ReadFile(ws.ConfigPath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &ws.Config)
}

// CountGeneratedFiles 统计生成的文件数量
func (ws *WebServer) CountGeneratedFiles() int {
	count := 0
	filepath.Walk(ws.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// RegisterRoutes 注册所有路由
func (ws *WebServer) RegisterRoutes(mux *http.ServeMux) {
	// 配置管理
	mux.HandleFunc("/api/config", ws.ConfigHandler)
	mux.HandleFunc("/api/config/reload", ws.ReloadConfigHandler)

	// 生成和输出
	mux.HandleFunc("/api/generate", ws.GenerateConfigHandler)
	mux.HandleFunc("/api/output", ws.GetOutputHandler)
	mux.HandleFunc("/api/output/download", ws.DownloadOutputHandler)
	mux.HandleFunc("/api/output/file", ws.GetOutputFileHandler)

	// 模板管理
	mux.HandleFunc("/api/templates", ws.GetTemplatesHandler)

	// 描述管理
	mux.HandleFunc("/api/descriptions/", ws.DescriptionsHandler)
	mux.HandleFunc("/api/descriptions/global", ws.GlobalDescriptionsHandler)

	// 引用检查
	mux.HandleFunc("/api/check-references", func(w http.ResponseWriter, r *http.Request) {
		CheckReferencesHandler(ws.TemplatesDir, ws.CheckNodeUsage, w, r)
	})

	// 静态文件服务
	mux.HandleFunc("/", ws.StaticFileHandler)
}

// GET /api/config - 获取当前配置
func (ws *WebServer) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		json.NewEncoder(w).Encode(ws.Config)
	} else if r.Method == "PUT" {
		ws.SaveConfigHandler(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// PUT /api/config - 保存配置
func (ws *WebServer) SaveConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var newConfig bigdata.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws.Config = &newConfig

	// 保存到文件
	data, err := yaml.Marshal(ws.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(ws.ConfigPath, data, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已保存",
	})
}

// POST /api/generate - 生成配置
func (ws *WebServer) GenerateConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 构建服务实例
	instances := generator.BuildServiceInstances(ws.Config)

	// 清空输出目录
	os.RemoveAll(ws.OutputDir)
	os.MkdirAll(ws.OutputDir, 0755)

	// 生成输出
	if err := generator.GenerateOutputs(ws.Config, instances, ws.OutputDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 统计结果
	stats := map[string]interface{}{
		"nodes":    len(ws.Config.Nodes),
		"services": len(ws.Config.ServiceTop),
		"generated": ws.CountGeneratedFiles(),
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置生成成功",
		"results": instances,
		"stats":   stats,
	})
}

// GET /api/output - 获取生成的文件列表
func (ws *WebServer) GetOutputHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	files := make(map[string][]string)
	total := 0

	filepath.Walk(ws.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(ws.OutputDir, path)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) >= 2 {
			ip := parts[0]
			files[ip] = append(files[ip], relPath)
			total++
		}
		return nil
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": files,
		"total": total,
	})
}

// GET /api/output/download - 下载生成的配置包
func (ws *WebServer) DownloadOutputHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=configs.zip")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	filepath.Walk(ws.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(ws.OutputDir, path)
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = writer.Write(data)
		return err
	})
}

// GET /api/output/file - 获取单个文件内容
func (ws *WebServer) GetOutputFileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path参数缺失", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(ws.OutputDir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":    path,
		"content": string(content),
	})
}

// GET /api/templates - 获取模板列表
func (ws *WebServer) GetTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var templates []map[string]string

	filepath.Walk(ws.TemplatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		relPath, _ := filepath.Rel(ws.TemplatesDir, path)
		serviceName := filepath.Dir(relPath)
		if serviceName == "." {
			serviceName = strings.TrimSuffix(filepath.Base(path), ".tmpl")
		}

		templates = append(templates, map[string]string{
			"path":    relPath,
			"service": serviceName,
			"name":    filepath.Base(path),
		})
		return nil
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
	})
}

// POST /api/config/reload - 重新加载配置文件
func (ws *WebServer) ReloadConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := ws.LoadConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已重新加载",
		"config":  ws.Config,
	})
}

// POST /api/descriptions/{serviceName} - 保存配置项说明
func (ws *WebServer) DescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 提取服务名
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	serviceName := pathParts[3]

	if r.Method == "POST" {
		ws.SaveDescriptionsHandler(w, r, serviceName)
	} else if r.Method == "GET" {
		ws.GetDescriptionsHandler(w, r, serviceName)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 保存配置项说明
func (ws *WebServer) SaveDescriptionsHandler(w http.ResponseWriter, r *http.Request, serviceName string) {
	var descriptions map[string]string
	if err := json.NewDecoder(r.Body).Decode(&descriptions); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 保存到文件
	descPath := filepath.Join(ws.TemplatesDir, serviceName, "_descriptions.yaml")
	data, err := yaml.Marshal(descriptions)
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
		"message": "说明已保存",
	})
}

// 获取配置项说明
func (ws *WebServer) GetDescriptionsHandler(w http.ResponseWriter, r *http.Request, serviceName string) {
	descPath := filepath.Join(ws.TemplatesDir, serviceName, "_descriptions.yaml")

	var descriptions map[string]string
	if data, err := os.ReadFile(descPath); err == nil {
		yaml.Unmarshal(data, &descriptions)
	} else {
		descriptions = make(map[string]string)
	}

	json.NewEncoder(w).Encode(descriptions)
}

// POST /api/descriptions/global - 保存全局配置说明
func (ws *WebServer) GlobalDescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "POST" {
		ws.SaveGlobalDescriptionsHandler(w, r)
	} else if r.Method == "GET" {
		ws.GetGlobalDescriptionsHandler(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 保存全局配置说明
func (ws *WebServer) SaveGlobalDescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	var descriptions map[string]string
	if err := json.NewDecoder(r.Body).Decode(&descriptions); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 保存到文件
	descPath := filepath.Join(ws.TemplatesDir, "_global_descriptions.yaml")
	data, err := yaml.Marshal(descriptions)
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
		"message": "说明已保存",
	})
}

// 获取全局配置说明
func (ws *WebServer) GetGlobalDescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	descPath := filepath.Join(ws.TemplatesDir, "_global_descriptions.yaml")

	var descriptions map[string]string
	if data, err := os.ReadFile(descPath); err == nil {
		yaml.Unmarshal(data, &descriptions)
	} else {
		descriptions = make(map[string]string)
	}

	json.NewEncoder(w).Encode(descriptions)
}

// 静态文件服务
func (ws *WebServer) StaticFileHandler(w http.ResponseWriter, r *http.Request) {
	// 如果是 API 请求，返回 404
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	// 检查请求的文件是否存在
	filePath := filepath.Join(ws.WebDir, r.URL.Path)
	if r.URL.Path == "/" {
		filePath = filepath.Join(ws.WebDir, "index.html")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 文件不存在，返回 index.html（SPA 路由）
		filePath = filepath.Join(ws.WebDir, "index.html")
	}

	http.ServeFile(w, r, filePath)
}
