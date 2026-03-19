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
// 配置结构 - v4 拓扑配置分离版
// ============================================================

// Config 根配置
type Config struct {
	Global        map[string]interface{}            `yaml:"global"`
	Nodes         map[string]NodeConfig             `yaml:"nodes"`
	ServiceTop    map[string]ServiceTopConfig       `yaml:"serviceTop"`
	ServerConfig  map[string]map[string]interface{} `yaml:"serverConfig"`
	NodeOverrides map[string]map[string]interface{} `yaml:"nodeOverrides"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	IP       string `yaml:"ip"`
	Hostname string `yaml:"hostname"`
}

// ServiceTopConfig 服务拓扑配置
type ServiceTopConfig struct {
	Nodes        []string `yaml:"nodes"`
	IDAutoDerive bool     `yaml:"id_auto_derive"`
}

// RenderContext 模板渲染上下文
type RenderContext struct {
	Global       map[string]interface{}
	Service      string
	ServiceVars  map[string]interface{}
	IP           string
	Hostname     string
	NodeName     string
	NodeIndex    int
	AutoID       int
	HostVars     map[string]interface{}
	Derived      map[string]interface{}
	ServiceTop   map[string]ServiceTopConfig
	ServerConfig map[string]map[string]interface{}
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
	log.Println("大数据平台配置生成器 v4.0 - 拓扑配置分离版")
	log.Println("========================================")

	// 1. 加载配置
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	log.Printf("✅ 配置加载成功: %d 个节点, %d 个服务拓扑", len(config.Nodes), len(config.ServiceTop))

	// 2. 扫描模板
	templates, err := scanTemplates("templates")
	if err != nil {
		log.Fatalf("❌ 扫描模板失败: %v", err)
	}
	log.Printf("✅ 发现 %d 个模板文件", len(templates))

	// 3. 构建主机-服务映射
	hostServices := buildHostServiceMap(config)
	log.Printf("✅ 构建主机-服务映射完成，共 %d 台主机", len(hostServices))

	// 4. 计算衍生变量
	derived := computeDerivedVars(config)
	log.Printf("✅ 衍生变量计算完成")

	// 5. 为每台主机生成配置
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

			// 获取服务拓扑配置
			serviceTop, exists := config.ServiceTop[serviceName]
			if !exists {
				log.Printf("⚠️ 服务 %s 未定义拓扑，跳过", serviceName)
				continue
			}

			// 获取节点配置
			nodeConfig, exists := config.Nodes[nodeName]
			if !exists {
				log.Printf("⚠️ 节点 %s 未定义，跳过", nodeName)
				continue
			}

			// 获取服务配置
			serviceVars := make(map[string]interface{})
			if serverConfig, exists := config.ServerConfig[serviceName]; exists {
				serviceVars = serverConfig
			}

			// 构建主机变量
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
			if serviceTop.IDAutoDerive {
				hostVars["auto_id"] = autoID
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
				Global:       config.Global,
				Service:      serviceName,
				ServiceVars:  serviceVars,
				IP:           nodeConfig.IP,
				Hostname:     nodeConfig.Hostname,
				NodeName:     nodeName,
				NodeIndex:    nodeIndex,
				AutoID:       autoID,
				HostVars:     hostVars,
				Derived:      derived,
				ServiceTop:   config.ServiceTop,
				ServerConfig: config.ServerConfig,
				AllNodes:     config.Nodes,
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

	// 6. 生成总览
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
		Global:        make(map[string]interface{}),
		Nodes:         make(map[string]NodeConfig),
		ServiceTop:    make(map[string]ServiceTopConfig),
		ServerConfig:  make(map[string]map[string]interface{}),
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

	for serviceName, serviceTop := range config.ServiceTop {
		for idx, nodeName := range serviceTop.Nodes {
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
	for serviceName, serviceTop := range config.ServiceTop {
		var ips []string
		var hostnames []string
		var nodes []map[string]interface{}

		for idx, nodeName := range serviceTop.Nodes {
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

		// 获取服务配置
		serviceConfig, hasConfig := config.ServerConfig[serviceName]
		if !hasConfig {
			serviceConfig = make(map[string]interface{})
		}

		// 生成节点:port列表
		if clientPort, ok := serviceConfig["client_port"].(int); ok {
			var nodesWithPort []string
			for _, ip := range ips {
				nodesWithPort = append(nodesWithPort, fmt.Sprintf("%s:%d", ip, clientPort))
			}
			derived[serviceName+"_connect_string"] = strings.Join(nodesWithPort, ",")
		}

		if brokerPort, ok := serviceConfig["broker_port"].(int); ok {
			var brokers []string
			for _, ip := range ips {
				brokers = append(brokers, fmt.Sprintf("%s:%d", ip, brokerPort))
			}
			derived[serviceName+"_brokers"] = strings.Join(brokers, ",")
		}
	}

	// 特殊处理：ZK连接串
	if zkNodes, ok := derived["zookeeper_nodes"].([]map[string]interface{}); ok {
		serviceConfig, hasConfig := config.ServerConfig["zookeeper"]
		if hasConfig {
			clientPort := 2181
			if cp, ok := serviceConfig["client_port"].(int); ok {
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
		serviceConfig, hasConfig := config.ServerConfig["hadoop_namenode"]
		if hasConfig {
			rpcPort := 9820
			if rp, ok := serviceConfig["namenode_port"].(int); ok {
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
		serviceConfig, hasConfig := config.ServerConfig["spark_master"]
		if hasConfig && len(sparkNodes) > 0 {
			masterPort := 7077
			if mp, ok := serviceConfig["master_port"].(int); ok {
				masterPort = mp
			}
			derived["spark_master_url"] = fmt.Sprintf("spark://%s:%d", sparkNodes[0]["ip"], masterPort)
		}
	}

	// 特殊处理：ES种子节点
	if esNodes, ok := derived["elasticsearch_nodes"].([]map[string]interface{}); ok {
		serviceConfig, hasConfig := config.ServerConfig["elasticsearch"]
		if hasConfig {
			transportPort := 9300
			if tp, ok := serviceConfig["transport_port"].(int); ok {
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
		"globalVar": func(varName string) interface{} {
			return ctx.Global[varName]
		},
		"serviceNodes": func(serviceName string) []map[string]interface{} {
			if nodes, ok := ctx.Derived[serviceName+"_nodes"].([]map[string]interface{}); ok {
				return nodes
			}
			return nil
		},
		"serviceIPs": func(serviceName string) []string {
			if ips, ok := ctx.Derived[serviceName+"_ips"].([]string); ok {
				return ips
			}
			return nil
		},
		"serviceVar": func(serviceName, varName string) interface{} {
			if config, exists := ctx.ServerConfig[serviceName]; exists {
				return config[varName]
			}
			return nil
		},
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
	for name := range config.ServiceTop {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)
	
	for _, name := range serviceNames {
		top := config.ServiceTop[name]
		nodeList := strings.Join(top.Nodes, ", ")
		idStatus := "否"
		if top.IDAutoDerive {
			idStatus = "✅"
		}
		content.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", name, len(top.Nodes), nodeList, idStatus))
	}
	
	content.WriteString("\n## 主机部署明细\n\n")
	content.WriteString("| IP | 主机名 | 部署服务 |\n")
	content.WriteString("|---|---|---|\n")
	
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
