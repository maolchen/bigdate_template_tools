package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	bigdata "bigdata-deploy-generator/config"
	"bigdata-deploy-generator/generator"
	"bigdata-deploy-generator/web"
)

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
	cfg, err := bigdata.LoadConfig("config.yaml")
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
	instances := generator.BuildServiceInstances(cfg)

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
					idStr = fmt.Sprintf(" [ID: %v]", inst.AutoID)
				}
				fmt.Printf("    - %s (%s%s)\n", inst.NodeName, inst.Node.IP, idStr)
			}
		}
	}

	// 统计IP复用情况
	fmt.Println("\n============================================")
	fmt.Println("IP 使用统计")
	fmt.Println("============================================")
	ipUsage := make(map[string][]string)
	for _, inst := range instances {
		ipUsage[inst.Node.IP] = append(ipUsage[inst.Node.IP], inst.ServiceName)
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
	if err := generator.GenerateOutputs(cfg, instances, outputDir); err != nil {
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
	configPath := filepath.Join(workDir, "config.yaml")
	outputDir := filepath.Join(workDir, "output")
	templatesDir := filepath.Join(workDir, "templates")
	webDir := filepath.Join(workDir, "web", "dist")

	fmt.Printf("尝试前端目录: %s\n", webDir)
	// 如果 web/dist 不存在，尝试 server/web/dist
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		webDir = filepath.Join(workDir, "server", "web", "dist")
		fmt.Printf("前端目录不存在，尝试: %s\n", webDir)
	}

	// 创建 WebServer
	server := web.NewWebServer(configPath, outputDir, templatesDir, webDir)

	// 设置检查节点是否有服务分配的回调函数
	server.CheckNodeUsage = func(nodeName string) (bool, []string) {
		var services []string
		for serviceName, topo := range server.Config.ServiceTop {
			if len(topo.Nodes) == 1 && (topo.Nodes[0] == "*" || topo.Nodes[0] == "all") {
				services = append(services, serviceName)
			} else {
				for _, node := range topo.Nodes {
					if node == nodeName {
						services = append(services, serviceName)
						break
					}
				}
			}
		}
		return len(services) > 0, services
	}

	// 加载配置
	if err := server.LoadConfig(); err != nil {
		fmt.Printf("警告: 无法加载配置文件 %s: %v\n", configPath, err)
		// 使用默认配置
		server.Config = &bigdata.Config{
			Global: map[string]interface{}{
				"user":           "bigdata",
				"group":          "bigdata",
				"install_base_dir": "/data/localization",
				"data_base_dir":    "/data",
				"java_home":       "/data/jdk",
			},
			Nodes:        make(bigdata.Nodes),
			ServiceTop:   make(bigdata.ServiceTopos),
			ServerConfig: make(bigdata.ServiceConfigs),
		}
	}

	// 创建路由
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	// 启动服务器
	if port == "" {
		port = "5000"
	}

	fmt.Printf("============================================\n")
	fmt.Printf("大数据平台配置生成器 Web 服务\n")
	fmt.Printf("============================================\n")
	fmt.Printf("工作目录: %s\n", workDir)
	fmt.Printf("配置文件: %s\n", configPath)
	fmt.Printf("模板目录: %s\n", templatesDir)
	fmt.Printf("输出目录: %s\n", outputDir)
	fmt.Printf("端口: %s\n", port)
	fmt.Printf("API: http://localhost:%s/api/config\n", port)
	fmt.Printf("Web: http://localhost:%s/\n", port)
	fmt.Printf("============================================\n")

	if err := http.ListenAndServe(":"+port, web.CORSMiddleware(mux)); err != nil {
		fmt.Printf("启动服务失败: %v\n", err)
		os.Exit(1)
	}
}
