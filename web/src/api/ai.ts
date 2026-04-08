import type { ServiceConfigItem, ServiceTopoItem } from './config';

const API_BASE = '/api';

export interface AISettings {
  baseUrl: string;
  model: string;
  hasApiKey: boolean;
  maskedApiKey: string;
  updatedAt: string;
}

export interface AISessionAttachment {
  id: string;
  name: string;
  kind: 'text' | 'image';
  mimeType: string;
  size: number;
  previewText?: string;
  downloadUrl: string;
  pending: boolean;
  createdAt: string;
}

export interface AIDraftFile {
  path: string;
  content: string;
  reason: string;
  source?: string;
  needsReview: boolean;
}

export interface AIPlannedAction {
  type: 'mkdir' | 'write_file';
  path: string;
  reason: string;
}

export interface AIConfigPatch {
  serviceTop: Record<string, ServiceTopoItem>;
  serverConfig: Record<string, ServiceConfigItem>;
}

export interface AIConfigIssue {
  severity: 'info' | 'warning' | 'error';
  service: string;
  field: string;
  message: string;
}

export interface AITemplateMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt: string;
  attachmentIds?: string[];
  warnings?: string[];
  followUpQuestions?: string[];
  draftFiles?: AIDraftFile[];
  plannedActions?: AIPlannedAction[];
  configPatch?: AIConfigPatch;
  configIssues?: AIConfigIssue[];
}

export interface AITemplateSession {
  id: string;
  messages: AITemplateMessage[];
  draftFiles: AIDraftFile[];
  plannedActions: AIPlannedAction[];
  configPatch: AIConfigPatch;
  configIssues: AIConfigIssue[];
  attachments: AISessionAttachment[];
  sessionRules: string;
  createdAt: string;
  updatedAt: string;
}

function normalizeSession(session: AITemplateSession): AITemplateSession {
  return {
    ...session,
    messages: Array.isArray(session.messages) ? session.messages : [],
    draftFiles: Array.isArray(session.draftFiles) ? session.draftFiles : [],
    plannedActions: Array.isArray(session.plannedActions) ? session.plannedActions : [],
    attachments: Array.isArray(session.attachments) ? session.attachments : [],
    configIssues: Array.isArray(session.configIssues) ? session.configIssues : [],
    configPatch: {
      serviceTop: session.configPatch?.serviceTop ?? {},
      serverConfig: session.configPatch?.serverConfig ?? {},
    },
  };
}

export async function fetchAISettings(): Promise<AISettings> {
  const response = await fetch(`${API_BASE}/ai/settings`);
  if (!response.ok) throw new Error(await response.text() || '获取 AI 设置失败');
  return response.json();
}

export async function saveAISettings(payload: {
  baseUrl: string;
  model: string;
  apiKey?: string;
  clearApiKey?: boolean;
}): Promise<AISettings> {
  const response = await fetch(`${API_BASE}/ai/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '保存 AI 设置失败');
  return response.json();
}

export async function testAISettings(payload?: {
  baseUrl?: string;
  model?: string;
  apiKey?: string;
  clearApiKey?: boolean;
}): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/ai/settings/test`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload ?? {}),
  });
  if (!response.ok) throw new Error(await response.text() || '测试连接失败');
  return response.json();
}

export async function fetchAIRules(): Promise<{ content: string }> {
  const response = await fetch(`${API_BASE}/ai/rules`);
  if (!response.ok) throw new Error(await response.text() || '获取模板规则失败');
  return response.json();
}

export async function saveAIRules(content: string): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/ai/rules`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  });
  if (!response.ok) throw new Error(await response.text() || '保存模板规则失败');
  return response.json();
}

export async function createAISession(): Promise<AITemplateSession> {
  const response = await fetch(`${API_BASE}/ai/template/session`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(await response.text() || '创建会话失败');
  return normalizeSession(await response.json());
}

export async function fetchAISession(sessionId: string): Promise<AITemplateSession> {
  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}`);
  if (!response.ok) throw new Error(await response.text() || '获取会话失败');
  return normalizeSession(await response.json());
}

export async function sendAISessionMessage(sessionId: string, payload: {
  message: string;
  sessionRules?: string;
  selectedDraftPaths?: string[];
}): Promise<AITemplateSession> {
  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/message`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '发送消息失败');
  return normalizeSession(await response.json());
}

export async function uploadAISessionAttachment(sessionId: string, file: File): Promise<AISessionAttachment> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/upload`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) throw new Error(await response.text() || '上传附件失败');
  return response.json();
}

export async function saveAISessionDrafts(sessionId: string, payload: {
  files: AIDraftFile[];
  applyConfigPatch?: boolean;
}): Promise<{
  savedFiles: string[];
  createdDirs: string[];
  configApplied: boolean;
  appliedServices: string[];
  configIssues: AIConfigIssue[];
}> {
  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/save`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '保存模板失败');
  return response.json();
}
