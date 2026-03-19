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
// 配置结构定义（通用、无硬编码）
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
	Nodes         []string `yaml:"nodes"`          // 节点别名列表，支持 ["*"]
	IDAutoDerive  bool     `yaml:"id_auto_derive"` // 是否自动推导ID
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
	ServerConfig   ServiceConfigs `yaml:"serverConfig"`
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

// Context 模板上下文
type Context struct {
	Global        Global
	Nodes         Nodes
	Instance      ServiceInstance
	AllInstances  []ServiceInstance // 所有服务实例，用于在模板中访问其他节点
	// 衍生变量（在运行时计算）
	ZKConnect     string // ZooKeeper连接串
	KafkaBrokers  string // Kafka Brokers列表
}

// ============================================================
// ID格式化器
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
// 衍生变量计算器
// ============================================================

// DeriveVars 计算衍生变量
func DeriveVars(cfg *Config) map[string]interface{} {
	derived := make(map[string]interface{})

	// 遍历所有服务配置
	for serviceName, serviceConfig := range cfg.ServerConfig {
		switch serviceName {
		case "zookeeper":
			// 计算ZK连接串
			if topo, exists := cfg.ServiceTop[serviceName]; exists {
				var servers []string
				for _, nodeName := range topo.Nodes {
					if node, ok := cfg.Nodes[nodeName]; ok {
						// 获取client_port
						clientPort := 2181
						if port, ok := serviceConfig.Vars["client_port"].(int); ok {
							clientPort = port
						}
						servers = append(servers, fmt.Sprintf("%s:%d", node.IP, clientPort))
					}
				}
				if len(servers) > 0 {
					derived["zk_connect"] = strings.Join(servers, ",")
				}
			}

		case "kafka":
			// 计算Kafka Brokers
			if topo, exists := cfg.ServiceTop[serviceName]; exists {
				var brokers []string
				for _, nodeName := range topo.Nodes {
					if node, ok := cfg.Nodes[nodeName]; ok {
						// 获取broker_port
						brokerPort := 9092
						if port, ok := serviceConfig.Vars["broker_port"].(int); ok {
							brokerPort = port
						}
						brokers = append(brokers, fmt.Sprintf("%s:%d", node.IP, brokerPort))
					}
				}
				if len(brokers) > 0 {
					derived["kafka_brokers"] = strings.Join(brokers, ",")
				}
			}

		// 可以继续添加其他服务的衍生变量计算逻辑
		// 或者完全通过模板函数在模板中计算
		}
	}

	return derived
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
// 模板渲染
// ============================================================

// RenderTemplate 渲染模板
func RenderTemplate(tmplPath string, ctx Context) (string, error) {
	// 读取模板文件
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("读取模板失败: %w", err)
	}

	// 创建模板，添加自定义函数
	tmpl, err := template.New("config").Funcs(template.FuncMap{
		// 可以添加自定义模板函数
		"toUpper": strings.ToUpper,
		"toLower": strings.ToLower,
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
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
// 输出生成
// ============================================================

// GenerateOutputs 生成所有配置文件和脚本
func GenerateOutputs(cfg *Config, instances []ServiceInstance, outputDir string) error {
	// 计算衍生变量
	derived := DeriveVars(cfg)

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 按服务分组
	serviceInstances := make(map[string][]ServiceInstance)
	for _, inst := range instances {
		serviceInstances[inst.ServiceName] = append(serviceInstances[inst.ServiceName], inst)
	}

	// 为每个服务生成配置
	for serviceName, svcs := range serviceInstances {
		// 创建服务目录
		serviceDir := filepath.Join(outputDir, serviceName)
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return fmt.Errorf("创建服务目录失败: %w", err)
		}

		// 检查模板目录是否存在
		tmplDir := filepath.Join("templates", serviceName)
		if _, err := os.Stat(tmplDir); os.IsNotExist(err) {
			fmt.Printf("提示: 服务 %s 没有模板目录，跳过模板渲染\n", serviceName)
			continue
		}

		// 为每个实例生成配置
		for _, inst := range svcs {
			// 构建上下文
			ctx := Context{
				Global:       cfg.Global,
				Nodes:        cfg.Nodes,
				Instance:     inst,
				AllInstances: instances, // 传入所有实例
				ZKConnect:    derived["zk_connect"].(string),
				KafkaBrokers: derived["kafka_brokers"].(string),
			}

			// 查找模板文件
			tmplFiles, err := filepath.Glob(filepath.Join(tmplDir, "*.tmpl"))
			if err != nil {
				return fmt.Errorf("查找模板失败: %w", err)
			}

			for _, tmplFile := range tmplFiles {
				// 渲染模板
				content, err := RenderTemplate(tmplFile, ctx)
				if err != nil {
					return fmt.Errorf("渲染模板 %s 失败: %w", tmplFile, err)
				}

				// 生成输出文件名
				baseName := filepath.Base(tmplFile)
				outputName := strings.TrimSuffix(baseName, ".tmpl")

				// 为节点特定配置添加节点名前缀
				if !strings.Contains(outputName, "install") {
					outputName = fmt.Sprintf("%s_%s", inst.NodeName, outputName)
				}

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
