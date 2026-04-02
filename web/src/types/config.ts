// config.yaml 数据结构定义

export interface GlobalConfig {
  user: string;
  group: string;
  version: string;
  pkg_base_dir: string;
  install_base_dir: string;
  data_base_dir: string;
  software_dir: string;
  log_base_dir: string;
  java_home: string;
}

export interface NodeInfo {
  ip: string;
  hostname: string;
  [key: string]: any;
}

export interface ServiceTopoItem {
  nodes: string[];
  description?: string;
  vars?: Record<string, any>;
  [key: string]: any;
}

export interface ServiceConfigItem {
  type?: 'global';
  description?: string;
  vars?: Record<string, any>;
  [key: string]: any;
}

export interface AppConfig {
  global: GlobalConfig;
  nodes: Record<string, NodeInfo>;
  serviceTop: Record<string, ServiceTopoItem>;
  serverConfig: Record<string, ServiceConfigItem>;
}

// 表单状态类型
export interface NodeFormData {
  name: string;
  ip: string;
  hostname: string;
}

export interface ServiceTopoFormData {
  name: string;
  nodes: string[];
  description: string;
  vars: string; // JSON string
}

export interface ServiceConfigFormData {
  name: string;
  type: '' | 'global';
  description: string;
  vars: string; // JSON string
}

// 编辑器标签页类型
export type EditorTab = 'overview' | 'global' | 'nodes' | 'services' | 'preview' | 'export';
