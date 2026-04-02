package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// ============================================================
// 数据结构定义
// ============================================================

type GlobalConfig struct {
	User           string `yaml:"user" json:"user"`
	Group          string `yaml:"group" json:"group"`
	Version        string `yaml:"version" json:"version"`
	PkgBaseDir     string `yaml:"pkg_base_dir" json:"pkg_base_dir"`
	InstallBaseDir string `yaml:"install_base_dir" json:"install_base_dir"`
	DataBaseDir    string `yaml:"data_base_dir" json:"data_base_dir"`
	SoftwareDir    string `yaml:"software_dir" json:"software_dir"`
	LogBaseDir     string `yaml:"log_base_dir" json:"log_base_dir"`
	JavaHome       string `yaml:"java_home" json:"java_home"`
}

type NodeInfo struct {
	IP       string `yaml:"ip" json:"ip"`
	Hostname string `yaml:"hostname" json:"hostname"`
}

type ServiceTopoItem struct {
	Nodes       []string               `yaml:"nodes" json:"nodes"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Vars        map[string]interface{} `yaml:"vars,omitempty" json:"vars,omitempty"`
}

type ServiceConfigItem struct {
	Type        string                 `yaml:"type,omitempty" json:"type,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Vars        map[string]interface{} `yaml:"vars,omitempty" json:"vars,omitempty"`
}

type Config struct {
	Global       GlobalConfig                 `yaml:"global" json:"global"`
	Nodes        map[string]NodeInfo          `yaml:"nodes" json:"nodes"`
	ServiceTop   map[string]ServiceTopoItem   `yaml:"serviceTop" json:"serviceTop"`
	ServerConfig map[string]ServiceConfigItem `yaml:"serverConfig" json:"serverConfig"`
}

// ============================================================
// 全局变量
// ============================================================

var (
	configPath   string // 将在 main 中初始化
	config       Config
	outputDir    string // 将在 main 中初始化
	templatesDir string // 将在 main 中初始化
	webDir       string // 将在 main 中初始化
)

// ============================================================
// API Handlers
// ============================================================

// GET /api/config - 获取当前配置
func getConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// PUT /api/config - 保存配置
func saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var newConfig Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config = newConfig

	// 保存到文件
	data, err := yaml.Marshal(&config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已保存",
	})
}

// POST /api/generate - 生成配置
func generateConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 清理输出目录
	os.RemoveAll(outputDir)
	os.MkdirAll(outputDir, 0755)

	results := make(map[string][]string)
	errors := []string{}

	// 遍历服务拓扑
	for serviceName, topo := range config.ServiceTop {
		// 确定模板路径
		templatePath := filepath.Join(templatesDir, serviceName)

		// 检查模板目录是否存在
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			// 尝试扁平化路径
			flatPath := filepath.Join(templatesDir, strings.ReplaceAll(serviceName, "/", string(filepath.Separator)))
			if _, err := os.Stat(flatPath); os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf("服务 %s 没有对应的模板目录", serviceName))
				continue
			}
			templatePath = flatPath
		}

		// 遍历节点
		for _, nodeName := range topo.Nodes {
			node, exists := config.Nodes[nodeName]
			if !exists {
				errors = append(errors, fmt.Sprintf("节点 %s 不存在", nodeName))
				continue
			}

			// 创建输出目录
			nodeOutputDir := filepath.Join(outputDir, node.IP, serviceName)
			os.MkdirAll(nodeOutputDir, 0755)

			// 渲染模板
			generatedFiles, err := renderTemplates(templatePath, nodeOutputDir, serviceName, nodeName, node, topo)
			if err != nil {
				errors = append(errors, fmt.Sprintf("渲染 %s@%s 失败: %v", serviceName, nodeName, err))
				continue
			}

			results[node.IP] = append(results[node.IP], generatedFiles...)
		}
	}

	response := map[string]interface{}{
		"success": len(errors) == 0,
		"message": "配置生成完成",
		"results": results,
		"errors":  errors,
		"stats": map[string]int{
			"nodes":     len(config.Nodes),
			"services":  len(config.ServiceTop),
			"generated": countGeneratedFiles(),
		},
	}

	json.NewEncoder(w).Encode(response)
}

// GET /api/output - 获取生成的文件列表
func getOutputHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	files := make(map[string][]string)

	filepath.Walk(outputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(outputDir, path)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) >= 2 {
			ip := parts[0]
			files[ip] = append(files[ip], strings.Join(parts[1:], "/"))
		}
		return nil
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": files,
		"total": countGeneratedFiles(),
	})
}

// GET /api/output/download - 下载生成的配置包
func downloadOutputHandler(w http.ResponseWriter, r *http.Request) {
	// 创建 zip 文件
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	filepath.Walk(outputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(outputDir, path)
		zipFile, _ := zipWriter.Create(relPath)

		fileContent, _ := os.ReadFile(path)
		zipFile.Write(fileContent)

		return nil
	})

	zipWriter.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=output.zip")
	w.Write(buf.Bytes())
}

// GET /api/output/file?path=xxx - 获取单个文件内容
func getOutputFileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "缺少 path 参数", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(outputDir, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":    filePath,
		"content": string(content),
	})
}

// GET /api/templates - 获取模板列表
func getTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	templates := []map[string]interface{}{}

	filepath.Walk(templatesDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if strings.HasSuffix(path, ".tmpl") {
			relPath, _ := filepath.Rel(templatesDir, path)
			serviceName := filepath.Dir(relPath)
			if serviceName == "." {
				serviceName = strings.TrimSuffix(filepath.Base(path), ".tmpl")
			}

			templates = append(templates, map[string]interface{}{
				"path":    relPath,
				"service": serviceName,
				"name":    filepath.Base(path),
			})
		}
		return nil
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
	})
}

// POST /api/config/reload - 从文件重新加载配置
func reloadConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := loadConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已重新加载",
		"config":  config,
	})
}

// ============================================================
// 模板渲染
// ============================================================

func renderTemplates(templatePath, outputDir, serviceName, nodeName string, node NodeInfo, topo ServiceTopoItem) ([]string, error) {
	generatedFiles := []string{}

	// 遍历模板目录
	err := filepath.Walk(templatePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// 读取模板
		tmplContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// 创建模板
		tmpl, err := template.New(filepath.Base(path)).Parse(string(tmplContent))
		if err != nil {
			return err
		}

		// 准备模板数据
		data := map[string]interface{}{
			"Global":   config.Global,
			"Node":     map[string]interface{}{"Name": nodeName, "IP": node.IP, "Hostname": node.Hostname},
			"Instance": map[string]interface{}{"Vars": mergeVars(serviceName, topo.Vars)},
			"Config":   config,
		}

		// 渲染模板
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return err
		}

		// 输出文件名（移除 .tmpl 后缀）
		outputName := strings.TrimSuffix(filepath.Base(path), ".tmpl")
		outputPath := filepath.Join(outputDir, outputName)

		// 确保目录存在
		os.MkdirAll(filepath.Dir(outputPath), 0755)

		// 写入文件
		if err := os.WriteFile(outputPath, buf.Bytes(), 0755); err != nil {
			return err
		}

		generatedFiles = append(generatedFiles, filepath.Join(serviceName, outputName))
		return nil
	})

	return generatedFiles, err
}

func mergeVars(serviceName string, topoVars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 从 serverConfig 获取变量
	if svcConfig, exists := config.ServerConfig[serviceName]; exists {
		for k, v := range svcConfig.Vars {
			result[k] = v
		}
	}

	// 从 topoVars 获取变量（覆盖）
	for k, v := range topoVars {
		result[k] = v
	}

	return result
}

func countGeneratedFiles() int {
	count := 0
	filepath.Walk(outputDir, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// ============================================================
// 主函数
// ============================================================

func loadConfig() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &config)
}

// 获取工作目录（server 目录的父目录）
func getWorkDir() string {
	// 尝试获取可执行文件所在目录
	execPath, err := os.Executable()
	if err == nil {
		// 如果在 server 目录下运行
		serverDir := filepath.Dir(execPath)
		if filepath.Base(serverDir) == "server" {
			return filepath.Dir(serverDir)
		}
	}

	// 尝试当前目录
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == "server" {
		return filepath.Dir(cwd)
	}

	// 假设在项目根目录
	return cwd
}

func main() {
	// 确定工作目录
	workDir := getWorkDir()

	// 设置路径
	configPath = filepath.Join(workDir, "config.yaml")
	outputDir = filepath.Join(workDir, "output")
	templatesDir = filepath.Join(workDir, "templates")
	webDir = filepath.Join(workDir, "web", "dist")

	// 如果 web/dist 不存在，尝试 server/web/dist
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		webDir = filepath.Join(workDir, "server", "web", "dist")
	}

	// 加载配置
	if err := loadConfig(); err != nil {
		fmt.Printf("警告: 无法加载配置文件 %s: %v\n", configPath, err)
		// 使用默认配置
		config = Config{
			Global: GlobalConfig{
				User:           "bigdata",
				Group:          "bigdata",
				InstallBaseDir: "/data/localization",
				DataBaseDir:    "/data",
				JavaHome:       "/data/jdk",
			},
			Nodes:        make(map[string]NodeInfo),
			ServiceTop:   make(map[string]ServiceTopoItem),
			ServerConfig: make(map[string]ServiceConfigItem),
		}
	}

	// 创建路由
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getConfigHandler(w, r)
		case "PUT":
			saveConfigHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		generateConfigHandler(w, r)
	})

	mux.HandleFunc("/api/output", getOutputHandler)
	mux.HandleFunc("/api/output/download", downloadOutputHandler)
	mux.HandleFunc("/api/output/file", getOutputFileHandler)
	mux.HandleFunc("/api/templates", getTemplatesHandler)
	mux.HandleFunc("/api/config/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		reloadConfigHandler(w, r)
	})

	// 静态文件服务（前端）
	if _, err := os.Stat(webDir); err == nil {
		fs := http.FileServer(http.Dir(webDir))
		mux.Handle("/", fs)
		fmt.Printf("前端静态文件目录: %s\n", webDir)
	} else {
		fmt.Printf("警告: 前端静态文件目录不存在: %s\n", webDir)
	}

	// 启动服务器
	port := os.Getenv("DEPLOY_RUN_PORT")
	if port == "" {
		port = "5000"
	}

	fmt.Printf("============================================\n")
	fmt.Printf("大数据平台配置生成器 API 服务\n")
	fmt.Printf("============================================\n")
	fmt.Printf("工作目录: %s\n", workDir)
	fmt.Printf("配置文件: %s\n", configPath)
	fmt.Printf("模板目录: %s\n", templatesDir)
	fmt.Printf("输出目录: %s\n", outputDir)
	fmt.Printf("端口: %s\n", port)
	fmt.Printf("API: http://localhost:%s/api/config\n", port)
	fmt.Printf("Web: http://localhost:%s/\n", port)
	fmt.Printf("============================================\n")

	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		fmt.Printf("启动服务失败: %v\n", err)
		os.Exit(1)
	}
}

// CORS 中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
