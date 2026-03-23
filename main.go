package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	IP       string `yaml:"ip"`
	Hostname string `yaml:"hostname"`
}

// Nodes 节点池（支持一个IP对应多个节点别名）
type Nodes map[string]Node

// ServiceTopo 服务拓扑配置
type ServiceTopo struct {
	Nodes        []string `yaml:"nodes"`          // 节点别名列表，支持 ["*"]
	IDAutoDerive bool     `yaml:"id_auto_derive"` // 是否自动推导ID
}

// ServiceTopos 服务拓扑列表
type ServiceTopos map[string]ServiceTopo

// ServiceConfig 服务配置（完全配置化）
type ServiceConfig struct {
	Type        string                 `yaml:"type"`        // 服务类型：global 或 空
	Description string                 `yaml:"description"` // 服务描述
	IDField     string                 `yaml:"id_field"`    // ID字段名
	IDFormat    string                 `yaml:"id_format"`   // ID格式化模板
	Vars        map[string]interface{} `yaml:"vars"`        // 服务变量
}

// ServiceConfigs 服务配置列表
type ServiceConfigs map[string]ServiceConfig

// NodeOverrides 节点特化配置
type NodeOverrides map[string]map[string]map[string]interface{}

// Config 完整配置
type Config struct {
	Global        Global         `yaml:"global"`
	Nodes         Nodes          `yaml:"nodes"`
	ServiceTop    ServiceTopos   `yaml:"serviceTop"`
	ServerConfig  ServiceConfigs `yaml:"serverConfig"`
	NodeOverrides NodeOverrides  `yaml:"nodeOverrides"`
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

			// 查找模板文件
			tmplFiles, err := filepath.Glob(filepath.Join(tmplDir, "*.tmpl"))
			if err != nil {
				return fmt.Errorf("查找模板失败: %w", err)
			}

			for _, tmplFile := range tmplFiles {
				// 渲染模板
				content, err := RenderTemplate(tmplFile, ctx, cfg)
				if err != nil {
					return fmt.Errorf("渲染模板 %s 失败: %w", tmplFile, err)
				}

				// 生成输出文件名（不再需要节点前缀，因为已经按节点分目录）
				baseName := filepath.Base(tmplFile)
				outputName := strings.TrimSuffix(baseName, ".tmpl")

				// 写入文件
				outputPath := filepath.Join(serviceDir, outputName)
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
// 主函数
// ============================================================

func main() {
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
