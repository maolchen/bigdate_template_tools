package utils

import (
	"fmt"
	"strings"
)

// getNestedPort 从嵌套的 map 中获取端口值
// 支持格式: "zookeeper.client_port" 或 "server.port"
func GetNestedPort(vars map[string]interface{}, fieldPath string) int {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 1 {
		// 直接字段
		if p, ok := vars[parts[0]].(int); ok {
			return p
		}
		return 0
	}

	// 嵌套字段
	current := vars
	for i, part := range parts {
		if i == len(parts)-1 {
			// 最后一层，获取端口值
			if p, ok := current[part].(int); ok {
				return p
			}
			return 0
		}
		// 中间层，继续深入
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return 0
		}
	}
	return 0
}

// FormatID 根据格式模板生成ID
// 支持格式：
//   - "index+1": 索引+1（如 1, 2, 3）
//   - "index": 原始索引（如 0, 1, 2）
//   - "nn{index+1}": 格式化模板（如 nn1, nn2）
//   - "broker-{index+1}": 带前缀（如 broker-1, broker-2）
func FormatID(format string, index int) interface{} {
	// 计算索引偏移后的值
	indexPlusOne := index + 1

	// 替换占位符
	result := strings.ReplaceAll(format, "{index}", fmt.Sprintf("%d", index))
	result = strings.ReplaceAll(result, "{index+1}", fmt.Sprintf("%d", indexPlusOne))

	// 特殊格式：纯数值
	if result == "index" {
		return index
	}
	if result == "index+1" {
		return indexPlusOne
	}

	// 格式化模板
	return result
}
