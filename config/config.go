package config

// ============================================================
// 配置结构定义（完全通用、零硬编码）
// ============================================================

// Global 全局配置
type Global map[string]interface{}

// Node 节点信息
type Node struct {
	IP       string `yaml:"ip" json:"ip"`
	Hostname string `yaml:"hostname" json:"hostname"`
}

// Nodes 节点池（支持一个IP对应多个节点别名）
type Nodes map[string]Node

// ServiceTopo 服务拓扑配置
type ServiceTopo struct {
	Nodes        []string `yaml:"nodes" json:"nodes"`                 // 节点别名列表，支持 ["*"]
	IDAutoDerive bool     `yaml:"id_auto_derive" json:"id_auto_derive"` // 是否自动推导ID
}

// ServiceTopos 服务拓扑列表
type ServiceTopos map[string]ServiceTopo

// ServiceConfig 服务配置（完全配置化）
type ServiceConfig struct {
	Type        string                 `yaml:"type" json:"type"`               // 服务类型：global 或 空
	Description string                 `yaml:"description" json:"description"` // 服务描述
	IDField     string                 `yaml:"id_field" json:"id_field"`       // ID字段名
	IDFormat    string                 `yaml:"id_format" json:"id_format"`     // ID格式化模板
	Vars        map[string]interface{} `yaml:"vars" json:"vars"`               // 服务变量
}

// ServiceConfigs 服务配置列表
type ServiceConfigs map[string]ServiceConfig

// NodeOverrides 节点特化配置
type NodeOverrides map[string]map[string]map[string]interface{}

// Config 完整配置
type Config struct {
	Global        Global         `yaml:"global" json:"global"`
	Nodes         Nodes          `yaml:"nodes" json:"nodes"`
	ServiceTop    ServiceTopos   `yaml:"serviceTop" json:"serviceTop"`
	ServerConfig  ServiceConfigs `yaml:"serverConfig" json:"serverConfig"`
	NodeOverrides NodeOverrides  `yaml:"nodeOverrides" json:"nodeOverrides"`
}

// ============================================================
// 运行时数据结构
// ============================================================

// NodeInstance 节点实例（运行时）
type NodeInstance struct {
	IP       string
	Hostname string
	NodeName string // 节点别名
}

// ServiceInstance 服务实例（运行时）
type ServiceInstance struct {
	ServiceName string
	NodeName    string
	Node        NodeInstance
	Vars        map[string]interface{} // 合并后的变量
	AutoID      interface{}            // 自动推导的ID
}

// Context 模板上下文（零硬编码，不包含任何特定服务字段）
type Context struct {
	Global       Global
	Nodes        Nodes
	Instance     ServiceInstance
	AllInstances []ServiceInstance // 所有服务实例
}
