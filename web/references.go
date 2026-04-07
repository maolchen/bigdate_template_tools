package web

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckReferencesHandler POST /api/check-references - 检查删除对象时是否被template引用
func CheckReferencesHandler(templatesDir string, checkNodeUsage func(nodeName string) (bool, []string), w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req struct {
		Type     string `json:"type"`     // "global", "node", "service", "vars"
		Key      string `json:"key"`      // 对于node和service，是名称；对于global，是字段名；对于vars，是变量名
		Service  string `json:"service"`  // 对于service和vars类型，指定服务名
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	references := []map[string]interface{}{}

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
			// 只检查该服务的template文件
			if filepath.Dir(relPath) == req.Service || filepath.Dir(relPath) == "." {
				pattern := fmt.Sprintf(`\.Instance\.Vars\.%s`, req.Key)
				isReferenced = regexp.MustCompile(pattern).MatchString(contentStr)
				if isReferenced {
					details = append(details, fmt.Sprintf("引用服务变量: .Instance.Vars.%s", req.Key))
				}
			} else {
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
			references = append(references, map[string]interface{}{
				"path":    relPath,
				"service": serviceName,
				"details": details,
			})
		}

		return nil
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"hasReferences": len(references) > 0,
		"references":    references,
	})
}
