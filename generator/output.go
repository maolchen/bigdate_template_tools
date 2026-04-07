package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"config-generator/config"
)

type SkippedService struct {
	NodeIP  string `json:"nodeIp"`
	Service string `json:"service"`
	Reason  string `json:"reason"`
}

type GenerationSummary struct {
	Results         map[string][]string `json:"results"`
	Generated       int                 `json:"generated"`
	SkippedServices []SkippedService    `json:"skippedServices"`
	Warnings        []string            `json:"warnings"`
}

// GenerateOutputs 生成所有配置文件和脚本
// 输出结构：output/<IP>/<服务名>/<配置文件>
func GenerateOutputs(cfg *config.Config, instances []config.ServiceInstance, outputDir string) (*GenerationSummary, error) {
	summary := &GenerationSummary{
		Results:         make(map[string][]string),
		SkippedServices: make([]SkippedService, 0),
		Warnings:        make([]string, 0),
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 按节点IP分组
	nodeInstances := make(map[string][]config.ServiceInstance)
	for _, inst := range instances {
		nodeInstances[inst.Node.IP] = append(nodeInstances[inst.Node.IP], inst)
	}

	// 为每个节点生成配置
	for nodeIP, svcs := range nodeInstances {
		// 创建节点目录（以IP命名）
		nodeDir := filepath.Join(outputDir, nodeIP)
		if err := os.MkdirAll(nodeDir, 0755); err != nil {
			return nil, fmt.Errorf("创建节点目录失败: %w", err)
		}

		// 为该节点上的每个服务生成配置
		for _, inst := range svcs {
			serviceName := inst.ServiceName

			// 检查模板目录是否存在
			tmplDir := filepath.Join("templates", serviceName)
			if _, err := os.Stat(tmplDir); os.IsNotExist(err) {
				warning := fmt.Sprintf("节点 %s 的服务 %s 没有模板目录，已跳过", nodeIP, serviceName)
				fmt.Printf("提示: %s\n", warning)
				summary.Warnings = append(summary.Warnings, warning)
				summary.SkippedServices = append(summary.SkippedServices, SkippedService{
					NodeIP:  nodeIP,
					Service: serviceName,
					Reason:  "模板目录不存在",
				})
				continue
			}

			// 创建服务目录
			serviceDir := filepath.Join(nodeDir, serviceName)
			if err := os.MkdirAll(serviceDir, 0755); err != nil {
				return nil, fmt.Errorf("创建服务目录失败: %w", err)
			}

			// 构建上下文
			ctx := config.Context{
				Global:       cfg.Global,
				Nodes:        cfg.Nodes,
				Instance:     inst,
				AllInstances: instances,
			}

			// 递归查找所有模板文件
			var tmplFiles []string
			err := filepath.Walk(tmplDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".tmpl") {
					tmplFiles = append(tmplFiles, path)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("查找模板失败: %w", err)
			}

			for _, tmplFile := range tmplFiles {
				// 渲染模板
				content, err := RenderTemplate(tmplFile, ctx, cfg)
				if err != nil {
					return nil, fmt.Errorf("渲染模板 %s 失败: %w", tmplFile, err)
				}

				// 计算相对路径，保持子目录结构
				relPath, err := filepath.Rel(tmplDir, tmplFile)
				if err != nil {
					return nil, fmt.Errorf("计算相对路径失败: %w", err)
				}

				// 生成输出文件名
				outputName := strings.TrimSuffix(relPath, ".tmpl")
				outputPath := filepath.Join(serviceDir, outputName)

				// 创建子目录（如果需要）
				outputSubDir := filepath.Dir(outputPath)
				if outputSubDir != "." && outputSubDir != serviceDir {
					if err := os.MkdirAll(outputSubDir, 0755); err != nil {
						return nil, fmt.Errorf("创建输出子目录失败: %w", err)
					}
				}

				// 写入文件
				if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
					return nil, fmt.Errorf("写入文件失败: %w", err)
				}

				fileInNode, err := filepath.Rel(nodeDir, outputPath)
				if err != nil {
					return nil, fmt.Errorf("计算节点内输出路径失败: %w", err)
				}
				summary.Results[nodeIP] = append(summary.Results[nodeIP], filepath.ToSlash(fileInNode))
				summary.Generated++

				fmt.Printf("生成: %s\n", outputPath)
			}
		}
	}

	for nodeIP := range summary.Results {
		sort.Strings(summary.Results[nodeIP])
	}

	return summary, nil
}
