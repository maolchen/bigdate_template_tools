package main

import (
	"flag"
	"fmt"
	"os"

	"config-generator/config"
	"config-generator/generator"
	"config-generator/server"
	"config-generator/utils"
)

func main() {
	// 解析命令行参数
	webMode := flag.Bool("web", false, "启动 Web 服务模式")
	port := flag.String("port", "", "Web 服务端口（默认 5000）")
	flag.Parse()

	if *webMode {
		runWebMode(*port)
	} else {
		runCLIMode()
	}
}

// runCLIMode 命令行模式
func runCLIMode() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
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
	nodeUsage := make(map[string][]string)
	for _, inst := range instances {
		nodeUsage[inst.NodeName] = append(nodeUsage[inst.NodeName], inst.ServiceName)
	}

	for nodeName, services := range nodeUsage {
		if node, ok := cfg.Nodes[nodeName]; ok {
			fmt.Printf("%s (%s):\n", nodeName, node.IP)
			for _, svc := range services {
				fmt.Printf("  - %s\n", svc)
			}
		}
	}

	// 生成配置
	fmt.Println("\n============================================")
	fmt.Println("生成配置文件")
	fmt.Println("============================================")
	if _, err := generator.GenerateOutputs(cfg, instances, "output"); err != nil {
		fmt.Printf("生成配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ 配置生成完成")
}

// runWebMode Web 模式
func runWebMode(port string) {
	// 确定工作目录
	workDir := utils.GetWorkDir()
	fmt.Printf("工作目录: %s\n", workDir)

	// 创建并启动服务器
	srv := server.NewServer(workDir)
	if err := srv.Start(port); err != nil {
		fmt.Printf("启动服务失败: %v\n", err)
		os.Exit(1)
	}
}
