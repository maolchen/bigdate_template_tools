package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// ============================================================
// 配置结构 - 极简版
// ============================================================

// Config 根配置
type Config struct {
	Global        GlobalConfig            `yaml:"global"`
	Nodes         map[string]NodeConfig   `yaml:"nodes"`
	Services      map[string]ServiceConfig `yaml:"services"`
	NodeOverrides map[string]map[string]interface{} `yaml:"node_overrides"`
}

// GlobalConfig 全局配置
type GlobalConfig map[string]interface{}

// NodeConfig 节点配置
type NodeConfig struct {
	IP       string `yaml:"ip"`
	Hostname string `yaml:"hostname"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Nodes        []string                 `yaml:"nodes"`
	IDAutoDerive bool                     `yaml:"id_auto_derive"`
	Vars         map[string]interface{}   `yaml:"vars"`
}

// RenderContext 模板渲染上下文
type RenderContext struct {
	Global       map[string]interface{}
	Service      string
	ServiceVars  map[string]interface{}
	IP           string
	Hostname     string
	NodeName     string  // 节点别名（如node1）
	NodeIndex    int     // 节点在该服务中的索引（用于自动推导ID）
	AutoID       int     // 自动推导的ID（索引+1）
	HostVars     map[string]interface{}
	Derived      map[string]interface{}
	AllServices  map[string]ServiceConfig
	AllNodes     map[string]NodeConfig
}

// ============================================================
// 主程序
// ============================================================

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.Create("run.log")
	if err != nil {
		log.Fatalf("创建日志文件失败: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	
	log.Println("========================================")
	log.Println("大数据平台配置生成器 v3.0 - 极简版")
	log.Println("========================================")

	// 1. 加载配置
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	log.Printf("✅ 配置加载成功: %d 个节点, %d 个服务", len(config.Nodes), len(config.Services))

	// 2. 扫描模板
	templates, err := scanTemplates("templates")
	if err != nil {
		log.Fatalf("❌ 扫描模板失败: %v", err)
	}
	log.Printf("✅ 发现 %d 个模板文件", len(templates))

	// 3. 构建主机-服务映射
	hostServices := buildHostServiceMap(config)
	log.Printf("✅ 构建主机-服务映射完成，共 %d 台主机", len(hostServices))

	// 5. 计算衍生变量
	derived := computeDerivedVars(config)
	log.Printf("✅ 衍生变量计算完成")

	// 6. 为每台主机生成配置
	for ip, serviceList := range hostServices {
		log.Printf("\n📦 处理主机: %s (部署 %d 个服务)", ip, len(serviceList))
		
		outputDir := filepath.Join("output", ip)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Printf("❌ 创建输出目录失败: %v", err)
			continue
		}

		for _, svc := range serviceList {
			serviceName := svc.ServiceName
			nodeName := svc.NodeName
			nodeIndex := svc.NodeIndex
			autoID := svc.AutoID

			serviceConfig, exists := config.Services[serviceName]
			if !exists {
				log.Printf("⚠️ 服务 %s 未定义，跳过", serviceName)
				continue
			}

			nodeConfig, exists := config.Nodes[nodeName]
			if !exists {
				log.Printf("⚠️ 节点 %s 未定义，跳过", nodeName)
				continue
			}

			// 构建主机变量（包含自动推导的ID）
			hostVars := make(map[string]interface{})
			
			// 应用节点特化配置
			if config.NodeOverrides != nil {
				if nodeOverrides, ok := config.NodeOverrides[nodeName]; ok {
					if svcOverrides, ok := nodeOverrides[serviceName].(map[string]interface{}); ok {
						for k, v := range svcOverrides {
							hostVars[k] = v
						}
					}
				}
			}

			// 如果开启自动推导ID，注入到hostVars
			if serviceConfig.IDAutoDerive {
				hostVars["auto_id"] = autoID
				// 根据服务类型设置特定的ID字段名
				switch serviceName {
				case "zookeeper":
					hostVars["myid"] = autoID
				case "kafka":
					hostVars["broker_id"] = autoID
				case "hadoop_namenode":
					hostVars["namenode_id"] = fmt.Sprintf("nn%d", autoID)
				case "doris_fe":
					hostVars["fe_id"] = autoID
				default:
					hostVars["instance_id"] = autoID
				}
			}

			// 构建渲染上下文
			ctx := RenderContext{
				Global:      config.Global,
				Service:     serviceName,
				ServiceVars: serviceConfig.Vars,
				IP:          nodeConfig.IP,
				Hostname:    nodeConfig.Hostname,
				NodeName:    nodeName,
				NodeIndex:   nodeIndex,
				AutoID:      autoID,
				HostVars:    hostVars,
				Derived:     derived,
				AllServices: config.Services,
				AllNodes:    config.Nodes,
			}

			// 渲染模板
			for _, tmpl := range templates {
				if tmpl.ServiceName != serviceName {
					continue
				}

				outputPath := filepath.Join(outputDir, serviceName, tmpl.OutputName)
				if err := renderTemplate(tmpl, outputPath, ctx); err != nil {
					log.Printf("❌ 渲染模板失败 %s: %v", tmpl.TemplatePath, err)
					continue
				}
				log.Printf("  ✅ 生成: %s/%s/%s", ip, serviceName, tmpl.OutputName)
			}
		}
	}

	// 7. 生成总览
	generateOverview(config, hostServices)

	log.Println("\n========================================")
	log.Println("🎉 配置生成完成！")
	log.Println("========================================")
}

// HostServiceInfo 主机-服务映射信息
type HostServiceInfo struct {
	ServiceName string
	NodeName    string
	NodeIndex   int
	AutoID      int
}

// ============================================================
// 配置解析
// ============================================================

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	config := &Config{
		Global:        make(GlobalConfig),
		Nodes:         make(map[string]NodeConfig),
		Services:      make(map[string]ServiceConfig),
		NodeOverrides: make(map[string]map[string]interface{}),
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return config, nil
}

// ============================================================
// 模板处理
// ============================================================

type TemplateInfo struct {
	ServiceName  string
	TemplatePath string
	OutputName   string
}

func scanTemplates(root string) ([]TemplateInfo, error) {
	var templates []TemplateInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) < 2 {
			return nil
		}

		templates = append(templates, TemplateInfo{
			ServiceName:  parts[0],
			TemplatePath: path,
			OutputName:   strings.TrimSuffix(parts[len(parts)-1], ".tmpl"),
		})
		return nil
	})

	return templates, err
}

// ============================================================
// 主机-服务映射
// ============================================================

func buildHostServiceMap(config *Config) map[string][]HostServiceInfo {
	hostServices := make(map[string][]HostServiceInfo)

	for serviceName, serviceConfig := range config.Services {
		for idx, nodeName := range serviceConfig.Nodes {
			nodeConfig, exists := config.Nodes[nodeName]
			if !exists {
				continue
			}

			autoID := idx + 1
			hostServices[nodeConfig.IP] = append(hostServices[nodeConfig.IP], HostServiceInfo{
				ServiceName: serviceName,
				NodeName:    nodeName,
				NodeIndex:   idx,
				AutoID:      autoID,
			})
		}
	}

	return hostServices
}

// ============================================================
// 衍生变量计算
// ============================================================

func computeDerivedVars(config *Config) map[string]interface{} {
	derived := make(map[string]interface{})

	// 为每个服务提取节点信息
	for serviceName, serviceConfig := range config.Services {
		var ips []string
		var hostnames []string
		var nodes []map[string]interface{}

		for idx, nodeName := range serviceConfig.Nodes {
			nodeConfig, exists := config.Nodes[nodeName]
			if !exists {
				continue
			}
			ips = append(ips, nodeConfig.IP)
			hostnames = append(hostnames, nodeConfig.Hostname)
			nodes = append(nodes, map[string]interface{}{
				"name":     nodeName,
				"ip":       nodeConfig.IP,
				"hostname": nodeConfig.Hostname,
				"index":    idx,
				"auto_id":  idx + 1,
			})
		}

		derived[serviceName+"_nodes"] = nodes
		derived[serviceName+"_ips"] = ips
		derived[serviceName+"_hostnames"] = hostnames

		// 生成节点:port列表
		if clientPort, ok := serviceConfig.Vars["client_port"].(int); ok {
			var nodesWithPort []string
			for _, ip := range ips {
				nodesWithPort = append(nodesWithPort, fmt.Sprintf("%s:%d", ip, clientPort))
			}
			derived[serviceName+"_connect_string"] = strings.Join(nodesWithPort, ",")
		}

		if brokerPort, ok := serviceConfig.Vars["broker_port"].(int); ok {
			var brokers []string
			for _, ip := range ips {
				brokers = append(brokers, fmt.Sprintf("%s:%d", ip, brokerPort))
			}
			derived[serviceName+"_brokers"] = strings.Join(brokers, ",")
		}
	}

	// 特殊处理：ZK连接串
	if zkNodes, ok := derived["zookeeper_nodes"].([]map[string]interface{}); ok {
		zkService, exists := config.Services["zookeeper"]
		if exists {
			clientPort := 2181
			if cp, ok := zkService.Vars["client_port"].(int); ok {
				clientPort = cp
			}
			var zkConnect []string
			for _, node := range zkNodes {
				zkConnect = append(zkConnect, fmt.Sprintf("%s:%d", node["ip"], clientPort))
			}
			derived["zk_connect_string"] = strings.Join(zkConnect, ",")
		}
	}

	// 特殊处理：Kafka Brokers
	if kafkaBrokers, ok := derived["kafka_brokers"].(string); ok {
		derived["kafka_broker_list"] = kafkaBrokers
	}

	// 特殊处理：HDFS NameNode地址
	if nnNodes, ok := derived["hadoop_namenode_nodes"].([]map[string]interface{}); ok {
		nnService, exists := config.Services["hadoop_namenode"]
		if exists {
			rpcPort := 9820
			if rp, ok := nnService.Vars["namenode_port"].(int); ok {
				rpcPort = rp
			}
			var nnAddresses []string
			for _, node := range nnNodes {
				nnAddresses = append(nnAddresses, fmt.Sprintf("%s:%d", node["ip"], rpcPort))
			}
			derived["namenode_rpc_addresses"] = strings.Join(nnAddresses, ",")
		}
	}

	// 特殊处理：Spark Master URL
	if sparkNodes, ok := derived["spark_master_nodes"].([]map[string]interface{}); ok {
		sparkService, exists := config.Services["spark_master"]
		if exists && len(sparkNodes) > 0 {
			masterPort := 7077
			if mp, ok := sparkService.Vars["master_port"].(int); ok {
				masterPort = mp
			}
			derived["spark_master_url"] = fmt.Sprintf("spark://%s:%d", sparkNodes[0]["ip"], masterPort)
		}
	}

	// 特殊处理：ES种子节点
	if esNodes, ok := derived["elasticsearch_nodes"].([]map[string]interface{}); ok {
		esService, exists := config.Services["elasticsearch"]
		if exists {
			transportPort := 9300
			if tp, ok := esService.Vars["transport_port"].(int); ok {
				transportPort = tp
			}
			var seedHosts []string
			for _, node := range esNodes {
				seedHosts = append(seedHosts, fmt.Sprintf("%s:%d", node["ip"], transportPort))
			}
			derived["es_seed_hosts"] = strings.Join(seedHosts, ",")
		}
	}

	return derived
}

// ============================================================
// 模板渲染
// ============================================================

func renderTemplate(tmplInfo TemplateInfo, outputPath string, ctx RenderContext) error {
	tmplContent, err := os.ReadFile(tmplInfo.TemplatePath)
	if err != nil {
		return fmt.Errorf("读取模板文件失败: %w", err)
	}

	funcMap := template.FuncMap{
		"join":      strings.Join,
		"add":       func(a, b int) int { return a + b },
		"sub":       func(a, b int) int { return a - b },
		"mul":       func(a, b int) int { return a * b },
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"replace":   strings.ReplaceAll,
		"default":   func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"printf":    fmt.Sprintf,
		// 获取全局变量
		"globalVar": func(varName string) interface{} {
			return ctx.Global[varName]
		},
		// 获取服务节点列表
		"serviceNodes": func(serviceName string) []map[string]interface{} {
			if nodes, ok := ctx.Derived[serviceName+"_nodes"].([]map[string]interface{}); ok {
				return nodes
			}
			return nil
		},
		// 获取服务IP列表
		"serviceIPs": func(serviceName string) []string {
			if ips, ok := ctx.Derived[serviceName+"_ips"].([]string); ok {
				return ips
			}
			return nil
		},
		// 获取服务变量
		"serviceVar": func(serviceName, varName string) interface{} {
			if svc, exists := ctx.AllServices[serviceName]; exists {
				return svc.Vars[varName]
			}
			return nil
		},
		// 获取节点配置
		"nodeConfig": func(nodeName string) map[string]interface{} {
			if node, exists := ctx.AllNodes[nodeName]; exists {
				return map[string]interface{}{
					"ip":       node.IP,
					"hostname": node.Hostname,
				}
			}
			return nil
		},
	}

	tmpl, err := template.New(tmplInfo.OutputName).Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("模板解析失败: %w", err)
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, ctx); err != nil {
		return fmt.Errorf("模板渲染失败: %w", err)
	}

	if strings.HasSuffix(outputPath, ".sh") {
		os.Chmod(outputPath, 0755)
	}

	return nil
}

// ============================================================
// 生成总览文件
// ============================================================

func generateOverview(config *Config, hostServices map[string][]HostServiceInfo) {
	var content strings.Builder
	
	content.WriteString("# 大数据平台部署总览\n\n")
	content.WriteString("## 节点列表\n\n")
	content.WriteString("| 节点名 | IP | 主机名 |\n")
	content.WriteString("|---|---|---|\n")
	
	// 按节点名排序
	var nodeNames []string
	for name := range config.Nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)
	
	for _, name := range nodeNames {
		node := config.Nodes[name]
		content.WriteString(fmt.Sprintf("| %s | %s | %s |\n", name, node.IP, node.Hostname))
	}
	
	content.WriteString("\n## 服务拓扑\n\n")
	content.WriteString("| 服务名 | 节点数 | 部署节点 | 自动ID |\n")
	content.WriteString("|---|---|---|---|\n")
	
	var serviceNames []string
	for name := range config.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)
	
	for _, name := range serviceNames {
		svc := config.Services[name]
		nodeList := strings.Join(svc.Nodes, ", ")
		idStatus := "否"
		if svc.IDAutoDerive {
			idStatus = "✅"
		}
		content.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", name, len(svc.Nodes), nodeList, idStatus))
	}
	
	content.WriteString("\n## 主机部署明细\n\n")
	content.WriteString("| IP | 主机名 | 部署服务 |\n")
	content.WriteString("|---|---|---|\n")
	
	// 按IP排序
	var ips []string
	for ip := range hostServices {
		ips = append(ips, ip)
	}
	sort.Strings(ips)
	
	for _, ip := range ips {
		services := hostServices[ip]
		var serviceNames []string
		var hostname string
		for _, s := range services {
			serviceNames = append(serviceNames, s.ServiceName)
			if hostname == "" {
				if node, exists := config.Nodes[s.NodeName]; exists {
					hostname = node.Hostname
				}
			}
		}
		content.WriteString(fmt.Sprintf("| %s | %s | %s |\n", ip, hostname, strings.Join(serviceNames, ", ")))
	}
	
	content.WriteString("\n## 交付步骤\n\n")
	content.WriteString("1. 将 `output/<ip>` 目录上传到对应主机\n")
	content.WriteString("2. 在每台主机上执行各服务的 `install.sh`\n")
	content.WriteString("3. 启动服务\n")
	
	os.WriteFile("output/README.md", []byte(content.String()), 0644)
}
