package generator

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"

	"config-generator/config"
	"config-generator/utils"
)

// BuildTemplateFuncMap 构建模板函数映射，供渲染和静态校验复用
func BuildTemplateFuncMap(ctx config.Context, cfg *config.Config) template.FuncMap {
	joinValues := func(values interface{}, sep string) string {
		if values == nil {
			return ""
		}
		if text, ok := values.(string); ok {
			return text
		}
		rv := reflect.ValueOf(values)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return fmt.Sprint(values)
		}
		parts := make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			parts = append(parts, fmt.Sprint(rv.Index(i).Interface()))
		}
		return strings.Join(parts, sep)
	}

	join := func(args ...interface{}) string {
		if len(args) != 2 {
			return ""
		}
		if sep, ok := args[0].(string); ok {
			return joinValues(args[1], sep)
		}
		if sep, ok := args[1].(string); ok {
			return joinValues(args[0], sep)
		}
		return joinValues(args[0], "")
	}

	return template.FuncMap{
		// =============== 基础函数 ===============
		"toUpper": strings.ToUpper,
		"toLower": strings.ToLower,
		"trim":    strings.TrimSpace,
		"replace": strings.ReplaceAll,
		"join":    join,
		"split":   strings.Split,
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
		"serviceNodes": func(serviceName string) []config.ServiceInstance {
			var result []config.ServiceInstance
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					result = append(result, inst)
				}
			}
			return result
		},

		"serviceEndpoints": func(serviceName, portField string) []string {
			var endpoints []string
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					port := utils.GetNestedPort(inst.Vars, portField)
					if port > 0 {
						endpoints = append(endpoints, fmt.Sprintf("%s:%d", inst.Node.IP, port))
					}
				}
			}
			return endpoints
		},

		"serviceEndpointsJoin": func(serviceName, portField string) string {
			endpoints := make([]string, 0)
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					port := utils.GetNestedPort(inst.Vars, portField)
					if port > 0 {
						endpoints = append(endpoints, fmt.Sprintf("%s:%d", inst.Node.IP, port))
					}
				}
			}
			return strings.Join(endpoints, ",")
		},

		"serviceIPs": func(serviceName string) []string {
			var ips []string
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					ips = append(ips, inst.Node.IP)
				}
			}
			return ips
		},

		"serviceHostnames": func(serviceName string) []string {
			var hostnames []string
			for _, inst := range ctx.AllInstances {
				if inst.ServiceName == serviceName {
					hostnames = append(hostnames, inst.Node.Hostname)
				}
			}
			return hostnames
		},

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

		"serviceVars": func(serviceName string) map[string]interface{} {
			if svc, ok := cfg.ServerConfig[serviceName]; ok {
				return svc.Vars
			}
			return make(map[string]interface{})
		},

		"serviceVar": func(serviceName, varName string) interface{} {
			if svc, ok := cfg.ServerConfig[serviceName]; ok {
				if val, exists := svc.Vars[varName]; exists {
					return val
				}
			}
			return nil
		},

		// =============== 配置访问函数 ===============
		"serviceConfig": func(serviceName string) config.ServiceConfig {
			return cfg.ServerConfig[serviceName]
		},

		"nodeInfo": func(nodeName string) *config.Node {
			if node, ok := cfg.Nodes[nodeName]; ok {
				return &node
			}
			return nil
		},
	}
}

// ParseTemplateContent 仅解析模板内容，用于静态校验
func ParseTemplateContent(name, content string, ctx config.Context, cfg *config.Config) (*template.Template, error) {
	return template.New(name).Funcs(BuildTemplateFuncMap(ctx, cfg)).Parse(content)
}

// RenderTemplate 渲染模板
func RenderTemplate(tmplPath string, ctx config.Context, cfg *config.Config) (string, error) {
	// 读取模板文件
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("读取模板失败: %w", err)
	}

	// 创建模板，添加通用模板函数（不包含任何特定服务逻辑）
	tmpl, err := ParseTemplateContent("config", string(tmplContent), ctx, cfg)
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
