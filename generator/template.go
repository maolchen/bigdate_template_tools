package generator

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"config-generator/config"
	"config-generator/utils"
)

// RenderTemplate 渲染模板
func RenderTemplate(tmplPath string, ctx config.Context, cfg *config.Config) (string, error) {
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
		"serviceNodes": func(serviceName string) []config.ServiceInstance {
			var result []config.ServiceInstance
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
					port := utils.GetNestedPort(inst.Vars, portField)
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
					port := utils.GetNestedPort(inst.Vars, portField)
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
		"serviceConfig": func(serviceName string) config.ServiceConfig {
			return cfg.ServerConfig[serviceName]
		},

		// 获取节点信息
		"nodeInfo": func(nodeName string) *config.Node {
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
