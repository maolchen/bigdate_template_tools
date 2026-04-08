package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"config-generator/config"

	"gopkg.in/yaml.v3"
)

// Server Web 服务器
type Server struct {
	configPath     string
	outputDir      string
	templatesDir   string
	webDir         string
	descsDir       string
	aiDir          string
	aiSessionsDir  string
	aiUploadsDir   string
	aiSettingsPath string
	aiRulesPath    string
	cfg            *config.Config
}

// NewServer 创建新的服务器实例
func NewServer(workDir string) *Server {
	return &Server{
		configPath:     filepath.Join(workDir, "config.yaml"),
		outputDir:      filepath.Join(workDir, "output"),
		templatesDir:   filepath.Join(workDir, "templates"),
		webDir:         filepath.Join(workDir, "web", "dist"),
		descsDir:       filepath.Join(workDir, "data", "descriptions"),
		aiDir:          filepath.Join(workDir, "data", "ai"),
		aiSessionsDir:  filepath.Join(workDir, "data", "ai", "sessions"),
		aiUploadsDir:   filepath.Join(workDir, "data", "ai", "uploads"),
		aiSettingsPath: filepath.Join(workDir, "data", "ai", "settings.json"),
		aiRulesPath:    filepath.Join(workDir, "data", "ai", "template_rules.md"),
	}
}

// loadConfig 加载配置
func (s *Server) loadConfig() error {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &s.cfg)
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
	if err := s.loadConfig(); err != nil {
		fmt.Printf("警告: 无法加载配置文件 %s: %v\n", s.configPath, err)
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

	mux := http.NewServeMux()
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
	mux.HandleFunc("/api/ai/settings", s.handleAISettings)
	mux.HandleFunc("/api/ai/settings/test", s.handleAISettingsTest)
	mux.HandleFunc("/api/ai/rules", s.handleAIRules)
	mux.HandleFunc("/api/ai/template/session", s.handleAITemplateSessionCollection)
	mux.HandleFunc("/api/ai/template/session/", s.handleAITemplateSessionDetail)

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

	if port == "" {
		port = "5000"
	}

	fmt.Printf("============================================\n")
	fmt.Printf("大数据平台配置生成器 Web 服务\n")
	fmt.Printf("============================================\n")
	fmt.Printf("配置文件: %s\n", s.configPath)
	fmt.Printf("模板目录: %s\n", s.templatesDir)
	fmt.Printf("输出目录: %s\n", s.outputDir)
	fmt.Printf("描述目录: %s\n", s.descsDir)
	fmt.Printf("AI 工作目录: %s\n", s.aiDir)
	fmt.Printf("端口: %s\n", port)
	fmt.Printf("API: http://localhost:%s/api/config\n", port)
	fmt.Printf("Web: http://localhost:%s/\n", port)
	fmt.Printf("============================================\n")

	return http.ListenAndServe(":"+port, corsMiddleware(mux))
}

// mergeDescriptions 仅同步服务级 description 到内存配置
func mergeDescriptions(serviceName string, descriptions map[string]string, cfg *config.Config) {
	if serviceName == "global" || cfg == nil {
		return
	}

	if service, ok := cfg.ServerConfig[serviceName]; ok {
		service.Description = descriptions["description"]
		cfg.ServerConfig[serviceName] = service
	}
}
