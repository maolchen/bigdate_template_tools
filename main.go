package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// ============================================================
// 配置结构定义（完全通用、零硬编码）
// ============================================================

// Global 全局配置
type Global map[string]interface{}

// Node 节点信息
type Node struct {
	IP       string `yaml:"ip" json:"ip"`
	Hostname string `yaml:"hostname" json:"hostname"`
}

// Nodes 节点池（支持一个IP对应多个节点别名）
type Nodes map[string]Node

// ServiceTopo 服务拓扑配置
type ServiceTopo struct {
	Nodes        []string `yaml:"nodes" json:"nodes"`                 // 节点别名列表，支持 ["*"]
	IDAutoDerive bool     `yaml:"id_auto_derive" json:"id_auto_derive"` // 是否自动推导ID
}

// ServiceTopos 服务拓扑列表
type ServiceTopos map[string]ServiceTopo

// ServiceConfig 服务配置（完全配置化）
type ServiceConfig struct {
	Type        string                 `yaml:"type" json:"type"`               // 服务类型：global 或 空
	Description string                 `yaml:"description" json:"description"` // 服务描述
	IDField     string                 `yaml:"id_field" json:"id_field"`       // ID字段名
	IDFormat    string                 `yaml:"id_format" json:"id_format"`     // ID格式化模板
	Vars        map[string]interface{} `yaml:"vars" json:"vars"`               // 服务变量
}

// ServiceConfigs 服务配置列表
type ServiceConfigs map[string]ServiceConfig

// NodeOverrides 节点特化配置
type NodeOverrides map[string]map[string]map[string]interface{}

// Config 完整配置
type Config struct {
	Global        Global         `yaml:"global" json:"global"`
	Nodes         Nodes          `yaml:"nodes" json:"nodes"`
	ServiceTop    ServiceTopos   `yaml:"serviceTop" json:"serviceTop"`
	ServerConfig  ServiceConfigs `yaml:"serverConfig" json:"serverConfig"`
	NodeOverrides NodeOverrides  `yaml:"nodeOverrides" json:"nodeOverrides"`
}

// ============================================================
// 运行时数据结构
// ============================================================

// NodeInstance 节点实例（运行时）
type NodeInstance struct {
	IP       string
	Hostname string
	NodeName string // 节点别名
}

// ServiceInstance 服务实例（运行时）
type ServiceInstance struct {
	ServiceName string
	NodeName    string
	Node        NodeInstance
	Vars        map[string]interface{} // 合并后的变量
	AutoID      interface{}            // 自动推导的ID
}

// Context 模板上下文（零硬编码，不包含任何特定服务字段）
type Context struct {
	Global       Global
	Nodes        Nodes
	Instance     ServiceInstance
	AllInstances []ServiceInstance // 所有服务实例
}

// ============================================================
// 全局变量（Web 模式使用）
// ============================================================

var (
	webConfigPath   string
	webConfig       Config
	webOutputDir    string
	webTemplatesDir string
	webWebDir       string
)

// ============================================================
// 辅助函数
// ============================================================

// getNestedPort 从嵌套的 map 中获取端口值
// 支持格式: "zookeeper.client_port" 或 "server.port"
func getNestedPort(vars map[string]interface{}, fieldPath string) int {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 1 {
		// 直接字段
		if p, ok := vars[parts[0]].(int); ok {
			return p
		}
		return 0
	}

	// 嵌套字段
	current := vars
	for i, part := range parts {
		if i == len(parts)-1 {
			// 最后一层，获取端口值
			if p, ok := current[part].(int); ok {
				return p
			}
			return 0
		}
		// 中间层，继续深入
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return 0
		}
	}
	return 0
}

// ============================================================
// ID格式化器（通用）
// ============================================================

// FormatID 根据格式模板生成ID
// 支持格式：
//   - "index+1": 索引+1（如 1, 2, 3）
//   - "index": 原始索引（如 0, 1, 2）
//   - "nn{index+1}": 格式化模板（如 nn1, nn2）
//   - "broker-{index+1}": 带前缀（如 broker-1, broker-2）
func FormatID(format string, index int) interface{} {
	// 计算索引偏移后的值
	indexPlusOne := index + 1

	// 替换占位符
	result := strings.ReplaceAll(format, "{index}", fmt.Sprintf("%d", index))
	result = strings.ReplaceAll(result, "{index+1}", fmt.Sprintf("%d", indexPlusOne))

	// 特殊格式：纯数值
	if result == "index" {
		return index
	}
	if result == "index+1" {
		return indexPlusOne
	}

	// 格式化模板
	return result
}

// ============================================================
// 配置加载
// ============================================================

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}

// ============================================================
// 服务实例构建
// ============================================================

// BuildServiceInstances 构建服务实例列表
func BuildServiceInstances(cfg *Config) []ServiceInstance {
	var instances []ServiceInstance

	for serviceName, topo := range cfg.ServiceTop {
		// 获取服务配置
		serviceConfig, exists := cfg.ServerConfig[serviceName]
		if !exists {
			// 如果没有配置，使用空配置
			serviceConfig = ServiceConfig{
				Vars: make(map[string]interface{}),
			}
		}

		// 处理全局服务（nodes: ["*"]）
		targetNodes := topo.Nodes
		if len(topo.Nodes) == 1 && (topo.Nodes[0] == "*" || topo.Nodes[0] == "all") {
			// 展开为所有节点
			targetNodes = make([]string, 0, len(cfg.Nodes))
			for nodeName := range cfg.Nodes {
				targetNodes = append(targetNodes, nodeName)
			}
		}

		// 构建每个节点上的服务实例
		for idx, nodeName := range targetNodes {
			node, exists := cfg.Nodes[nodeName]
			if !exists {
				fmt.Printf("警告: 节点 %s 未在nodes中定义\n", nodeName)
				continue
			}

			// 合并变量
			vars := make(map[string]interface{})
			for k, v := range serviceConfig.Vars {
				vars[k] = v
			}

			// 应用节点特化配置
			if nodeOverrides, ok := cfg.NodeOverrides[nodeName]; ok {
				if serviceOverrides, ok := nodeOverrides[serviceName]; ok {
					for k, v := range serviceOverrides {
						vars[k] = v
					}
				}
			}

			// 自动推导ID
			var autoID interface{}
			if topo.IDAutoDerive && serviceConfig.IDField != "" {
				autoID = FormatID(serviceConfig.IDFormat, idx)
			}

			instances = append(instances, ServiceInstance{
				ServiceName: serviceName,
				NodeName:    nodeName,
				Node: NodeInstance{
					IP:       node.IP,
					Hostname: node.Hostname,
					NodeName: nodeName,
				},
				Vars:   vars,
				AutoID: autoID,
			})
		}
	}

	return instances
}

// ============================================================
// 模板渲染（零硬编码，通过模板函数实现通用性）
// ============================================================

// RenderTemplate 渲染模板
func RenderTemplate(tmplPath string, ctx Context, cfg *Config) (string, error) {
	// 读取模板文件
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("读取模板失败: %w", err)
	}

	// 创建模板，添加通用模板函数（不包含任何特定服务逻辑）
	tmpl, err := template.New("config").Funcs(template.FuncMap{
		// =============== 基础函数 ===============
		"toUpper": strings.ToUpper,
		"toLower": strings.ToLower,
		"trim":    strings.TrimSpace,
		"replace": strings.ReplaceAll,
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},

		// =============== 数学函数 ===============
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			return a / b
		},

		// =============== 服务节点相关函数 ===============
		// 获取同一服务的所有节点实例
		"serviceNodes": func(serviceName string) []ServiceInstance {
			var result []ServiceInstance
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					result = append(result, inst)
				}
			}
			return result
		},

		// 获取服务的端点列表（IP:Port格式）
		// serviceName: 服务名
		// portField: 端口字段名（支持嵌套，如 "zookeeper.client_port"）
		"serviceEndpoints": func(serviceName, portField string) []string {
			var endpoints []string
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					// 获取端口值（支持嵌套字段）
					port := getNestedPort(inst.Vars, portField)
					if port > 0 {
						endpoints = append(endpoints, fmt.Sprintf("%s:%d", inst.Node.IP, port))
					}
				}
			}
			return endpoints
		},

		// 获取服务的端点连接串（用逗号连接）
		"serviceEndpointsJoin": func(serviceName, portField string) string {
			endpoints := make([]string, 0)
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					// 获取端口值（支持嵌套字段）
					port := getNestedPort(inst.Vars, portField)
					if port > 0 {
						endpoints = append(endpoints, fmt.Sprintf("%s:%d", inst.Node.IP, port))
					}
				}
			}
			return strings.Join(endpoints, ",")
		},

		// 获取服务的所有IP列表
		"serviceIPs": func(serviceName string) []string {
			var ips []string
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					ips = append(ips, inst.Node.IP)
				}
			}
			return ips
		},

		// 获取服务的所有主机名列表
		"serviceHostnames": func(serviceName string) []string {
			var hostnames []string
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					hostnames = append(hostnames, inst.Node.Hostname)
				}
			}
			return hostnames
		},

		// 获取服务的所有节点别名列表
		"getServiceNodes": func(serviceName string) []string {
			var nodeNames []string
			seen := make(map[string]bool)
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName && !seen[inst.NodeName] {
					nodeNames = append(nodeNames, inst.NodeName)
					seen[inst.NodeName] = true
				}
			}
			return nodeNames
		},

		// 获取服务配置变量
		"serviceVars": func(serviceName string) map[string]interface{} {
			if svc, ok := cfg.ServerConfig[serviceName]; ok {
				return svc.Vars
			}
			return make(map[string]interface{})
		},

		// 获取服务的特定变量值
		"serviceVar": func(serviceName, varName string) interface{} {
			if svc, ok := cfg.ServerConfig[serviceName]; ok {
				if val, exists := svc.Vars[varName]; exists {
					return val
				}
			}
			return nil
		},

		// =============== 配置访问函数 ===============
		// 获取服务配置
		"serviceConfig": func(serviceName string) ServiceConfig {
			return cfg.ServerConfig[serviceName]
		},

		// 获取节点信息
		"nodeInfo": func(nodeName string) *Node {
			if node, ok := cfg.Nodes[nodeName]; ok {
				return &node
			}
			return nil
		},
	}).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	// 渲染
	var result strings.Builder
	if err := tmpl.Execute(&result, ctx); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return result.String(), nil
}

// ============================================================
// 输出生成（按节点组织目录）
// ============================================================

// GenerateOutputs 生成所有配置文件和脚本
// 输出结构：output/<IP>/<服务名>/<配置文件>
func GenerateOutputs(cfg *Config, instances []ServiceInstance, outputDir string) error {
	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 按节点IP分组
	nodeInstances := make(map[string][]ServiceInstance)
	for _, inst := range instances {
		nodeInstances[inst.Node.IP] = append(nodeInstances[inst.Node.IP], inst)
	}

	// 为每个节点生成配置
	for nodeIP, svcs := range nodeInstances {
		// 创建节点目录（以IP命名）
		nodeDir := filepath.Join(outputDir, nodeIP)
		if err := os.MkdirAll(nodeDir, 0755); err != nil {
			return fmt.Errorf("创建节点目录失败: %w", err)
		}

		// 为该节点上的每个服务生成配置
		for _, inst := range svcs {
			serviceName := inst.ServiceName

			// 检查模板目录是否存在
			tmplDir := filepath.Join("templates", serviceName)
			if _, err := os.Stat(tmplDir); os.IsNotExist(err) {
				fmt.Printf("提示: 节点 %s 的服务 %s 没有模板目录，跳过\n", nodeIP, serviceName)
				continue
			}

			// 创建服务目录
			serviceDir := filepath.Join(nodeDir, serviceName)
			if err := os.MkdirAll(serviceDir, 0755); err != nil {
				return fmt.Errorf("创建服务目录失败: %w", err)
			}

			// 构建上下文
			ctx := Context{
				Global:       cfg.Global,
				Nodes:        cfg.Nodes,
				Instance:     inst,
				AllInstances: instances,
			}

			// 递归查找所有模板文件
			var tmplFiles []string
			err := filepath.Walk(tmplDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".tmpl") {
					tmplFiles = append(tmplFiles, path)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("查找模板失败: %w", err)
			}

			for _, tmplFile := range tmplFiles {
				// 渲染模板
				content, err := RenderTemplate(tmplFile, ctx, cfg)
				if err != nil {
					return fmt.Errorf("渲染模板 %s 失败: %w", tmplFile, err)
				}

				// 计算相对路径，保持子目录结构
				relPath, err := filepath.Rel(tmplDir, tmplFile)
				if err != nil {
					return fmt.Errorf("计算相对路径失败: %w", err)
				}

				// 生成输出文件名
				outputName := strings.TrimSuffix(relPath, ".tmpl")
				outputPath := filepath.Join(serviceDir, outputName)

				// 创建子目录（如果需要）
				outputSubDir := filepath.Dir(outputPath)
				if outputSubDir != "." && outputSubDir != serviceDir {
					if err := os.MkdirAll(outputSubDir, 0755); err != nil {
						return fmt.Errorf("创建输出子目录失败: %w", err)
					}
				}

				// 写入文件
				if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
					return fmt.Errorf("写入文件失败: %w", err)
				}

				fmt.Printf("生成: %s\n", outputPath)
			}
		}
	}

	return nil
}

// ============================================================
// Web 模式 API Handlers
// ============================================================

// GET /api/config - 获取当前配置
func webGetConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(webConfig)
}

// PUT /api/config - 保存配置
func webSaveConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var newConfig Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	webConfig = newConfig

	// 保存到文件
	data, err := yaml.Marshal(&webConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(webConfigPath, data, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已保存",
	})
}

// POST /api/generate - 生成配置
func webGenerateConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 清理输出目录
	os.RemoveAll(webOutputDir)
	os.MkdirAll(webOutputDir, 0755)

	results := make(map[string][]string)
	errors := []string{}
	warnings := []string{}

	// 构建服务实例
	instances := BuildServiceInstances(&webConfig)

	// 按节点IP分组
	nodeInstances := make(map[string][]ServiceInstance)
	for _, inst := range instances {
		nodeInstances[inst.Node.IP] = append(nodeInstances[inst.Node.IP], inst)
	}

	// 为每个节点生成配置
	for _, svcs := range nodeInstances {
		for _, inst := range svcs {
			serviceName := inst.ServiceName

			// 确定模板路径
			templatePath := filepath.Join(webTemplatesDir, serviceName)

			// 检查模板目录是否存在（改为警告而非错误）
			if _, err := os.Stat(templatePath); os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("服务 %s 没有对应的模板目录，已跳过生成", serviceName))
				continue
			}

			// 创建输出目录
			nodeOutputDir := filepath.Join(webOutputDir, inst.Node.IP, serviceName)
			os.MkdirAll(nodeOutputDir, 0755)

			// 构建上下文
			ctx := Context{
				Global:       webConfig.Global,
				Nodes:        webConfig.Nodes,
				Instance:     inst,
				AllInstances: instances,
			}

			// 递归查找所有模板文件
			var tmplFiles []string
			err := filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".tmpl") {
					tmplFiles = append(tmplFiles, path)
				}
				return nil
			})
			if err != nil {
				errors = append(errors, fmt.Sprintf("查找模板失败: %v", err))
				continue
			}

			generatedFiles := []string{}
			for _, tmplFile := range tmplFiles {
				// 渲染模板
				content, err := RenderTemplate(tmplFile, ctx, &webConfig)
				if err != nil {
					errors = append(errors, fmt.Sprintf("渲染 %s 失败: %v", tmplFile, err))
					continue
				}

				// 计算相对路径，保持子目录结构
				relPath, err := filepath.Rel(templatePath, tmplFile)
				if err != nil {
					errors = append(errors, fmt.Sprintf("计算相对路径失败: %v", err))
					continue
				}

				// 生成输出文件名
				outputName := strings.TrimSuffix(relPath, ".tmpl")
				outputPath := filepath.Join(nodeOutputDir, outputName)

				// 创建子目录（如果需要）
				outputSubDir := filepath.Dir(outputPath)
				if outputSubDir != "." && outputSubDir != nodeOutputDir {
					if err := os.MkdirAll(outputSubDir, 0755); err != nil {
						errors = append(errors, fmt.Sprintf("创建输出子目录失败: %v", err))
						continue
					}
				}

				// 写入文件
				if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
					errors = append(errors, fmt.Sprintf("写入文件失败: %v", err))
					continue
				}

				relOutputPath, _ := filepath.Rel(webOutputDir, outputPath)
				generatedFiles = append(generatedFiles, strings.ReplaceAll(relOutputPath, "\\", "/"))
			}

			results[inst.Node.IP] = append(results[inst.Node.IP], generatedFiles...)
		}
	}

	response := map[string]interface{}{
		"success":  len(errors) == 0,
		"message":  "配置生成完成",
		"results":  results,
		"errors":   errors,
		"warnings": warnings,
		"stats": map[string]int{
			"nodes":     len(webConfig.Nodes),
			"services":  len(webConfig.ServiceTop),
			"generated": webCountGeneratedFiles(),
			"skipped":   len(warnings),
		},
	}

	json.NewEncoder(w).Encode(response)
}

// GET /api/output - 获取生成的文件列表
func webGetOutputHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	files := make(map[string][]string)

	filepath.Walk(webOutputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(webOutputDir, path)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) >= 2 {
			ip := parts[0]
			files[ip] = append(files[ip], strings.Join(parts[1:], "/"))
		}
		return nil
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": files,
		"total": webCountGeneratedFiles(),
	})
}

// GET /api/output/download - 下载生成的配置包
func webDownloadOutputHandler(w http.ResponseWriter, r *http.Request) {
	// 创建 zip 文件
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	filepath.Walk(webOutputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(webOutputDir, path)
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
func webGetOutputFileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "缺少 path 参数", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(webOutputDir, filePath)
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
func webGetTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	templates := []map[string]interface{}{}

	filepath.Walk(webTemplatesDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if strings.HasSuffix(path, ".tmpl") {
			relPath, _ := filepath.Rel(webTemplatesDir, path)
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
func webReloadConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := webLoadConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已重新加载",
		"config":  webConfig,
	})
}

// POST /api/descriptions/{serviceName} - 保存服务配置说明
func webSaveDescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 从URL路径中提取服务名称
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "无效的路径", http.StatusBadRequest)
		return
	}
	serviceName := pathParts[3]

	// 解析说明数据
	var descriptions map[string]string
	if err := json.NewDecoder(r.Body).Decode(&descriptions); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 确定数据目录
	dataDir := filepath.Join(getWorkDir(), "data", "descriptions")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 保存到文件
	descPath := filepath.Join(dataDir, serviceName+".json")
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
		"message": "说明已保存",
	})
}

// GET /api/descriptions/{serviceName} - 获取服务配置说明
func webGetDescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 从URL路径中提取服务名称
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "无效的路径", http.StatusBadRequest)
		return
	}
	serviceName := pathParts[3]

	// 读取说明文件
	dataDir := filepath.Join(getWorkDir(), "data", "descriptions")
	descPath := filepath.Join(dataDir, serviceName+".json")

	descriptions := make(map[string]string)
	if data, err := os.ReadFile(descPath); err == nil {
		json.Unmarshal(data, &descriptions)
	}

	json.NewEncoder(w).Encode(descriptions)
}

// POST /api/descriptions/global - 保存全局配置说明
func webSaveGlobalDescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 解析说明数据
	var descriptions map[string]string
	if err := json.NewDecoder(r.Body).Decode(&descriptions); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 确定数据目录
	dataDir := filepath.Join(getWorkDir(), "data", "descriptions")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 保存到文件
	descPath := filepath.Join(dataDir, "global.json")
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
		"message": "说明已保存",
	})
}

// GET /api/descriptions/global - 获取全局配置说明
func webGetGlobalDescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 读取说明文件
	dataDir := filepath.Join(getWorkDir(), "data", "descriptions")
	descPath := filepath.Join(dataDir, "global.json")

	descriptions := make(map[string]string)
	if data, err := os.ReadFile(descPath); err == nil {
		json.Unmarshal(data, &descriptions)
	}

	json.NewEncoder(w).Encode(descriptions)
}

// POST /api/check-references - 检查删除对象时是否被template引用
func webCheckReferencesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req struct {
		Type     string `json:"type"`     // "global", "node", "service", "vars"
		Key      string `json:"key"`      // 对于node和service，是名称；对于global和vars，是字段名
		Service  string `json:"service"`  // 对于service和vars类型，指定服务名
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("[CheckReferences] 请求参数: type=%s, key=%s, service=%s\n", req.Type, req.Key, req.Service)

	references := []map[string]interface{}{}

	// 遍历所有template文件
	filepath.Walk(webTemplatesDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(webTemplatesDir, path)
		serviceName := filepath.Dir(relPath)
		if serviceName == "." {
			serviceName = strings.TrimSuffix(filepath.Base(path), ".tmpl")
		}

		contentStr := string(content)
		var isReferenced bool
		var details []string

		switch req.Type {
		case "global":
			// 检查全局配置引用: .Global.xxx
			pattern := fmt.Sprintf(`.Global\.%s`, req.Key)
			isReferenced = regexp.MustCompile(pattern).MatchString(contentStr)
			if isReferenced {
				details = append(details, fmt.Sprintf("引用全局配置: .Global.%s", req.Key))
			}

		case "node":
			// 检查节点引用: .Instance.Node.NodeName 或 serviceNodes "nodeName"
			pattern1 := fmt.Sprintf(`\.Instance\.Node\.NodeName\s*===\s*"%s"`, req.Key)
			pattern2 := fmt.Sprintf(`serviceNodes\s+"%s"`, req.Key)
			isReferenced = regexp.MustCompile(pattern1).MatchString(contentStr) ||
				regexp.MustCompile(pattern2).MatchString(contentStr)
			if isReferenced {
				details = append(details, fmt.Sprintf("引用节点: %s", req.Key))
			}

		case "service":
			// 检查服务引用: serviceNodes "serviceName", serviceEndpointsJoin "serviceName", 或者该服务有template文件夹
			pattern1 := fmt.Sprintf(`serviceNodes\s+"%s"`, req.Service)
			pattern2 := fmt.Sprintf(`serviceEndpointsJoin\s+"%s"`, req.Service)
			// 检查是否有其他template引用了该服务（通过template路径）
			isReferenced = regexp.MustCompile(pattern1).MatchString(contentStr) ||
				regexp.MustCompile(pattern2).MatchString(contentStr) ||
				(filepath.Dir(relPath) == req.Service)

			if isReferenced {
				if filepath.Dir(relPath) == req.Service {
					details = append(details, fmt.Sprintf("该服务的template目录: %s", relPath))
				} else {
					details = append(details, fmt.Sprintf("引用服务: %s", req.Service))
				}
			}
			case "vars":
				// 检查服务变量引用: .Instance.Vars.xxx
				// 如果指定了service，只检查该服务的template文件；否则检查所有template文件
				if req.Service == "" || filepath.Dir(relPath) == req.Service || filepath.Dir(relPath) == "." {
					pattern := fmt.Sprintf(`\.Instance\.Vars\.%s`, req.Key)
					isReferenced = regexp.MustCompile(pattern).MatchString(contentStr)
					if isReferenced {
						if req.Service != "" {
							details = append(details, fmt.Sprintf("引用服务变量: .Instance.Vars.%s (在服务 %s 中)", req.Key, req.Service))
						} else {
							details = append(details, fmt.Sprintf("引用服务变量: .Instance.Vars.%s (在 %s 中)", req.Key, relPath))
						}
					}
				} else if req.Service != "" {
					// 检查其他服务是否通过serviceNodes或serviceEndpointsJoin引用了该服务
					// 如果其他服务引用了该服务，而该服务的template中使用了这个变量，那么这个变量被间接引用
					pattern1 := fmt.Sprintf(`serviceNodes\s+"%s"`, req.Service)
					pattern2 := fmt.Sprintf(`serviceEndpointsJoin\s+"%s"`, req.Service)
					if regexp.MustCompile(pattern1).MatchString(contentStr) ||
						regexp.MustCompile(pattern2).MatchString(contentStr) {
						// 需要检查该服务的template文件是否使用这个变量
						serviceTemplateDir := filepath.Join(webTemplatesDir, req.Service)
						if _, err := os.Stat(serviceTemplateDir); err == nil {
							// 该服务有template目录，标记为引用
							isReferenced = true
							details = append(details, fmt.Sprintf("通过服务引用间接使用: .Instance.Vars.%s (在%s服务的template中)", req.Key, req.Service))
						}
					}
				}

		if isReferenced {
			references = append(references, map[string]interface{}{
				"path":    relPath,
				"service": serviceName,
				"details": details,
			})
		}

		return nil
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"hasReferences": len(references) > 0,
		"references":    references,
	})
}

func webLoadConfig() error {
	data, err := os.ReadFile(webConfigPath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &webConfig)
}

func webCountGeneratedFiles() int {
	count := 0
	filepath.Walk(webOutputDir, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
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

// ============================================================
// 获取工作目录
// ============================================================

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

// ============================================================
// 命令行模式
// ============================================================

func runCLIMode() {
	// 加载配置
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("============================================")
	fmt.Println("配置加载成功")
	fmt.Println("============================================")
	fmt.Printf("节点数: %d\n", len(cfg.Nodes))
	fmt.Printf("服务数: %d\n", len(cfg.ServiceTop))
	fmt.Println()

	// 构建服务实例
	instances := BuildServiceInstances(cfg)

	// 打印服务实例摘要
	fmt.Println("============================================")
	fmt.Println("服务实例列表")
	fmt.Println("============================================")
	for serviceName, topo := range cfg.ServiceTop {
		serviceConfig := cfg.ServerConfig[serviceName]
		fmt.Printf("\n[%s]", serviceName)
		if serviceConfig.Description != "" {
			fmt.Printf(" - %s", serviceConfig.Description)
		}
		fmt.Println()

		// 显示节点类型
		if len(topo.Nodes) == 1 && (topo.Nodes[0] == "*" || topo.Nodes[0] == "all") {
			fmt.Println("  类型: 全局服务（所有节点）")
		}

		// 显示ID配置
		if topo.IDAutoDerive && serviceConfig.IDField != "" {
			fmt.Printf("  ID字段: %s (格式: %s)\n", serviceConfig.IDField, serviceConfig.IDFormat)
		}

		// 显示节点列表
		fmt.Println("  节点:")
		for _, inst := range instances {
			if inst.ServiceName == serviceName {
				idStr := ""
				if inst.AutoID != nil {
					idStr = fmt.Sprintf(" [%s=%v]", serviceConfig.IDField, inst.AutoID)
				}
				fmt.Printf("    - %s (%s)%s\n", inst.NodeName, inst.Node.IP, idStr)
			}
		}
	}

	// 节点复用统计
	fmt.Println("\n============================================")
	fmt.Println("节点复用统计")
	fmt.Println("============================================")
	ipUsage := make(map[string][]string)
	for nodeName, node := range cfg.Nodes {
		ipUsage[node.IP] = append(ipUsage[node.IP], nodeName)
	}
	for ip, names := range ipUsage {
		if len(names) > 1 {
			fmt.Printf("%s: %v (复用)\n", ip, names)
		}
	}

	// 生成输出
	fmt.Println("\n============================================")
	fmt.Println("开始生成配置文件")
	fmt.Println("============================================")
	outputDir := "output"
	if err := GenerateOutputs(cfg, instances, outputDir); err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n============================================")
	fmt.Println("生成完成！")
	fmt.Println("============================================")
	fmt.Printf("输出目录: %s\n", outputDir)
}

// ============================================================
// Web 模式
// ============================================================

func runWebMode(port string) {
	// 确定工作目录
	workDir := getWorkDir()
	fmt.Printf("工作目录: %s\n", workDir)

	// 设置路径
	webConfigPath = filepath.Join(workDir, "config.yaml")
	webOutputDir = filepath.Join(workDir, "output")
	webTemplatesDir = filepath.Join(workDir, "templates")
	webWebDir = filepath.Join(workDir, "web", "dist")

	fmt.Printf("尝试前端目录: %s\n", webWebDir)
	// 如果 web/dist 不存在，尝试 server/web/dist
	if _, err := os.Stat(webWebDir); os.IsNotExist(err) {
		webWebDir = filepath.Join(workDir, "server", "web", "dist")
		fmt.Printf("前端目录不存在，尝试: %s\n", webWebDir)
	}

	// 加载配置
	if err := webLoadConfig(); err != nil {
		fmt.Printf("警告: 无法加载配置文件 %s: %v\n", webConfigPath, err)
		// 使用默认配置
		webConfig = Config{
			Global: map[string]interface{}{
				"user":           "bigdata",
				"group":          "bigdata",
				"install_base_dir": "/data/localization",
				"data_base_dir":    "/data",
				"java_home":       "/data/jdk",
			},
			Nodes:        make(Nodes),
			ServiceTop:   make(ServiceTopos),
			ServerConfig: make(ServiceConfigs),
		}
	}

	// 创建路由
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			webGetConfigHandler(w, r)
		case "PUT":
			webSaveConfigHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		webGenerateConfigHandler(w, r)
	})

	mux.HandleFunc("/api/output", webGetOutputHandler)
	mux.HandleFunc("/api/output/download", webDownloadOutputHandler)
	mux.HandleFunc("/api/output/file", webGetOutputFileHandler)
	mux.HandleFunc("/api/templates", webGetTemplatesHandler)
	mux.HandleFunc("/api/config/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		webReloadConfigHandler(w, r)
	})

	// 说明相关API
	mux.HandleFunc("/api/descriptions/global", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			webGetGlobalDescriptionsHandler(w, r)
		case "POST":
			webSaveGlobalDescriptionsHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/descriptions/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			webGetDescriptionsHandler(w, r)
		case "POST":
			webSaveDescriptionsHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/check-references", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		webCheckReferencesHandler(w, r)
	})

	// 静态文件服务（前端）
	fmt.Printf("检查前端目录: %s\n", webWebDir)
	if info, err := os.Stat(webWebDir); err == nil {
		if info.IsDir() {
			fs := http.FileServer(http.Dir(webWebDir))
			mux.Handle("/", fs)
			fmt.Printf("✓ 前端静态文件目录: %s\n", webWebDir)
			// 列出文件数量
			files, _ := os.ReadDir(webWebDir)
			fmt.Printf("  文件数: %d\n", len(files))
		} else {
			fmt.Printf("✗ 前端路径不是目录: %s\n", webWebDir)
		}
	} else {
		fmt.Printf("✗ 前端静态文件目录不存在: %s, 错误: %v\n", webWebDir, err)
	}

	// 启动服务器
	if port == "" {
		port = "5000"
	}

	fmt.Printf("============================================\n")
	fmt.Printf("大数据平台配置生成器 Web 服务\n")
	fmt.Printf("============================================\n")
	fmt.Printf("工作目录: %s\n", workDir)
	fmt.Printf("配置文件: %s\n", webConfigPath)
	fmt.Printf("模板目录: %s\n", webTemplatesDir)
	fmt.Printf("输出目录: %s\n", webOutputDir)
	fmt.Printf("端口: %s\n", port)
	fmt.Printf("API: http://localhost:%s/api/config\n", port)
	fmt.Printf("Web: http://localhost:%s/\n", port)
	fmt.Printf("============================================\n")

	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		fmt.Printf("启动服务失败: %v\n", err)
		os.Exit(1)
	}
}

// ============================================================
// 主函数
// ============================================================

func main() {
	// 解析命令行参数
	webMode := flag.Bool("web", false, "启动 Web 服务模式")
	port := flag.String("port", "", "Web 服务端口（默认 5000）")
	flag.Parse()

	if *webMode {
		// Web 模式
		runWebMode(*port)
	} else {
		// 命令行模式（默认）
		runCLIMode()
	}
}
