package checker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckRequest 引用检查请求
type CheckRequest struct {
	Type    string `json:"type"`    // "global", "node", "service", "vars"
	Key     string `json:"key"`     // 对于node和service，是名称；对于global和vars，是字段名
	Service string `json:"service"` // 对于service和vars类型，指定服务名
}

// Reference 引用信息
type Reference struct {
	Path    string   `json:"path"`    // 模板文件路径
	Service string   `json:"service"` // 服务名称
	Details []string `json:"details"` // 引用详情
}

// CheckResult 引用检查结果
type CheckResult struct {
	HasReferences bool        `json:"hasReferences"`
	References    []Reference `json:"references"`
}

// CheckReferences 检查对象是否被 template 引用
func CheckReferences(req CheckRequest, templatesDir string) (*CheckResult, error) {
	fmt.Printf("[CheckReferences] 请求参数: type=%s, key=%s, service=%s\n", req.Type, req.Key, req.Service)

	references := []Reference{}

	// 遍历所有template文件
	filepath.Walk(templatesDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(templatesDir, path)
		serviceName := filepath.Dir(relPath)
		if serviceName == "." {
			serviceName = strings.TrimSuffix(filepath.Base(path), ".tmpl")
		}

		contentStr := string(content)
		var isReferenced bool
		var details []string

		switch req.Type {
		case "global":
			// 检查全局配置引用: .Global.xxx
			pattern := fmt.Sprintf(`.Global\.%s`, req.Key)
			isReferenced = regexp.MustCompile(pattern).MatchString(contentStr)
			if isReferenced {
				details = append(details, fmt.Sprintf("引用全局配置: .Global.%s", req.Key))
			}

		case "node":
			// 检查节点引用: .Instance.Node.NodeName 或 serviceNodes "nodeName"
			pattern1 := fmt.Sprintf(`\.Instance\.Node\.NodeName\s*===\s*"%s"`, req.Key)
			pattern2 := fmt.Sprintf(`serviceNodes\s+"%s"`, req.Key)
			isReferenced = regexp.MustCompile(pattern1).MatchString(contentStr) ||
				regexp.MustCompile(pattern2).MatchString(contentStr)
			if isReferenced {
				details = append(details, fmt.Sprintf("引用节点: %s", req.Key))
			}

		case "service":
			// 检查服务引用: serviceNodes "serviceName", serviceEndpointsJoin "serviceName", 或者该服务有template文件夹
			pattern1 := fmt.Sprintf(`serviceNodes\s+"%s"`, req.Service)
			pattern2 := fmt.Sprintf(`serviceEndpointsJoin\s+"%s"`, req.Service)
			// 检查是否有其他template引用了该服务（通过template路径）
			isReferenced = regexp.MustCompile(pattern1).MatchString(contentStr) ||
				regexp.MustCompile(pattern2).MatchString(contentStr) ||
				(filepath.Dir(relPath) == req.Service)

			if isReferenced {
				if filepath.Dir(relPath) == req.Service {
					details = append(details, fmt.Sprintf("该服务的template目录: %s", relPath))
				} else {
					details = append(details, fmt.Sprintf("引用服务: %s", req.Service))
				}
			}
		case "vars":
			// 检查服务变量引用: .Instance.Vars.xxx
			// 如果指定了service，只检查该服务的template文件；否则检查所有template文件
			if req.Service == "" || filepath.Dir(relPath) == req.Service || filepath.Dir(relPath) == "." {
				pattern := fmt.Sprintf(`\.Instance\.Vars\.%s`, req.Key)
				isReferenced = regexp.MustCompile(pattern).MatchString(contentStr)
				if isReferenced {
					if req.Service != "" {
						details = append(details, fmt.Sprintf("引用服务变量: .Instance.Vars.%s (在服务 %s 中)", req.Key, req.Service))
					} else {
						details = append(details, fmt.Sprintf("引用服务变量: .Instance.Vars.%s (在 %s 中)", req.Key, relPath))
					}
				}
			} else if req.Service != "" {
				// 检查其他服务是否通过serviceNodes或serviceEndpointsJoin引用了该服务
				// 如果其他服务引用了该服务，而该服务的template中使用了这个变量，那么这个变量被间接引用
				pattern1 := fmt.Sprintf(`serviceNodes\s+"%s"`, req.Service)
				pattern2 := fmt.Sprintf(`serviceEndpointsJoin\s+"%s"`, req.Service)
				if regexp.MustCompile(pattern1).MatchString(contentStr) ||
					regexp.MustCompile(pattern2).MatchString(contentStr) {
					// 需要检查该服务的template文件是否使用这个变量
					serviceTemplateDir := filepath.Join(templatesDir, req.Service)
					if _, err := os.Stat(serviceTemplateDir); err == nil {
						// 该服务有template目录，标记为引用
						isReferenced = true
						details = append(details, fmt.Sprintf("通过服务引用间接使用: .Instance.Vars.%s (在%s服务的template中)", req.Key, req.Service))
					}
				}
			}
		}

		if isReferenced {
			references = append(references, Reference{
				Path:    relPath,
				Service: serviceName,
				Details: details,
			})
		}

		return nil
	})

	return &CheckResult{
		HasReferences: len(references) > 0,
		References:    references,
	}, nil
}
