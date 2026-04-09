package server

import (
	"config-generator/config"
	"config-generator/generator"
	"sort"
	"strings"
)

func projectTemplateFunctionNames() []string {
	funcMap := generator.BuildTemplateFuncMap(config.Context{}, &config.Config{
		Global:       config.Global{},
		Nodes:        config.Nodes{},
		ServiceTop:   config.ServiceTopos{},
		ServerConfig: config.ServiceConfigs{},
	})

	names := make([]string, 0, len(funcMap))
	for name := range funcMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func templateFunctionWhitelistSummary() string {
	names := projectTemplateFunctionNames()
	if len(names) == 0 {
		return "无"
	}
	return strings.Join(names, ", ")
}
