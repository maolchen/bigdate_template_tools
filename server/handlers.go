package server

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"config-generator/checker"
	"config-generator/config"
	"config-generator/generator"
)

type rootConfigVersionResponse struct {
	Path          string `json:"path"`
	MTime         string `json:"mtime"`
	MTimeUnixNano int64  `json:"mtimeUnixNano"`
	Size          int64  `json:"size"`
	SHA256        string `json:"sha256"`
}

type syncMainConfigResponse struct {
	Success bool                      `json:"success"`
	Message string                    `json:"message"`
	Sync    userConfigSyncLog         `json:"sync"`
	Config  *config.Config            `json:"config"`
	Version rootConfigVersionResponse `json:"version"`
}

type syncPreviewResponse struct {
	Success bool              `json:"success"`
	Sync    userConfigSyncLog `json:"sync"`
}

type templateEditorTreeNode struct {
	Name     string                   `json:"name"`
	Path     string                   `json:"path"`
	Type     string                   `json:"type"`
	Children []templateEditorTreeNode `json:"children,omitempty"`
}

type templateEditorFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type templateEditorItemRequest struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

type templateEditorFileResponse struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Readonly  bool   `json:"readonly"`
	UpdatedAt string `json:"updatedAt"`
}

// readRootConfigVersion returns root config.yaml version metadata for polling.
func (s *Server) readRootConfigVersion() (rootConfigVersionResponse, error) {
	info, err := os.Stat(s.configPath)
	if err != nil {
		return rootConfigVersionResponse{}, err
	}
	content, err := os.ReadFile(s.configPath)
	if err != nil {
		return rootConfigVersionResponse{}, err
	}
	sum := sha256.Sum256(content)
	return rootConfigVersionResponse{
		Path:          s.configPath,
		MTime:         info.ModTime().UTC().Format(time.RFC3339Nano),
		MTimeUnixNano: info.ModTime().UnixNano(),
		Size:          info.Size(),
		SHA256:        hex.EncodeToString(sum[:]),
	}, nil
}

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
		cfg, _, _, err := s.loadConfigForPrincipal(principal)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.cfg = cfg
		json.NewEncoder(w).Encode(cfg)

	case http.MethodPut:
		_, activePath, activeTemplateID, err := s.loadConfigForPrincipal(principal)
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

		if principal.Role == roleAdmin {
			userCount, syncLog := s.syncRootConfigToAllUsers(&newConfig)
			fmt.Printf("[Config] admin saved root and synced users=%d %s\n", userCount, formatSyncLog(syncLog))
		} else {
			backupSync, err := s.syncUserBackupsFromMainConfig(principal.Username, &newConfig)
			if err != nil {
				fmt.Printf("[Config] user backup sync failed user=%s err=%v\n", principal.Username, err)
			} else if _, err := s.refreshUserTemplateIndex(principal.Username, userMainTemplateID, nil); err != nil {
				fmt.Printf("[Config] refresh template index after backup sync failed user=%s err=%v\n", principal.Username, err)
			} else {
				fmt.Printf("[Config] user backup sync done user=%s %s\n", principal.Username, formatSyncLog(backupSync))
			}
		}

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
	cfg, _, activeTemplateID, err := s.loadConfigForPrincipal(principal)
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
	cfg, _, _, err := s.loadConfigForPrincipal(principal)
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

// handleConfigVersion returns root config.yaml version for client-side polling.
func (s *Server) handleConfigVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	version, err := s.readRootConfigVersion()
	if err != nil {
		fmt.Printf("[Config] version failed user=%s path=%s err=%v\n", principal.Username, s.configPath, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[Config] version user=%s mtime=%s size=%d\n", principal.Username, version.MTime, version.Size)
	json.NewEncoder(w).Encode(version)
}

// handleSyncMainPreview returns a non-destructive diff summary between root config
// and current principal config. It is used by the frontend modal to present
// concrete add/remove changes before the user decides whether to apply delete sync.
func (s *Server) handleSyncMainPreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	s.configMu.Lock()
	defer s.configMu.Unlock()

	rootCfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		fmt.Printf("[Config] sync-preview load root failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cfg, _, _, err := s.loadConfigForPrincipal(principal)
	if err != nil {
		fmt.Printf("[Config] sync-preview load current failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	diff := diffConfigChangesFromMain(rootCfg, cfg)
	fmt.Printf("[Config] sync-preview user=%s %s\n", principal.Username, formatSyncLog(diff))
	json.NewEncoder(w).Encode(syncPreviewResponse{
		Success: true,
		Sync:    diff,
	})
}

// handleSyncMainConfig synchronizes root config incremental additions.
// For admin: propagate root additions to all non-admin user workspaces.
// For normal user: synchronize root additions into current user workspace.
func (s *Server) handleSyncMainConfig(w http.ResponseWriter, r *http.Request) {
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
	rootCfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		s.configMu.Unlock()
		fmt.Printf("[Config] sync-main load root failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	syncLog := newUserConfigSyncLog()
	var cfg *config.Config
	activePath := s.configPath
	message := "main config sync finished"

	if principal.Role == roleAdmin {
		userCount, total := s.syncRootConfigToAllUsers(rootCfg)
		syncLog = total
		cfg = rootCfg
		message = fmt.Sprintf("root sync applied to %d user workspaces", userCount)
		fmt.Printf("[Config] sync-main admin=%s users=%d %s\n", principal.Username, userCount, formatSyncLog(total))
	} else {
		if err := s.ensureUserBaseConfig(principal.Username); err != nil {
			s.configMu.Unlock()
			fmt.Printf("[Config] sync-main ensure base failed user=%s err=%v\n", principal.Username, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.migrateLegacyActiveTemplateToMain(principal.Username); err != nil {
			s.configMu.Unlock()
			fmt.Printf("[Config] sync-main migrate active failed user=%s err=%v\n", principal.Username, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cfg, activePath, _, err = s.loadUserActiveConfig(principal.Username)
		if err != nil {
			s.configMu.Unlock()
			fmt.Printf("[Config] sync-main load active failed user=%s err=%v\n", principal.Username, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		changed, appliedLog := syncMissingServicesFromMain(rootCfg, cfg)
		syncLog = appliedLog
		if changed {
			if err := config.SaveConfig(activePath, cfg); err != nil {
				s.configMu.Unlock()
				fmt.Printf("[Config] sync-main save active failed user=%s path=%s err=%v\n", principal.Username, activePath, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		fmt.Printf("[Config] sync-main apply user=%s changed=%t %s\n", principal.Username, changed, formatSyncLog(syncLog))
	}

	version, err := s.readRootConfigVersion()
	if err != nil {
		s.configMu.Unlock()
		fmt.Printf("[Config] sync-main version failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.cfg = cfg
	s.configMu.Unlock()

	fmt.Printf("[Config] sync-main user=%s activePath=%s %s\n", principal.Username, activePath, formatSyncLog(syncLog))
	json.NewEncoder(w).Encode(syncMainConfigResponse{
		Success: true,
		Message: message,
		Sync:    syncLog,
		Config:  cfg,
		Version: version,
	})
}

// handleSyncMainDelete applies only deletion-type sync from root config into current user active config.
// This endpoint is intentionally dangerous and should be protected by client-side confirmation.
func (s *Server) handleSyncMainDelete(w http.ResponseWriter, r *http.Request) {
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
	if principal.Role == roleAdmin {
		http.Error(w, "admin is not supported for delete-sync endpoint", http.StatusForbidden)
		return
	}

	s.configMu.Lock()
	defer s.configMu.Unlock()

	rootCfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		fmt.Printf("[Config] sync-main-delete load root failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userCfg, activePath, _, err := s.loadUserActiveConfig(principal.Username)
	if err != nil {
		fmt.Printf("[Config] sync-main-delete load active failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	changed, deleteLog := removeDeletedFromMain(rootCfg, userCfg)
	if changed {
		if err := config.SaveConfig(activePath, userCfg); err != nil {
			fmt.Printf("[Config] sync-main-delete save failed user=%s path=%s err=%v\n", principal.Username, activePath, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	version, err := s.readRootConfigVersion()
	if err != nil {
		fmt.Printf("[Config] sync-main-delete version failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.cfg = userCfg
	fmt.Printf("[Config] sync-main-delete user=%s path=%s changed=%t %s\n",
		principal.Username, activePath, changed, formatSyncLog(deleteLog))

	json.NewEncoder(w).Encode(syncMainConfigResponse{
		Success: true,
		Message: "main delete sync finished",
		Sync:    deleteLog,
		Config:  userCfg,
		Version: version,
	})
}

func (s *Server) buildTemplateEditorTree() ([]templateEditorTreeNode, error) {
	root := filepath.Clean(s.templatesDir)
	var buildNode func(fullPath string, relativePath string) (templateEditorTreeNode, error)
	buildNode = func(fullPath string, relativePath string) (templateEditorTreeNode, error) {
		info, err := os.Stat(fullPath)
		if err != nil {
			return templateEditorTreeNode{}, err
		}
		node := templateEditorTreeNode{
			Name: info.Name(),
			Path: filepath.ToSlash(relativePath),
		}
		if info.IsDir() {
			node.Type = "dir"
			entries, err := os.ReadDir(fullPath)
			if err != nil {
				return templateEditorTreeNode{}, err
			}
			children := make([]templateEditorTreeNode, 0, len(entries))
			for _, entry := range entries {
				childRel := filepath.ToSlash(filepath.Join(relativePath, entry.Name()))
				childFull := filepath.Join(fullPath, entry.Name())
				child, err := buildNode(childFull, childRel)
				if err != nil {
					continue
				}
				children = append(children, child)
			}
			sort.SliceStable(children, func(i, j int) bool {
				if children[i].Type == children[j].Type {
					return children[i].Name < children[j].Name
				}
				return children[i].Type == "dir"
			})
			node.Children = children
			return node, nil
		}
		node.Type = "file"
		return node, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []templateEditorTreeNode{}, nil
		}
		return nil, err
	}
	result := make([]templateEditorTreeNode, 0, len(entries))
	for _, entry := range entries {
		rel := filepath.ToSlash(entry.Name())
		full := filepath.Join(root, entry.Name())
		node, err := buildNode(full, rel)
		if err != nil {
			continue
		}
		result = append(result, node)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Type == result[j].Type {
			return result[i].Name < result[j].Name
		}
		return result[i].Type == "dir"
	})
	return result, nil
}

func (s *Server) normalizeTemplateEditorPath(input string) (string, string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(input)))
	clean = strings.TrimPrefix(clean, "./")
	if clean == "." || clean == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if strings.HasPrefix(clean, "/") || strings.Contains(clean, "..") {
		return "", "", fmt.Errorf("invalid template path")
	}
	full := filepath.Join(s.templatesDir, filepath.FromSlash(clean))
	templatesRoot := filepath.Clean(s.templatesDir)
	if !strings.HasPrefix(filepath.Clean(full), templatesRoot) {
		return "", "", fmt.Errorf("template path out of root")
	}
	return clean, full, nil
}

// handleTemplateEditorTree returns templates directory tree for online editor.
func (s *Server) handleTemplateEditorTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	nodes, err := s.buildTemplateEditorTree()
	if err != nil {
		fmt.Printf("[TemplateEditor] tree failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"root":     "templates",
		"readonly": principal.Role != roleAdmin,
		"nodes":    nodes,
	})
}

// handleTemplateEditorFile reads/saves one template file. Save is admin-only.
func (s *Server) handleTemplateEditorFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rawPath := strings.TrimSpace(r.URL.Query().Get("path"))
		normalizedPath, fullPath, err := s.normalizeTemplateEditorPath(rawPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		content, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "template file not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		info, _ := os.Stat(fullPath)
		updatedAt := ""
		if info != nil {
			updatedAt = info.ModTime().Format(time.RFC3339)
		}
		json.NewEncoder(w).Encode(templateEditorFileResponse{
			Path:      normalizedPath,
			Content:   string(content),
			Readonly:  principal.Role != roleAdmin,
			UpdatedAt: updatedAt,
		})
	case http.MethodPut:
		if principal.Role != roleAdmin {
			http.Error(w, "only admin can edit templates", http.StatusForbidden)
			return
		}
		var req templateEditorFileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		normalizedPath, fullPath, err := s.normalizeTemplateEditorPath(req.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !strings.HasSuffix(normalizedPath, ".tmpl") {
			http.Error(w, "only .tmpl file is editable", http.StatusBadRequest)
			return
		}
		if err := s.validateDraftTemplate(req.Content); err != nil {
			http.Error(w, fmt.Sprintf("%s template parse failed: %v", normalizedPath, err), http.StatusBadRequest)
			return
		}
		if err := validateDraftTemplateContract(normalizedPath, req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("[TemplateEditor] save user=%s path=%s bytes=%d\n", principal.Username, normalizedPath, len(req.Content))
		info, _ := os.Stat(fullPath)
		updatedAt := ""
		if info != nil {
			updatedAt = info.ModTime().Format(time.RFC3339)
		}
		json.NewEncoder(w).Encode(templateEditorFileResponse{
			Path:      normalizedPath,
			Content:   req.Content,
			Readonly:  false,
			UpdatedAt: updatedAt,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTemplateEditorItem creates or deletes template files/directories. Admin-only.
func (s *Server) handleTemplateEditorItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal, err := requestPrincipal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if principal.Role != roleAdmin {
		http.Error(w, "only admin can change template tree", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req templateEditorItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		normalizedPath, fullPath, err := s.normalizeTemplateEditorPath(req.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch req.Type {
		case "dir":
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "file":
			if !strings.HasSuffix(normalizedPath, ".tmpl") {
				http.Error(w, "template file must end with .tmpl", http.StatusBadRequest)
				return
			}
			if _, err := os.Stat(fullPath); err == nil {
				http.Error(w, "template file already exists", http.StatusConflict)
				return
			}
			if err := s.validateDraftTemplate(req.Content); err != nil {
				http.Error(w, fmt.Sprintf("%s template parse failed: %v", normalizedPath, err), http.StatusBadRequest)
				return
			}
			if err := validateDraftTemplateContract(normalizedPath, req.Content); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			http.Error(w, "type must be file or dir", http.StatusBadRequest)
			return
		}
		fmt.Printf("[TemplateEditor] create user=%s type=%s path=%s\n", principal.Username, req.Type, normalizedPath)
		json.NewEncoder(w).Encode(map[string]any{"success": true, "path": normalizedPath, "type": req.Type})
	case http.MethodDelete:
		rawPath := strings.TrimSpace(r.URL.Query().Get("path"))
		normalizedPath, fullPath, err := s.normalizeTemplateEditorPath(rawPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := os.Stat(fullPath); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "template item not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.RemoveAll(fullPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("[TemplateEditor] delete user=%s path=%s\n", principal.Username, normalizedPath)
		json.NewEncoder(w).Encode(map[string]any{"success": true, "path": normalizedPath})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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

	cfg, _, _, err := s.loadConfigForPrincipal(principal)
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

	cfg, _, _, err := s.loadConfigForPrincipal(principal)
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
