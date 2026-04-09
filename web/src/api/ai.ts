import type { ServiceConfigItem, ServiceTopoItem } from './config';

const API_BASE = '/api';

export interface AISettings {
  baseUrl: string;
  model: string;
  hasApiKey: boolean;
  maskedApiKey: string;
  updatedAt: string;
}

export interface AIProviderModel {
  id: string;
  ownedBy?: string;
  created?: number;
}

export async function fetchAISessions(): Promise<AISessionSummary[]> {
  const response = await fetch(`${API_BASE}/ai/template/session`);
  if (!response.ok) throw new Error(await response.text() || '获取历史会话失败');
  const data = await response.json();
  return Array.isArray(data) ? data : [];
}

export async function updateAISessionMeta(sessionId: string, payload: {
  title: string;
}): Promise<AITemplateSession> {
  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/meta`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '更新会话名称失败');
  return normalizeSession(await response.json());
}

export async function deleteAISessionDrafts(sessionId: string, payload: {
  paths: string[];
  removeFromDisk?: boolean;
}): Promise<{
  deletedDrafts: string[];
  deletedTemplates: string[];
  session: AITemplateSession;
}> {
  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/drafts`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '删除模板失败');
  const data = await response.json();
  return {
    deletedDrafts: Array.isArray(data?.deletedDrafts) ? data.deletedDrafts : [],
    deletedTemplates: Array.isArray(data?.deletedTemplates) ? data.deletedTemplates : [],
    session: normalizeSession(data.session),
  };
}

async function deleteAISessionLegacy(sessionId: string): Promise<{ deletedSessionId: string }> {
  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error(await response.text() || '删除历史会话失败');
  return response.json();
}

export async function deleteAISession(sessionId: string): Promise<{ deletedSessionId: string }> {
  try {
    return await deleteAISessionLegacy(sessionId);
  } catch {
    // fallback below
  }
  const fallback = await fetch(`${API_BASE}/ai/template/session?id=${encodeURIComponent(sessionId)}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: sessionId }),
  });
  if (!fallback.ok) throw new Error(await fallback.text() || '鍒犻櫎鍘嗗彶浼氳瘽澶辫触');
  return fallback.json();
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

export interface AISkillRef {
  id: string;
  scope: string;
  title: string;
  tags: string[];
  required: boolean;
}

export interface AIExampleRef {
  path: string;
  service: string;
  kind: string;
  tags: string[];
  reason: string;
}

export interface AIPromptTrace {
  skillRefs: AISkillRef[];
  exampleRefs: AIExampleRef[];
  mode: string;
  provider: string;
}

export interface AIPromptPreviewAttachment {
  id: string;
  name: string;
  kind: 'text' | 'image';
}

export interface AIPromptPreview {
  summary: string;
  promptTrace: AIPromptTrace;
  selectedDraftPaths: string[];
  selectedSkillIds: string[];
  attachments: AIPromptPreviewAttachment[];
  hasSessionRules: boolean;
}

export interface AISkillCatalogItem {
  id: string;
  scope: string;
  title: string;
  tags: string[];
  required: boolean;
  path: string;
  source: string;
  content: string;
}

export interface AIExampleCatalogItem {
  path: string;
  service: string;
  fileName: string;
  kind: string;
  tags: string[];
  isCluster: boolean;
  snippet: string;
}

export interface AICustomSkill {
  id: string;
  title: string;
  scope: 'core' | 'task' | 'service';
  tags: string[];
  autoAttach: boolean;
  path: string;
  content: string;
  updatedAt: string;
}

export interface AITemplateMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt: string;
  pending?: boolean;
  failed?: boolean;
  streamStatus?: string[];
  attachmentIds?: string[];
  warnings?: string[];
  followUpQuestions?: string[];
  draftFiles?: AIDraftFile[];
  plannedActions?: AIPlannedAction[];
  configPatch?: AIConfigPatch;
  configIssues?: AIConfigIssue[];
  promptTrace?: AIPromptTrace;
}

export interface AITemplateSession {
  id: string;
  title: string;
  messages: AITemplateMessage[];
  draftFiles: AIDraftFile[];
  plannedActions: AIPlannedAction[];
  configPatch: AIConfigPatch;
  configIssues: AIConfigIssue[];
  attachments: AISessionAttachment[];
  promptTrace: AIPromptTrace;
  selectedSkillIds: string[];
  selectedModel: string;
  sessionRules: string;
  createdAt: string;
  updatedAt: string;
}

export interface AISessionSummary {
  id: string;
  title: string;
  createdAt: string;
  updatedAt: string;
  messageCount: number;
  draftCount: number;
}

export interface AISessionStreamEvent {
  type: 'accepted' | 'status' | 'trace' | 'delta' | 'complete' | 'error';
  message?: string;
  delta?: string;
  draftFiles?: AIDraftFile[];
  promptTrace?: AIPromptTrace;
  session?: AITemplateSession;
}

function normalizePromptTrace(trace?: AIPromptTrace | null): AIPromptTrace {
  return {
    skillRefs: Array.isArray(trace?.skillRefs) ? trace?.skillRefs : [],
    exampleRefs: Array.isArray(trace?.exampleRefs) ? trace?.exampleRefs : [],
    mode: trace?.mode || 'layered-skills',
    provider: trace?.provider || 'chat-completions',
  };
}

async function fetchWithTimeout(input: RequestInfo | URL, init?: RequestInit, timeoutMs = 180000): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = window.setTimeout(() => controller.abort(), timeoutMs);

  try {
    return await fetch(input, {
      ...init,
      signal: controller.signal,
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new Error('AI 响应超时，请稍后重试，或减少本次生成内容');
    }
    throw error;
  } finally {
    window.clearTimeout(timeoutId);
  }
}

function normalizePromptPreview(preview: AIPromptPreview): AIPromptPreview {
  return {
    ...preview,
    promptTrace: normalizePromptTrace(preview?.promptTrace),
    selectedDraftPaths: Array.isArray(preview?.selectedDraftPaths) ? preview.selectedDraftPaths : [],
    selectedSkillIds: Array.isArray(preview?.selectedSkillIds) ? preview.selectedSkillIds : [],
    attachments: Array.isArray(preview?.attachments) ? preview.attachments : [],
    summary: preview?.summary || '',
    hasSessionRules: Boolean(preview?.hasSessionRules),
  };
}

function normalizeSession(session: AITemplateSession): AITemplateSession {
  return {
    ...session,
    title: session.title || '新会话',
    messages: Array.isArray(session.messages)
      ? session.messages.map((message) => ({
        ...message,
        streamStatus: Array.isArray(message.streamStatus) ? message.streamStatus : [],
        promptTrace: normalizePromptTrace(message.promptTrace),
      }))
      : [],
    draftFiles: Array.isArray(session.draftFiles) ? session.draftFiles : [],
    plannedActions: Array.isArray(session.plannedActions) ? session.plannedActions : [],
    attachments: Array.isArray(session.attachments) ? session.attachments : [],
    configIssues: Array.isArray(session.configIssues) ? session.configIssues : [],
    promptTrace: normalizePromptTrace(session.promptTrace),
    selectedSkillIds: Array.isArray(session.selectedSkillIds) ? session.selectedSkillIds : [],
    selectedModel: session.selectedModel || '',
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

export async function fetchAIModels(): Promise<AIProviderModel[]> {
  const response = await fetch(`${API_BASE}/ai/models`);
  if (!response.ok) throw new Error(await response.text() || '获取模型列表失败');
  const data = await response.json();
  return Array.isArray(data) ? data : [];
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

export async function fetchAIPromptCatalog(): Promise<{
  skills: AISkillCatalogItem[];
  examples: AIExampleCatalogItem[];
}> {
  const response = await fetch(`${API_BASE}/ai/catalog`);
  if (!response.ok) throw new Error(await response.text() || '获取 AI 规则目录失败');
  return response.json();
}

export async function saveAISkillFile(payload: {
  path: string;
  content: string;
}): Promise<AISkillCatalogItem> {
  const response = await fetch(`${API_BASE}/ai/skill-file`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '保存 skill 文件失败');
  return response.json();
}

export async function fetchCustomAISkills(): Promise<AICustomSkill[]> {
  const response = await fetch(`${API_BASE}/ai/skills`);
  if (!response.ok) throw new Error(await response.text() || '获取自定义 skill 失败');
  return response.json();
}

export async function createCustomAISkill(payload: {
  id: string;
  title: string;
  scope: 'core' | 'task' | 'service';
  tags: string[];
  autoAttach: boolean;
  content: string;
}): Promise<AICustomSkill> {
  const response = await fetch(`${API_BASE}/ai/skills`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '创建自定义 skill 失败');
  return response.json();
}

export async function updateCustomAISkill(skillId: string, payload: {
  title: string;
  scope: 'core' | 'task' | 'service';
  tags: string[];
  autoAttach: boolean;
  content: string;
}): Promise<AICustomSkill> {
  const response = await fetch(`${API_BASE}/ai/skills/${encodeURIComponent(skillId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '更新自定义 skill 失败');
  return response.json();
}

export async function deleteCustomAISkill(skillId: string): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/ai/skills/${encodeURIComponent(skillId)}`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error(await response.text() || '删除自定义 skill 失败');
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
  selectedSkillIds?: string[];
  model?: string;
}): Promise<AITemplateSession> {
  const response = await fetchWithTimeout(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/message`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  }, 180000);
  if (!response.ok) throw new Error(await response.text() || '发送消息失败');
  return normalizeSession(await response.json());
}

export async function sendAISessionMessageStream(sessionId: string, payload: {
  message: string;
  sessionRules?: string;
  selectedDraftPaths?: string[];
  selectedSkillIds?: string[];
  model?: string;
}, handlers: {
  onEvent?: (event: AISessionStreamEvent) => void;
  onAccepted?: (event: AISessionStreamEvent) => void;
  onStatus?: (event: AISessionStreamEvent) => void;
  onTrace?: (event: AISessionStreamEvent) => void;
  onDelta?: (event: AISessionStreamEvent) => void;
  onComplete?: (session: AITemplateSession, event: AISessionStreamEvent) => void;
} = {}): Promise<AITemplateSession> {
  const response = await fetchWithTimeout(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/message?stream=1`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    },
    body: JSON.stringify(payload),
  }, 300000);
  if (!response.ok) throw new Error(await response.text() || '发送消息失败');
  if (!response.body) throw new Error('浏览器不支持流式响应');

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let completedSession: AITemplateSession | null = null;

  const processEventBlock = (block: string) => {
    const lines = block.split('\n');
    let eventName = 'message';
    const dataLines: string[] = [];

    for (const line of lines) {
      if (line.startsWith('event:')) {
        eventName = line.slice(6).trim();
      } else if (line.startsWith('data:')) {
        dataLines.push(line.slice(5).trim());
      }
    }

    if (dataLines.length === 0) return;

    const event = JSON.parse(dataLines.join('\n')) as AISessionStreamEvent;
    event.type = (event.type || eventName) as AISessionStreamEvent['type'];
    event.promptTrace = normalizePromptTrace(event.promptTrace);
    event.draftFiles = Array.isArray(event.draftFiles) ? event.draftFiles : [];
    if (event.session) {
      event.session = normalizeSession(event.session);
    }

    handlers.onEvent?.(event);
    if (event.type === 'accepted') handlers.onAccepted?.(event);
    if (event.type === 'status') handlers.onStatus?.(event);
    if (event.type === 'trace') handlers.onTrace?.(event);
    if (event.type === 'delta') handlers.onDelta?.(event);
    if (event.type === 'error') throw new Error(event.message || '发送消息失败');
    if (event.type === 'complete' && event.session) {
      completedSession = event.session;
      handlers.onComplete?.(event.session, event);
    }
  };

  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done });

    let separatorIndex = buffer.indexOf('\n\n');
    while (separatorIndex >= 0) {
      const block = buffer.slice(0, separatorIndex).trim();
      buffer = buffer.slice(separatorIndex + 2);
      if (block) {
        processEventBlock(block);
      }
      separatorIndex = buffer.indexOf('\n\n');
    }

    if (done) {
      const tail = buffer.trim();
      if (tail) {
        processEventBlock(tail);
      }
      break;
    }
  }

  if (!completedSession) {
    throw new Error('流式消息未返回最终会话结果');
  }
  return completedSession;
}

export async function previewAISessionMessage(sessionId: string, payload: {
  message: string;
  sessionRules?: string;
  selectedDraftPaths?: string[];
  selectedSkillIds?: string[];
}): Promise<AIPromptPreview> {
  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/preview`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await response.text() || '获取发送前预览失败');
  return normalizePromptPreview(await response.json());
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

export async function deleteAISessionAttachment(sessionId: string, attachmentId: string): Promise<AITemplateSession> {
  const response = await fetch(`${API_BASE}/ai/template/session/${encodeURIComponent(sessionId)}/attachment/${encodeURIComponent(attachmentId)}`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error(await response.text() || '删除附件失败');
  const data = await response.json();
  return normalizeSession(data.session);
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
