package generator

import (
	"fmt"

	"config-generator/config"
	"config-generator/utils"
)

// BuildServiceInstances 构建服务实例列表
func BuildServiceInstances(cfg *config.Config) []config.ServiceInstance {
	var instances []config.ServiceInstance

	for serviceName, topo := range cfg.ServiceTop {
		// 获取服务配置
		serviceConfig, exists := cfg.ServerConfig[serviceName]
		if !exists {
			// 如果没有配置，使用空配置
			serviceConfig = config.ServiceConfig{
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
				autoID = utils.FormatID(serviceConfig.IDFormat, idx)
			}

			instances = append(instances, config.ServiceInstance{
				ServiceName: serviceName,
				NodeName:    nodeName,
				Node: config.NodeInstance{
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
