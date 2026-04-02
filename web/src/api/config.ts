// API 基础路径
const API_BASE = '/api';

// 获取配置
export async function fetchConfig(): Promise<AppConfig> {
  const response = await fetch(`${API_BASE}/config`);
  if (!response.ok) throw new Error('获取配置失败');
  return response.json();
}

// 保存配置
export async function saveConfig(config: AppConfig): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/config`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  if (!response.ok) throw new Error('保存配置失败');
  return response.json();
}

// 生成配置
export async function generateConfig(): Promise<GenerateResult> {
  const response = await fetch(`${API_BASE}/generate`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error('生成配置失败');
  return response.json();
}

// 获取生成的文件列表
export async function fetchOutput(): Promise<OutputResult> {
  const response = await fetch(`${API_BASE}/output`);
  if (!response.ok) throw new Error('获取输出文件失败');
  return response.json();
}

// 获取单个文件内容
export async function fetchOutputFile(path: string): Promise<{ path: string; content: string }> {
  const response = await fetch(`${API_BASE}/output/file?path=${encodeURIComponent(path)}`);
  if (!response.ok) throw new Error('获取文件内容失败');
  return response.json();
}

// 下载生成的配置包
export function downloadOutput(): void {
  window.open(`${API_BASE}/output/download`, '_blank');
}

// 重新加载配置文件
export async function reloadConfig(): Promise<{ success: boolean; message: string; config: AppConfig }> {
  const response = await fetch(`${API_BASE}/config/reload`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error('重新加载配置失败');
  return response.json();
}

// 获取模板列表
export async function fetchTemplates(): Promise<TemplatesResult> {
  const response = await fetch(`${API_BASE}/templates`);
  if (!response.ok) throw new Error('获取模板列表失败');
  return response.json();
}

// 类型定义
export interface AppConfig {
  global: GlobalConfig;
  nodes: Record<string, NodeInfo>;
  serviceTop: Record<string, ServiceTopoItem>;
  serverConfig: Record<string, ServiceConfigItem>;
}

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
}

export interface ServiceTopoItem {
  nodes: string[];
  description?: string;
  vars?: Record<string, any>;
}

export interface ServiceConfigItem {
  type?: 'global';
  description?: string;
  vars?: Record<string, any>;
}

export interface GenerateResult {
  success: boolean;
  message: string;
  results: Record<string, string[]>;
  errors: string[];
  stats: {
    nodes: number;
    services: number;
    generated: number;
  };
}

export interface OutputResult {
  files: Record<string, string[]>;
  total: number;
}

export interface TemplatesResult {
  templates: Array<{
    path: string;
    service: string;
    name: string;
  }>;
}
