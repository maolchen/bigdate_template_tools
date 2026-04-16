const API_BASE = '/api';

function groupOutputFiles(files: string[]): OutputResult {
  const grouped: Record<string, string[]> = {};

  files.forEach((file) => {
    const normalized = file.replace(/\\/g, '/');
    const [node, ...rest] = normalized.split('/');
    if (!node) {
      return;
    }

    const nestedPath = rest.join('/');
    if (!grouped[node]) {
      grouped[node] = [];
    }
    grouped[node].push(nestedPath || node);
  });

  Object.values(grouped).forEach((entries) => entries.sort());

  return {
    files: grouped,
    total: files.length,
  };
}

export async function fetchConfig(): Promise<AppConfig> {
  const response = await fetch(`${API_BASE}/config`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取配置失败');
  return response.json();
}

export async function saveConfig(config: AppConfig): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/config`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  if (!response.ok) throw new Error(await response.text() || '保存配置失败');
  return response.json();
}

export async function generateConfig(): Promise<GenerateResult> {
  const response = await fetch(`${API_BASE}/generate`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '生成配置失败');
  const data = await response.json();

  return {
    success: Boolean(data.success),
    message: data.message ?? '配置生成完成',
    results: data.results ?? {},
    errors: Array.isArray(data.errors) ? data.errors : [],
    warnings: Array.isArray(data.warnings) ? data.warnings : [],
    skippedServices: Array.isArray(data.skippedServices) ? data.skippedServices : [],
    stats: {
      nodes: data.stats?.nodes ?? 0,
      services: data.stats?.services ?? 0,
      generated: data.stats?.generated ?? data.files ?? 0,
      skipped: data.stats?.skipped ?? 0,
    },
  };
}

export async function fetchOutput(): Promise<OutputResult> {
  const response = await fetch(`${API_BASE}/output`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取输出文件失败');
  const data = await response.json();

  if (Array.isArray(data)) {
    return groupOutputFiles(data);
  }

  return {
    files: data.files ?? {},
    total: data.total ?? 0,
  };
}

export async function fetchOutputFile(path: string): Promise<{ path: string; content: string }> {
  const response = await fetch(`${API_BASE}/output/file?path=${encodeURIComponent(path)}`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取文件内容失败');

  const contentType = response.headers.get('content-type') ?? '';
  if (contentType.includes('application/json')) {
    return response.json();
  }

  const content = await response.text();
  return { path, content };
}

export function downloadOutput(): void {
  window.open(`${API_BASE}/output/download`, '_blank');
}

export async function reloadConfig(): Promise<{ success: boolean; message: string; config: AppConfig }> {
  const response = await fetch(`${API_BASE}/config/reload`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '重新加载配置失败');
  return response.json();
}

export interface RootConfigVersion {
  path: string;
  mtime: string;
  mtimeUnixNano: number;
  size: number;
  sha256: string;
}

export interface TemplateEditorTreeNode {
  name: string;
  path: string;
  type: 'dir' | 'file';
  children?: TemplateEditorTreeNode[];
}

export interface TemplateEditorFile {
  path: string;
  content: string;
  readonly: boolean;
  updatedAt: string;
}

export async function fetchRootConfigVersion(): Promise<RootConfigVersion> {
  const response = await fetch(`${API_BASE}/config/version`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取主配置版本失败');
  return response.json();
}

export interface ConfigSyncSummary {
  addedGlobalKeys: string[];
  removedGlobalKeys: string[];
  globalAdded: number;
  globalRemoved: number;
  addedServices: string[];
  removedServices: string[];
  addedServiceTopServices: string[];
  removedServiceTopServices: string[];
  addedServerConfigServices: string[];
  removedServerConfigServices: string[];
  serviceTopAdded: number;
  serviceTopRemoved: number;
  serverConfigAdded: number;
  serverConfigRemoved: number;
}

export async function fetchSyncMainPreview(): Promise<{
  success: boolean;
  sync: ConfigSyncSummary;
}> {
  const response = await fetch(`${API_BASE}/config/sync-preview`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取主配置差异失败');
  return response.json();
}

export async function syncMainConfig(): Promise<{
  success: boolean;
  message: string;
  sync: ConfigSyncSummary;
  config: AppConfig;
  version: RootConfigVersion;
}> {
  const response = await fetch(`${API_BASE}/config/sync-main`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '同步主配置失败');
  return response.json();
}

export async function syncMainDelete(): Promise<{
  success: boolean;
  message: string;
  sync: ConfigSyncSummary;
  config: AppConfig;
  version: RootConfigVersion;
}> {
  const response = await fetch(`${API_BASE}/config/sync-main-delete`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '清理删除项失败');
  return response.json();
}

export async function fetchTemplates(): Promise<TemplatesResult> {
  const response = await fetch(`${API_BASE}/templates`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取模板列表失败');
  return response.json();
}

export async function saveDescriptions(serviceName: string, descriptions: Record<string, string>): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/descriptions/${encodeURIComponent(serviceName)}`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(descriptions),
  });
  if (!response.ok) throw new Error(await response.text() || '保存说明失败');
  return response.json();
}

export async function fetchDescriptions(serviceName: string): Promise<Record<string, string>> {
  const response = await fetch(`${API_BASE}/descriptions/${encodeURIComponent(serviceName)}`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取说明失败');
  return response.json();
}

export async function saveGlobalDescriptions(descriptions: Record<string, string>): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/descriptions/global`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(descriptions),
  });
  if (!response.ok) throw new Error(await response.text() || '保存说明失败');
  return response.json();
}

export async function fetchGlobalDescriptions(): Promise<Record<string, string>> {
  const response = await fetch(`${API_BASE}/descriptions/global`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取说明失败');
  return response.json();
}

export async function checkReferences(params: {
  type: 'global' | 'node' | 'service' | 'vars';
  key: string;
  service?: string;
}): Promise<{ hasReferences: boolean; references: Array<{ path: string; service: string; details: string[] }> }> {
  const response = await fetch(`${API_BASE}/check-references`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  });
  if (!response.ok) throw new Error(await response.text() || '检查引用失败');
  return response.json();
}

export interface ConfigTemplateEntry {
  id: string;
  fileName: string;
  kind?: 'main' | 'backup';
  remark: string;
  nodeSummary: string;
  updatedAt: string;
  isActive: boolean;
}

export async function fetchTemplateEditorTree(): Promise<{
  root: string;
  readonly: boolean;
  nodes: TemplateEditorTreeNode[];
}> {
  const response = await fetch(`${API_BASE}/templates/editor/tree`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取模板目录失败');
  return response.json();
}

export async function fetchTemplateEditorFile(path: string): Promise<TemplateEditorFile> {
  const response = await fetch(`${API_BASE}/templates/editor/file?path=${encodeURIComponent(path)}`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取模板文件失败');
  return response.json();
}

export async function saveTemplateEditorFile(path: string, content: string): Promise<TemplateEditorFile> {
  const response = await fetch(`${API_BASE}/templates/editor/file`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path, content }),
  });
  if (!response.ok) throw new Error(await response.text() || '保存模板文件失败');
  return response.json();
}

export async function createTemplateEditorItem(path: string, type: 'file' | 'dir', content = ''): Promise<{ success: boolean; path: string; type: 'file' | 'dir' }> {
  const response = await fetch(`${API_BASE}/templates/editor/item`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path, type, content }),
  });
  if (!response.ok) throw new Error(await response.text() || '创建模板项失败');
  return response.json();
}

export async function deleteTemplateEditorItem(path: string): Promise<{ success: boolean; path: string }> {
  const response = await fetch(`${API_BASE}/templates/editor/item?path=${encodeURIComponent(path)}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '删除模板项失败');
  return response.json();
}

export async function fetchConfigTemplates(): Promise<ConfigTemplateEntry[]> {
  const response = await fetch(`${API_BASE}/config-templates`, {
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '获取配置模板失败');
  return response.json();
}

export async function createConfigTemplate(id: string, remark: string): Promise<ConfigTemplateEntry[]> {
  const response = await fetch(`${API_BASE}/config-templates`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id, remark }),
  });
  if (!response.ok) throw new Error(await response.text() || '保存配置模板失败');
  const data = await response.json();
  return Array.isArray(data.templates) ? data.templates : [];
}

export async function switchConfigTemplate(id: string): Promise<{
  success: boolean;
  activeTemplateId: string;
  sync: {
    addedGlobalKeys: string[];
    globalAdded: number;
    addedServices: string[];
    serviceTopAdded: number;
    serverConfigAdded: number;
  };
  config: AppConfig;
}> {
  const response = await fetch(`${API_BASE}/config-templates/${encodeURIComponent(id)}/switch`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '切换配置模板失败');
  return response.json();
}

export async function deleteConfigTemplate(id: string): Promise<ConfigTemplateEntry[]> {
  const response = await fetch(`${API_BASE}/config-templates/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  if (!response.ok) throw new Error(await response.text() || '删除配置模板失败');
  const data = await response.json();
  return Array.isArray(data.templates) ? data.templates : [];
}

export interface AppConfig {
  global: GlobalConfig;
  nodes: Record<string, NodeInfo>;
  serviceTop: Record<string, ServiceTopoItem>;
  serverConfig: Record<string, ServiceConfigItem>;
}

export interface GlobalConfig {
  [key: string]: any;
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
  id_auto_derive?: boolean;
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
  warnings: string[];
  skippedServices: Array<{
    nodeIp: string;
    service: string;
    reason: string;
  }>;
  stats: {
    nodes: number;
    services: number;
    generated: number;
    skipped: number;
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
