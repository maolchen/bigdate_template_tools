package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"config-generator/config"
)

// Server Web 服务器
type Server struct {
	configPath   string
	outputDir    string
	templatesDir string
	webDir       string
	cfg          *config.Config
}

// NewServer 创建新的服务器实例
func NewServer(workDir string) *Server {
	return &Server{
		configPath:   filepath.Join(workDir, "config.yaml"),
		outputDir:    filepath.Join(workDir, "output"),
		templatesDir: filepath.Join(workDir, "templates"),
		webDir:       filepath.Join(workDir, "web", "dist"),
	}
}

// loadConfig 加载配置
func (s *Server) loadConfig() error {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.cfg)
}

// corsMiddleware CORS 中间件
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

// Start 启动 Web 服务器
func (s *Server) Start(port string) error {
	// 尝试加载配置
	if err := s.loadConfig(); err != nil {
		fmt.Printf("警告: 无法加载配置文件 %s: %v\n", s.configPath, err)
		// 使用默认配置
		s.cfg = &config.Config{
			Global: config.Global{
				"user":             "bigdata",
				"group":            "bigdata",
				"install_base_dir": "/data/localization",
				"data_base_dir":    "/data",
				"java_home":        "/data/jdk",
			},
			Nodes:        make(config.Nodes),
			ServiceTop:   make(config.ServiceTopos),
			ServerConfig: make(config.ServiceConfigs),
		}
	}

	// 创建路由
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/generate", s.handleGenerate)
	mux.HandleFunc("/api/output", s.handleGetOutput)
	mux.HandleFunc("/api/output/download", s.handleDownloadOutput)
	mux.HandleFunc("/api/output/file", s.handleGetOutputFile)
	mux.HandleFunc("/api/templates", s.handleGetTemplates)
	mux.HandleFunc("/api/config/reload", s.handleReloadConfig)
	mux.HandleFunc("/api/descriptions/global", s.handleGlobalDescriptions)
	mux.HandleFunc("/api/descriptions/", s.handleDescriptions)
	mux.HandleFunc("/api/check-references", s.handleCheckReferences)

	// 静态文件服务（前端）
	if info, err := os.Stat(s.webDir); err == nil && info.IsDir() {
		fs := http.FileServer(http.Dir(s.webDir))
		mux.Handle("/", fs)
		fmt.Printf("✓ 前端静态文件目录: %s\n", s.webDir)
		if files, err := os.ReadDir(s.webDir); err == nil {
			fmt.Printf("  文件数: %d\n", len(files))
		}
	} else {
		fmt.Printf("✗ 前端静态文件目录不存在: %s\n", s.webDir)
	}

	// 启动服务器
	if port == "" {
		port = "5000"
	}

	fmt.Printf("============================================\n")
	fmt.Printf("大数据平台配置生成器 Web 服务\n")
	fmt.Printf("============================================\n")
	fmt.Printf("配置文件: %s\n", s.configPath)
	fmt.Printf("模板目录: %s\n", s.templatesDir)
	fmt.Printf("输出目录: %s\n", s.outputDir)
	fmt.Printf("端口: %s\n", port)
	fmt.Printf("API: http://localhost:%s/api/config\n", port)
	fmt.Printf("Web: http://localhost:%s/\n", port)
	fmt.Printf("============================================\n")

	return http.ListenAndServe(":"+port, corsMiddleware(mux))
}

// 获取描述文件路径
func getDescriptionPath(serviceName string, webDir string) string {
	if serviceName == "global" {
		return filepath.Join(webDir, "..", "src", "api", "global_descriptions.json")
	}
	return filepath.Join(webDir, "..", "src", "api", "service_descriptions", serviceName+".json")
}

// 合并描述到配置
func mergeDescriptions(serviceName string, descriptions map[string]string, cfg *config.Config) {
	if serviceName == "global" {
		// 全局描述暂时不存储到配置中
		return
	}

	if service, ok := cfg.ServerConfig[serviceName]; ok {
		service.Description = descriptions["description"]
		cfg.ServerConfig[serviceName] = service
	}
}
