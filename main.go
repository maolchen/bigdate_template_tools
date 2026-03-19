package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"log"

	"gopkg.in/yaml.v3"
)

// ============================================================
// 配置结构 - 完全动态，不硬编码任何服务名
// ============================================================

// Config 根配置
type Config struct {
	All     AllConfig               `yaml:"all"`
	Services map[string]ServiceConfig `yaml:",inline"` // 动态解析所有服务
}

// AllConfig 全局配置
type AllConfig struct {
	Vars map[string]interface{} `yaml:"vars"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Vars  map[string]interface{} `yaml:"vars"`
	Hosts map[string]HostConfig  `yaml:"hosts"`
}

// HostConfig 主机配置
type HostConfig map[string]interface{}

// ============================================================
// 渲染上下文 - 为模板提供结构化数据
// ============================================================

// RenderContext 模板渲染上下文
type RenderContext struct {
	// 全局变量
	Vars map[string]interface{}
	
	// 当前服务名称
	Service string
	
	// 当前服务级变量
	ServiceVars map[string]interface{}
	
	// 当前主机IP
	IP string
	
	// 当前主机配置（主机级变量）
	HostVars map[string]interface{}
	
	// 当前主机名
	Hostname string
	
	// 所有服务配置（用于跨服务引用）
	AllServices map[string]ServiceConfig
	
	// 衍生变量（自动计算）
	Derived map[string]interface{}
}

// ============================================================
// 主程序
// ============================================================

func main() {
	// 初始化日志
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.Create("run.log")
	if err != nil {
		log.Fatalf("创建日志文件失败: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	
	log.Println("========================================")
	log.Println("大数据平台配置生成器 v2.0 - 精简版")
	log.Println("========================================")

	// 1. 加载配置
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	log.Printf("✅ 配置加载成功")

	// 2. 扫描模板目录
	templates, err := scanTemplates("templates")
	if err != nil {
		log.Fatalf("扫描模板失败: %v", err)
	}
	log.Printf("✅ 发现 %d 个模板文件", len(templates))

	// 3. 构建主机-服务映射（每个IP部署了哪些服务）
	hostServices := buildHostServiceMap(config)
	log.Printf("✅ 构建主机-服务映射完成，共 %d 台主机", len(hostServices))

	// 4. 预计算所有衍生变量
	derived := computeDerivedVars(config)
	log.Printf("✅ 衍生变量计算完成")

	// 5. 为每台主机生成配置
	for ip, services := range hostServices {
		log.Printf("\n📦 处理主机: %s (部署 %d 个服务)", ip, len(services))
		
		outputDir := filepath.Join("output", ip)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Printf("❌ 创建输出目录失败: %v", err)
			continue
		}

		// 为该主机的每个服务生成配置
		for _, serviceName := range services {
			serviceConfig, exists := config.Services[serviceName]
			if !exists {
				log.Printf("⚠️ 服务 %s 未定义，跳过", serviceName)
				continue
			}

			hostConfig, exists := serviceConfig.Hosts[ip]
			if !exists {
				log.Printf("⚠️ 主机 %s 不在服务 %s 中，跳过", ip, serviceName)
				continue
			}

			// 构建渲染上下文
			ctx := RenderContext{
				Vars:        config.All.Vars,
				Service:     serviceName,
				ServiceVars: serviceConfig.Vars,
				IP:          ip,
				HostVars:    hostConfig,
				AllServices: config.Services,
				Derived:     derived,
			}

			// 提取主机名
			if hostname, ok := hostConfig["hostname"].(string); ok {
				ctx.Hostname = hostname
			}

			// 渲染该服务的所有模板
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

	// 6. 生成总览文件
	generateOverview(config, hostServices)

	log.Println("\n========================================")
	log.Println("🎉 配置生成完成！")
	log.Println("========================================")
}

// ============================================================
// 配置解析
// ============================================================

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 先解析为通用map，提取all和services
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	config := &Config{
		Services: make(map[string]ServiceConfig),
	}

	// 解析all字段
	if all, ok := rawConfig["all"].(map[string]interface{}); ok {
		if vars, ok := all["vars"].(map[string]interface{}); ok {
			config.All.Vars = vars
		}
	}

	// 解析其他字段作为服务
	for key, value := range rawConfig {
		if key == "all" {
			continue
		}

		serviceMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		service := ServiceConfig{
			Vars:  make(map[string]interface{}),
			Hosts: make(map[string]HostConfig),
		}

		// 解析vars
		if vars, ok := serviceMap["vars"].(map[string]interface{}); ok {
			service.Vars = vars
		}

		// 解析hosts
		if hosts, ok := serviceMap["hosts"].(map[string]interface{}); ok {
			for ip, hostConfig := range hosts {
				if hc, ok := hostConfig.(map[string]interface{}); ok {
					service.Hosts[ip] = hc
				}
			}
		}

		config.Services[key] = service
	}

	return config, nil
}

// ============================================================
// 模板处理
// ============================================================

type TemplateInfo struct {
	ServiceName string
	TemplatePath string
	OutputName  string
}

func scanTemplates(root string) ([]TemplateInfo, error) {
	var templates []TemplateInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 只处理.tmpl文件
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// 提取服务名：templates/Zookeeper/zoo.cfg.tmpl -> Zookeeper
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) < 2 {
			return nil
		}

		serviceName := parts[0]
		outputName := strings.TrimSuffix(parts[len(parts)-1], ".tmpl")

		templates = append(templates, TemplateInfo{
			ServiceName: serviceName,
			TemplatePath: path,
			OutputName:  outputName,
		})

		return nil
	})

	return templates, err
}

// ============================================================
// 主机-服务映射
// ============================================================

func buildHostServiceMap(config *Config) map[string][]string {
	hostServices := make(map[string][]string)

	for serviceName, serviceConfig := range config.Services {
		for ip := range serviceConfig.Hosts {
			hostServices[ip] = append(hostServices[ip], serviceName)
		}
	}

	return hostServices
}

// ============================================================
// 衍生变量计算
// ============================================================

func computeDerivedVars(config *Config) map[string]interface{} {
	derived := make(map[string]interface{})

	// 1. 为每个服务自动提取节点列表（排序后）
	for serviceName, serviceConfig := range config.Services {
		var nodes []string
		for ip := range serviceConfig.Hosts {
			nodes = append(nodes, ip)
		}
		// 排序节点列表，保证顺序一致
		sort.Strings(nodes)
		derived[serviceName+"_nodes"] = nodes

		// 2. 如果服务有port变量，生成节点:port列表
		if port, ok := serviceConfig.Vars["port"].(int); ok {
			var nodesWithPort []string
			for _, ip := range nodes {
				nodesWithPort = append(nodesWithPort, fmt.Sprintf("%s:%d", ip, port))
			}
			derived[serviceName+"_nodes_with_port"] = nodesWithPort
		}

		// 3. 如果有client_port变量
		if clientPort, ok := serviceConfig.Vars["client_port"].(int); ok {
			var nodesWithPort []string
			for _, ip := range nodes {
				nodesWithPort = append(nodesWithPort, fmt.Sprintf("%s:%d", ip, clientPort))
			}
			derived[serviceName+"_nodes_with_client_port"] = nodesWithPort
		}
	}

	// 4. 特殊处理：Zookeeper连接串
	if zkNodes, ok := derived["zookeeper_nodes"].([]string); ok {
		zkService, exists := config.Services["zookeeper"]
		if exists {
			clientPort := 2181 // 默认端口
			if cp, ok := zkService.Vars["client_port"].(int); ok {
				clientPort = cp
			}
			var zkConnect []string
			for _, ip := range zkNodes {
				zkConnect = append(zkConnect, fmt.Sprintf("%s:%d", ip, clientPort))
			}
			derived["zk_connect_string"] = strings.Join(zkConnect, ",")
		}
	}

	// 5. 特殊处理：Kafka Broker列表
	if kafkaNodes, ok := derived["kafka_nodes"].([]string); ok {
		kafkaService, exists := config.Services["kafka"]
		if exists {
			brokerPort := 9092 // 默认端口
			if bp, ok := kafkaService.Vars["broker_port"].(int); ok {
				brokerPort = bp
			}
			var brokers []string
			for _, ip := range kafkaNodes {
				brokers = append(brokers, fmt.Sprintf("%s:%d", ip, brokerPort))
			}
			derived["kafka_brokers"] = strings.Join(brokers, ",")
		}
	}

	// 6. 特殊处理：HDFS NameNode RPC地址
	if nnNodes, ok := derived["hadoop_namenode_nodes"].([]string); ok {
		hdfsService, exists := config.Services["hadoop_namenode"]
		if exists {
			rpcPort := 9820 // 默认端口
			if rp, ok := hdfsService.Vars["namenode_port"].(int); ok {
				rpcPort = rp
			}
			var nnAddresses []string
			for _, ip := range nnNodes {
				nnAddresses = append(nnAddresses, fmt.Sprintf("%s:%d", ip, rpcPort))
			}
			derived["namenode_rpc_addresses"] = strings.Join(nnAddresses, ",")
		}
	}

	// 7. 特殊处理：Spark Master URL
	if sparkNodes, ok := derived["spark_master_nodes"].([]string); ok {
		sparkService, exists := config.Services["spark_master"]
		if exists {
			masterPort := 7077 // 默认端口
			if mp, ok := sparkService.Vars["master_port"].(int); ok {
				masterPort = mp
			}
			var masterURLs []string
			for _, ip := range sparkNodes {
				masterURLs = append(masterURLs, fmt.Sprintf("spark://%s:%d", ip, masterPort))
			}
			derived["spark_master_url"] = strings.Join(masterURLs, ",")
		}
	}

	// 8. 特殊处理：ES种子节点
	if esNodes, ok := derived["elasticsearch_nodes"].([]string); ok {
		esService, exists := config.Services["elasticsearch"]
		if exists {
			transportPort := 9300 // 默认端口
			if tp, ok := esService.Vars["transport_port"].(int); ok {
				transportPort = tp
			}
			var seedHosts []string
			for _, ip := range esNodes {
				seedHosts = append(seedHosts, fmt.Sprintf("%s:%d", ip, transportPort))
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
	// 读取模板文件
	tmplContent, err := os.ReadFile(tmplInfo.TemplatePath)
	if err != nil {
		return fmt.Errorf("读取模板文件失败: %w", err)
	}

	// 创建模板函数
	funcMap := template.FuncMap{
		"join":      strings.Join,
		"add":       func(a, b int) int { return a + b },
		"sub":       func(a, b int) int { return a - b },
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"replace":   strings.ReplaceAll,
		"default":   func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		// 字符串拼接
		"printf":    fmt.Sprintf,
		// 获取服务节点列表
		"serviceNodes": func(serviceName string) []string {
			if nodes, ok := ctx.Derived[serviceName+"_nodes"].([]string); ok {
				return nodes
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
		// 获取全局变量
		"globalVar": func(varName string) interface{} {
			return ctx.Vars[varName]
		},
	}

	// 创建模板
	tmpl, err := template.New(tmplInfo.OutputName).Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("模板解析失败: %w", err)
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 创建输出文件
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer f.Close()

	// 渲染模板
	if err := tmpl.Execute(f, ctx); err != nil {
		return fmt.Errorf("模板渲染失败: %w", err)
	}

	// 如果是shell脚本，设置可执行权限
	if strings.HasSuffix(outputPath, ".sh") {
		if err := os.Chmod(outputPath, 0755); err != nil {
			return fmt.Errorf("设置权限失败: %w", err)
		}
	}

	return nil
}

// ============================================================
// 生成总览文件
// ============================================================

func generateOverview(config *Config, hostServices map[string][]string) {
	var content strings.Builder
	
	content.WriteString("# 大数据平台部署总览\n\n")
	content.WriteString(fmt.Sprintf("生成时间: %s\n\n", "2025-01-XX"))
	
	content.WriteString("## 全局变量\n\n")
	content.WriteString("| 变量名 | 值 |\n")
	content.WriteString("|---|---|\n")
	for k, v := range config.All.Vars {
		content.WriteString(fmt.Sprintf("| %s | %v |\n", k, v))
	}
	
	content.WriteString("\n## 集群拓扑\n\n")
	content.WriteString("| IP | 主机名 | 部署服务 |\n")
	content.WriteString("|---|---|---|\n")
	
	for ip, services := range hostServices {
		hostname := "N/A"
		// 尝试从任一服务获取主机名
		for _, svc := range services {
			if svcConfig, exists := config.Services[svc]; exists {
				if hostConfig, exists := svcConfig.Hosts[ip]; exists {
					if hn, ok := hostConfig["hostname"].(string); ok {
						hostname = hn
						break
					}
				}
			}
		}
		content.WriteString(fmt.Sprintf("| %s | %s | %s |\n", ip, hostname, strings.Join(services, ", ")))
	}
	
	content.WriteString("\n## 服务统计\n\n")
	content.WriteString("| 服务名 | 节点数 | 节点列表 |\n")
	content.WriteString("|---|---|---|\n")
	
	for serviceName, serviceConfig := range config.Services {
		nodes := make([]string, 0, len(serviceConfig.Hosts))
		for ip := range serviceConfig.Hosts {
			nodes = append(nodes, ip)
		}
		content.WriteString(fmt.Sprintf("| %s | %d | %s |\n", serviceName, len(nodes), strings.Join(nodes, ", ")))
	}
	
	os.WriteFile("output/README.md", []byte(content.String()), 0644)
}
