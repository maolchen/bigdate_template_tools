package server

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"config-generator/config"
)

var (
	instanceVarPattern      = regexp.MustCompile(`\.Instance\.Vars\.([A-Za-z0-9_]+)`)
	instanceVarIndexPattern = regexp.MustCompile(`index\s+\.Instance\.Vars\s+"([^"]+)"`)
)

func emptyAIConfigPatch() aiConfigPatch {
	return aiConfigPatch{
		ServiceTop:   make(map[string]config.ServiceTopo),
		ServerConfig: make(map[string]config.ServiceConfig),
	}
}

func normalizeAIConfigPatch(patch aiConfigPatch) aiConfigPatch {
	normalized := emptyAIConfigPatch()

	for serviceName, topo := range patch.ServiceTop {
		serviceName = strings.TrimSpace(serviceName)
		if serviceName == "" {
			continue
		}

		seenNodes := make(map[string]struct{}, len(topo.Nodes))
		nodes := make([]string, 0, len(topo.Nodes))
		for _, node := range topo.Nodes {
			node = strings.TrimSpace(node)
			if node == "" {
				continue
			}
			if _, exists := seenNodes[node]; exists {
				continue
			}
			seenNodes[node] = struct{}{}
			nodes = append(nodes, node)
		}
		topo.Nodes = nodes
		normalized.ServiceTop[serviceName] = topo
	}

	for serviceName, serviceCfg := range patch.ServerConfig {
		serviceName = strings.TrimSpace(serviceName)
		if serviceName == "" {
			continue
		}

		cleanVars := make(map[string]interface{})
		for key, value := range serviceCfg.Vars {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			cleanVars[key] = value
		}
		serviceCfg.Vars = cleanVars
		normalized.ServerConfig[serviceName] = serviceCfg
	}

	return normalized
}

func mergeAIConfigPatches(base, incoming aiConfigPatch) aiConfigPatch {
	result := normalizeAIConfigPatch(base)
	incoming = normalizeAIConfigPatch(incoming)

	for serviceName, topo := range incoming.ServiceTop {
		result.ServiceTop[serviceName] = topo
	}

	for serviceName, update := range incoming.ServerConfig {
		current, exists := result.ServerConfig[serviceName]
		if !exists {
			result.ServerConfig[serviceName] = update
			continue
		}

		if strings.TrimSpace(update.Type) != "" {
			current.Type = update.Type
		}
		if strings.TrimSpace(update.Description) != "" {
			current.Description = update.Description
		}
		if strings.TrimSpace(update.IDField) != "" {
			current.IDField = update.IDField
		}
		if strings.TrimSpace(update.IDFormat) != "" {
			current.IDFormat = update.IDFormat
		}
		if current.Vars == nil {
			current.Vars = make(map[string]interface{})
		}
		for key, value := range update.Vars {
			current.Vars[key] = value
		}
		result.ServerConfig[serviceName] = current
	}

	return result
}

func serviceNameFromTemplatePath(path string) string {
	parts := strings.Split(filepathToSlash(path), "/")
	if len(parts) < 3 || parts[0] != "templates" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func filepathToSlash(path string) string {
	return strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
}

func extractInstanceVars(content string) []string {
	seen := make(map[string]struct{})

	for _, matches := range instanceVarPattern.FindAllStringSubmatch(content, -1) {
		if len(matches) < 2 {
			continue
		}
		seen[matches[1]] = struct{}{}
	}
	for _, matches := range instanceVarIndexPattern.FindAllStringSubmatch(content, -1) {
		if len(matches) < 2 {
			continue
		}
		seen[matches[1]] = struct{}{}
	}

	vars := make([]string, 0, len(seen))
	for key := range seen {
		vars = append(vars, key)
	}
	sort.Strings(vars)
	return vars
}

func collectDraftServicesAndVars(drafts []aiDraftFile) (map[string]struct{}, map[string][]string) {
	services := make(map[string]struct{})
	referencedVars := make(map[string][]string)

	for _, draft := range drafts {
		serviceName := serviceNameFromTemplatePath(draft.Path)
		if serviceName == "" {
			continue
		}
		services[serviceName] = struct{}{}
		referencedVars[serviceName] = mergeStringLists(referencedVars[serviceName], extractInstanceVars(draft.Content))
	}

	return services, referencedVars
}

func mergeStringLists(current, incoming []string) []string {
	seen := make(map[string]struct{}, len(current)+len(incoming))
	merged := make([]string, 0, len(current)+len(incoming))
	for _, value := range current {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		merged = append(merged, value)
	}
	for _, value := range incoming {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		merged = append(merged, value)
	}
	sort.Strings(merged)
	return merged
}

func appendConfigIssue(issues []aiConfigIssue, severity, service, field, message string) []aiConfigIssue {
	for _, issue := range issues {
		if issue.Severity == severity && issue.Service == service && issue.Field == field && issue.Message == message {
			return issues
		}
	}
	return append(issues, aiConfigIssue{
		Severity: severity,
		Service:  service,
		Field:    field,
		Message:  message,
	})
}

func mergeAIConfigIssues(base []aiConfigIssue, incoming []aiConfigIssue) []aiConfigIssue {
	merged := append([]aiConfigIssue(nil), base...)
	for _, issue := range incoming {
		merged = appendConfigIssue(merged, issue.Severity, issue.Service, issue.Field, issue.Message)
	}
	return merged
}

func (s *Server) enrichConfigPatchForDrafts(drafts []aiDraftFile, patch aiConfigPatch) (aiConfigPatch, []aiConfigIssue) {
	cfg := s.cfg
	if cfg == nil {
		cfg = &config.Config{
			Nodes:        config.Nodes{},
			ServiceTop:   config.ServiceTopos{},
			ServerConfig: config.ServiceConfigs{},
		}
	}
	normalized := mergeAIConfigPatches(emptyAIConfigPatch(), patch)
	issues := make([]aiConfigIssue, 0)
	services, referencedVars := collectDraftServicesAndVars(drafts)

	for serviceName, topo := range normalized.ServiceTop {
		if len(topo.Nodes) == 0 {
			issues = appendConfigIssue(issues, "error", serviceName, "serviceTop.nodes", "serviceTop.nodes 不能为空")
		}
		for _, nodeName := range topo.Nodes {
			if nodeName == "*" {
				continue
			}
			if _, exists := cfg.Nodes[nodeName]; !exists {
				issues = appendConfigIssue(issues, "error", serviceName, "serviceTop.nodes", fmt.Sprintf("节点 %s 未在当前 nodes 配置中定义", nodeName))
			}
		}
		if currentTopo, exists := cfg.ServiceTop[serviceName]; exists {
			if !reflect.DeepEqual(currentTopo.Nodes, topo.Nodes) || currentTopo.IDAutoDerive != topo.IDAutoDerive {
				issues = appendConfigIssue(issues, "warning", serviceName, "serviceTop", "将更新现有 serviceTop 配置，请确认节点范围和 id_auto_derive 是否符合预期")
			}
		}
	}

	for serviceName := range services {
		serviceCfg := normalized.ServerConfig[serviceName]
		if serviceCfg.Vars == nil {
			serviceCfg.Vars = make(map[string]interface{})
		}

		existingCfg, hasExistingCfg := cfg.ServerConfig[serviceName]
		if !hasExistingCfg {
			issues = appendConfigIssue(issues, "info", serviceName, "serverConfig", "将为新服务创建 serverConfig 配置骨架")
		}

		for _, varName := range referencedVars[serviceName] {
			if _, exists := serviceCfg.Vars[varName]; exists {
				continue
			}
			if existingCfg.Vars != nil {
				if existingValue, exists := existingCfg.Vars[varName]; exists {
					serviceCfg.Vars[varName] = existingValue
					continue
				}
			}
			serviceCfg.Vars[varName] = ""
			issues = appendConfigIssue(issues, "warning", serviceName, "vars."+varName, fmt.Sprintf("模板引用了变量 %s，已自动补充为空占位值，请在配置页补齐", varName))
		}

		if hasExistingCfg {
			if strings.TrimSpace(serviceCfg.Type) == "" {
				serviceCfg.Type = existingCfg.Type
			}
			if strings.TrimSpace(serviceCfg.Description) == "" {
				serviceCfg.Description = existingCfg.Description
			}
			if strings.TrimSpace(serviceCfg.IDField) == "" {
				serviceCfg.IDField = existingCfg.IDField
			}
			if strings.TrimSpace(serviceCfg.IDFormat) == "" {
				serviceCfg.IDFormat = existingCfg.IDFormat
			}
			for key, value := range serviceCfg.Vars {
				if existingCfg.Vars == nil {
					break
				}
				if existingValue, exists := existingCfg.Vars[key]; exists && fmt.Sprint(existingValue) != fmt.Sprint(value) {
					issues = appendConfigIssue(issues, "warning", serviceName, "vars."+key, fmt.Sprintf("变量 %s 将覆盖现有配置值", key))
				}
			}
		}

		normalized.ServerConfig[serviceName] = serviceCfg

		if _, exists := normalized.ServiceTop[serviceName]; !exists {
			if _, exists = cfg.ServiceTop[serviceName]; !exists {
				issues = appendConfigIssue(issues, "error", serviceName, "serviceTop", "新服务缺少 serviceTop 草案，请先让 AI 规划节点拓扑")
			}
		}
	}

	return normalized, issues
}

func selectServicesFromDrafts(drafts []aiDraftFile) map[string]struct{} {
	services, _ := collectDraftServicesAndVars(drafts)
	return services
}

func selectConfigPatchServices(patch aiConfigPatch, services map[string]struct{}) aiConfigPatch {
	if len(services) == 0 {
		return normalizeAIConfigPatch(patch)
	}

	selected := emptyAIConfigPatch()
	for serviceName, topo := range patch.ServiceTop {
		if _, exists := services[serviceName]; exists {
			selected.ServiceTop[serviceName] = topo
		}
	}
	for serviceName, serviceCfg := range patch.ServerConfig {
		if _, exists := services[serviceName]; exists {
			selected.ServerConfig[serviceName] = serviceCfg
		}
	}
	return selected
}

func selectConfigIssues(issues []aiConfigIssue, services map[string]struct{}) []aiConfigIssue {
	if len(services) == 0 {
		return append([]aiConfigIssue(nil), issues...)
	}
	filtered := make([]aiConfigIssue, 0, len(issues))
	for _, issue := range issues {
		if _, exists := services[issue.Service]; exists {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func hasBlockingConfigIssues(issues []aiConfigIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

func applyConfigPatchToConfig(cfg *config.Config, patch aiConfigPatch) []string {
	appliedServices := make([]string, 0)
	if cfg.ServiceTop == nil {
		cfg.ServiceTop = make(config.ServiceTopos)
	}
	if cfg.ServerConfig == nil {
		cfg.ServerConfig = make(config.ServiceConfigs)
	}

	serviceNames := make(map[string]struct{})
	for serviceName := range patch.ServiceTop {
		serviceNames[serviceName] = struct{}{}
	}
	for serviceName := range patch.ServerConfig {
		serviceNames[serviceName] = struct{}{}
	}

	for serviceName := range serviceNames {
		if topo, exists := patch.ServiceTop[serviceName]; exists {
			cfg.ServiceTop[serviceName] = topo
		}

		if serviceCfg, exists := patch.ServerConfig[serviceName]; exists {
			current := cfg.ServerConfig[serviceName]
			if strings.TrimSpace(serviceCfg.Type) != "" {
				current.Type = serviceCfg.Type
			}
			if strings.TrimSpace(serviceCfg.Description) != "" {
				current.Description = serviceCfg.Description
			}
			if strings.TrimSpace(serviceCfg.IDField) != "" {
				current.IDField = serviceCfg.IDField
			}
			if strings.TrimSpace(serviceCfg.IDFormat) != "" {
				current.IDFormat = serviceCfg.IDFormat
			}
			if current.Vars == nil {
				current.Vars = make(map[string]interface{})
			}
			for key, value := range serviceCfg.Vars {
				current.Vars[key] = value
			}
			cfg.ServerConfig[serviceName] = current
		}

		appliedServices = append(appliedServices, serviceName)
	}

	sort.Strings(appliedServices)
	return appliedServices
}

func removeAppliedServicesFromSession(session *aiSession, services map[string]struct{}) {
	for serviceName := range services {
		delete(session.ConfigPatch.ServiceTop, serviceName)
		delete(session.ConfigPatch.ServerConfig, serviceName)
	}

	filteredIssues := make([]aiConfigIssue, 0, len(session.ConfigIssues))
	for _, issue := range session.ConfigIssues {
		if _, exists := services[issue.Service]; exists {
			continue
		}
		filteredIssues = append(filteredIssues, issue)
	}
	session.ConfigIssues = filteredIssues
}

func buildConfigPromptContext(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("当前已配置节点别名（只能从这些节点中选择，不允许新增节点）:\n")
	nodeNames := make([]string, 0, len(cfg.Nodes))
	for nodeName := range cfg.Nodes {
		nodeNames = append(nodeNames, nodeName)
	}
	sort.Strings(nodeNames)
	if len(nodeNames) == 0 {
		builder.WriteString("- 当前没有可用节点配置\n")
	} else {
		for _, nodeName := range nodeNames {
			node := cfg.Nodes[nodeName]
			builder.WriteString(fmt.Sprintf("- %s (%s / %s)\n", nodeName, node.IP, node.Hostname))
		}
	}

	serviceNames := make([]string, 0, len(cfg.ServiceTop))
	for serviceName := range cfg.ServiceTop {
		serviceNames = append(serviceNames, serviceName)
	}
	sort.Strings(serviceNames)
	if len(serviceNames) > 0 {
		builder.WriteString("\n当前已有服务拓扑:\n")
		for _, serviceName := range serviceNames {
			topo := cfg.ServiceTop[serviceName]
			builder.WriteString(fmt.Sprintf("- %s nodes=%v id_auto_derive=%t\n", serviceName, topo.Nodes, topo.IDAutoDerive))
		}
	}

	return strings.TrimSpace(builder.String())
}
