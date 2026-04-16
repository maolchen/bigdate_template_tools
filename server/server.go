package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"config-generator/config"

	"gopkg.in/yaml.v3"
)

// Server Web 服务器
type Server struct {
	workDir          string
	dataDir          string
	configPath       string
	outputDir        string
	templatesDir     string
	webDir           string
	descsDir         string
	usersRootDir     string
	authDir          string
	usersFilePath    string
	sessionsFilePath string
	aiDir            string
	aiSessionsDir    string
	aiUploadsDir     string
	aiSkillsDir      string
	aiExamplesPath   string
	aiSettingsPath   string
	aiRulesPath      string
	aiClient         AIClient
	aiSessionLocks   sync.Map
	authMu           sync.Mutex
	authUsers        map[string]authUserRecord
	authSessions     map[string]authSessionRecord
	configMu         sync.Mutex
	cfg              *config.Config
}

// NewServer 创建新的服务器实例
func NewServer(workDir string) *Server {
	return &Server{
		workDir:          workDir,
		dataDir:          filepath.Join(workDir, "data"),
		configPath:       filepath.Join(workDir, "config.yaml"),
		outputDir:        filepath.Join(workDir, "output"),
		templatesDir:     filepath.Join(workDir, "templates"),
		webDir:           filepath.Join(workDir, "web", "dist"),
		descsDir:         filepath.Join(workDir, "data", "descriptions"),
		usersRootDir:     filepath.Join(workDir, "data", "users"),
		authDir:          filepath.Join(workDir, "data", "auth"),
		usersFilePath:    filepath.Join(workDir, "data", "users", "users.json"),
		sessionsFilePath: filepath.Join(workDir, "data", "auth", "sessions.json"),
		aiDir:            filepath.Join(workDir, "data", "ai"),
		aiSessionsDir:    filepath.Join(workDir, "data", "ai", "sessions"),
		aiUploadsDir:     filepath.Join(workDir, "data", "ai", "uploads"),
		aiSkillsDir:      filepath.Join(workDir, "data", "ai", "skills"),
		aiExamplesPath:   filepath.Join(workDir, "data", "ai", "examples_index.json"),
		aiSettingsPath:   filepath.Join(workDir, "data", "ai", "settings.json"),
		aiRulesPath:      filepath.Join(workDir, "data", "ai", "template_rules.md"),
		aiClient:         newDefaultAIClient(),
		authUsers:        make(map[string]authUserRecord),
		authSessions:     make(map[string]authSessionRecord),
	}
}

// SetAIClient injects a custom AI client (mainly used by tests and compatibility switches).
func (s *Server) SetAIClient(client AIClient) {
	if client == nil {
		return
	}
	s.aiClient = client
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
		origin := r.Header.Get("Origin")
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(payload)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *statusRecorder) ReadFrom(src io.Reader) (int64, error) {
	readerFrom, ok := r.ResponseWriter.(io.ReaderFrom)
	if !ok {
		return io.Copy(r.ResponseWriter, src)
	}
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := readerFrom.ReadFrom(src)
	r.bytes += int(n)
	return n, err
}

// loggingMiddleware logs method/path/status/bytes/latency for every HTTP request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path = path + "?" + r.URL.RawQuery
		}
		fmt.Printf("[HTTP] %s %s %d %dB %s\n", r.Method, path, rec.status, rec.bytes, time.Since(start))
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
	if err := s.initAuthStore(); err != nil {
		return fmt.Errorf("init auth store failed: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", s.handleAuthLogin)
	mux.HandleFunc("/api/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("/api/auth/me", s.withAuth(s.handleAuthMe))
	mux.HandleFunc("/api/auth/change-password", s.withAuth(s.handleAuthChangePassword))
	mux.HandleFunc("/api/users", s.withAdmin(s.handleUsersCollection))
	mux.HandleFunc("/api/users/", s.withAdmin(s.handleUsersDetail))

	mux.HandleFunc("/api/config", s.withAuth(s.handleConfig))
	mux.HandleFunc("/api/config/version", s.withAuth(s.handleConfigVersion))
	mux.HandleFunc("/api/config/sync-preview", s.withAuth(s.handleSyncMainPreview))
	mux.HandleFunc("/api/config/sync-main", s.withAuth(s.handleSyncMainConfig))
	mux.HandleFunc("/api/config/sync-main-delete", s.withAuth(s.handleSyncMainDelete))
	mux.HandleFunc("/api/generate", s.withAuth(s.handleGenerate))
	mux.HandleFunc("/api/output", s.withAuth(s.handleGetOutput))
	mux.HandleFunc("/api/output/download", s.withAuth(s.handleDownloadOutput))
	mux.HandleFunc("/api/output/file", s.withAuth(s.handleGetOutputFile))
	mux.HandleFunc("/api/templates", s.withAuth(s.handleGetTemplates))
	mux.HandleFunc("/api/templates/editor/tree", s.withAuth(s.handleTemplateEditorTree))
	mux.HandleFunc("/api/templates/editor/file", s.withAuth(s.handleTemplateEditorFile))
	mux.HandleFunc("/api/templates/editor/item", s.withAuth(s.handleTemplateEditorItem))
	mux.HandleFunc("/api/config/reload", s.withAuth(s.handleReloadConfig))
	mux.HandleFunc("/api/config-templates", s.withAuth(s.handleConfigTemplatesCollection))
	mux.HandleFunc("/api/config-templates/", s.withAuth(s.handleConfigTemplatesDetail))
	mux.HandleFunc("/api/descriptions/global", s.withAuth(s.handleGlobalDescriptions))
	mux.HandleFunc("/api/descriptions/", s.withAuth(s.handleDescriptions))
	mux.HandleFunc("/api/check-references", s.withAuth(s.handleCheckReferences))
	mux.HandleFunc("/api/ai/settings", s.withAuth(s.handleAISettings))
	mux.HandleFunc("/api/ai/settings/test", s.withAuth(s.handleAISettingsTest))
	mux.HandleFunc("/api/ai/models", s.withAuth(s.handleAIModels))
	mux.HandleFunc("/api/ai/rules", s.withAuth(s.handleAIRules))
	mux.HandleFunc("/api/ai/catalog", s.withAuth(s.handleAIPromptCatalog))
	mux.HandleFunc("/api/ai/skill-file", s.withAuth(s.handleAISkillFile))
	mux.HandleFunc("/api/ai/skills", s.withAuth(s.handleAICustomSkillsCollection))
	mux.HandleFunc("/api/ai/skills/", s.withAuth(s.handleAICustomSkillsDetail))
	mux.HandleFunc("/api/ai/template/session", s.withAuth(s.handleAITemplateSessionCollection))
	mux.HandleFunc("/api/ai/template/session/", s.withAuth(s.handleAITemplateSessionDetail))

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

	return http.ListenAndServe(":"+port, loggingMiddleware(corsMiddleware(mux)))
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
