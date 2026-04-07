package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"config-generator/config"
)

// GenerateOutputs 生成所有配置文件和脚本
// 输出结构：output/<IP>/<服务名>/<配置文件>
func GenerateOutputs(cfg *config.Config, instances []config.ServiceInstance, outputDir string) error {
	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
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
			return fmt.Errorf("创建节点目录失败: %w", err)
		}

		// 为该节点上的每个服务生成配置
		for _, inst := range svcs {
			serviceName := inst.ServiceName

			// 检查模板目录是否存在
			tmplDir := filepath.Join("templates", serviceName)
			if _, err := os.Stat(tmplDir); os.IsNotExist(err) {
				fmt.Printf("提示: 节点 %s 的服务 %s 没有模板目录，跳过\n", nodeIP, serviceName)
				continue
			}

			// 创建服务目录
			serviceDir := filepath.Join(nodeDir, serviceName)
			if err := os.MkdirAll(serviceDir, 0755); err != nil {
				return fmt.Errorf("创建服务目录失败: %w", err)
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
				return fmt.Errorf("查找模板失败: %w", err)
			}

			for _, tmplFile := range tmplFiles {
				// 渲染模板
				content, err := RenderTemplate(tmplFile, ctx, cfg)
				if err != nil {
					return fmt.Errorf("渲染模板 %s 失败: %w", tmplFile, err)
				}

				// 计算相对路径，保持子目录结构
				relPath, err := filepath.Rel(tmplDir, tmplFile)
				if err != nil {
					return fmt.Errorf("计算相对路径失败: %w", err)
				}

				// 生成输出文件名
				outputName := strings.TrimSuffix(relPath, ".tmpl")
				outputPath := filepath.Join(serviceDir, outputName)

				// 创建子目录（如果需要）
				outputSubDir := filepath.Dir(outputPath)
				if outputSubDir != "." && outputSubDir != serviceDir {
					if err := os.MkdirAll(outputSubDir, 0755); err != nil {
						return fmt.Errorf("创建输出子目录失败: %w", err)
					}
				}

				// 写入文件
				if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
					return fmt.Errorf("写入文件失败: %w", err)
				}

				fmt.Printf("生成: %s\n", outputPath)
			}
		}
	}

	return nil
}
