import { useEffect, useMemo, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import {
  Bot,
  Check,
  CopyPlus,
  Cpu,
  FileText,
  FolderPlus,
  History,
  ImagePlus,
  LoaderCircle,
  PencilLine,
  RefreshCw,
  Save,
  Send,
  Settings2,
  ShieldCheck,
  Sparkles,
  Trash2,
  Upload,
  Wand2,
  X,
} from 'lucide-react';
import {
  createAISession,
  deleteAISession,
  deleteAISessionAttachment,
  deleteAISessionDrafts,
  fetchAIModels,
  fetchAIRules,
  fetchAISessions,
  fetchAISettings,
  fetchAISession,
  fetchAIPromptCatalog,
  saveAIRules,
  saveAISkillFile,
  saveAISessionDrafts,
  saveAISettings,
  sendAISessionMessageStream,
  testAISettings,
  updateAISessionMeta,
  uploadAISessionAttachment,
  type AIDraftFile,
  type AIExampleCatalogItem,
  type AIProviderModel,
  type AISettings,
  type AISessionSummary,
  type AISkillCatalogItem,
  type AIPromptTrace,
  type AITemplateMessage,
  type AITemplateSession,
} from '../api/ai';

const SESSION_STORAGE_KEY = 'config-generator:ai-template-session-id';

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'short',
    timeStyle: 'short',
  }).format(date);
}

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

function getDraftLanguage(path: string): string {
  const lower = path.toLowerCase();
  if (lower.endsWith('.sh.tmpl')) return 'shell';
  if (lower.endsWith('.md.tmpl')) return 'markdown';
  if (lower.endsWith('.yaml.tmpl') || lower.endsWith('.yml.tmpl')) return 'yaml';
  if (lower.endsWith('.json.tmpl')) return 'json';
  if (lower.endsWith('.xml.tmpl')) return 'xml';
  if (lower.endsWith('.conf.tmpl') || lower.endsWith('.properties.tmpl')) return 'ini';
  return 'plaintext';
}

function getServiceNameFromTemplatePath(path: string): string | null {
  const normalized = path.replace(/\\/g, '/').trim();
  const parts = normalized.split('/');
  if (parts.length < 3 || parts[0] !== 'templates') {
    return null;
  }
  return parts[1] || null;
}

function getMessagePatchServices(message: AITemplateMessage): string[] {
  const patch = message.configPatch;
  if (!patch) return [];
  const names = new Set([
    ...Object.keys(patch.serviceTop || {}),
    ...Object.keys(patch.serverConfig || {}),
  ]);
  return [...names].sort();
}

function getMessageAttachments(session: AITemplateSession | null, message: AITemplateMessage) {
  if (!session || !Array.isArray(message.attachmentIds) || message.attachmentIds.length === 0) {
    return [];
  }
  const attachmentsById = new Map(session.attachments.map((attachment) => [attachment.id, attachment]));
  return message.attachmentIds
    .map((attachmentId) => attachmentsById.get(attachmentId))
    .filter((attachment): attachment is NonNullable<typeof attachment> => Boolean(attachment));
}

function appendUniqueStatus(lines: string[] | undefined, next: string): string[] {
  const text = next.trim();
  if (!text) return Array.isArray(lines) ? lines : [];
  const current = Array.isArray(lines) ? lines : [];
  if (current[current.length - 1] === text) return current;
  return [...current, text];
}

function mergePromptTrace(base: AIPromptTrace | undefined, incoming: AIPromptTrace | undefined): AIPromptTrace | undefined {
  if (!incoming) return base;
  return {
    skillRefs: incoming.skillRefs || base?.skillRefs || [],
    exampleRefs: incoming.exampleRefs || base?.exampleRefs || [],
    mode: incoming.mode || base?.mode || 'layered-skills',
    provider: incoming.provider || base?.provider || 'chat-completions',
  };
}

function extractFilesFromDataTransfer(dataTransfer: DataTransfer | null): File[] {
  if (!dataTransfer) return [];
  const filesFromItems = Array.from(dataTransfer.items || [])
    .filter((item) => item.kind === 'file')
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file));
  if (filesFromItems.length > 0) {
    return filesFromItems;
  }
  return Array.from(dataTransfer.files || []);
}

interface SettingsModalProps {
  open: boolean;
  settings: AISettings | null;
  modelOptions: string[];
  modelsLoading: boolean;
  modelsError: string | null;
  saving: boolean;
  testing: boolean;
  testSucceeded: boolean;
  error: string | null;
  onClose: () => void;
  onRefreshModels: () => Promise<void>;
  onSave: (payload: {
    baseUrl: string;
    model: string;
    apiKey: string;
    clearApiKey: boolean;
  }) => Promise<void>;
  onTest: (payload: {
    baseUrl: string;
    model: string;
    apiKey: string;
    clearApiKey: boolean;
  }) => Promise<void>;
}

interface RulesModalProps {
  open: boolean;
  content: string;
  saving: boolean;
  onClose: () => void;
  onChange: (value: string) => void;
  onSave: () => Promise<void>;
}

interface SkillManagerModalProps {
  open: boolean;
  skills: AISkillCatalogItem[];
  examples: AIExampleCatalogItem[];
  loading: boolean;
  saving: boolean;
  error: string | null;
  onClose: () => void;
  onRefresh: () => Promise<void>;
  onSave: (payload: {
    path: string;
    content: string;
  }) => Promise<void>;
}

interface SessionHistoryModalProps {
  open: boolean;
  sessions: AISessionSummary[];
  activeSessionId: string | null;
  loading: boolean;
  savingSessionId: string | null;
  onClose: () => void;
  onRefresh: () => Promise<void>;
  onOpenSession: (sessionId: string) => Promise<void>;
  onRenameSession: (sessionId: string, title: string) => Promise<void>;
  onDeleteSession: (sessionId: string) => Promise<void>;
}

interface SkillDraft {
  path: string;
  title: string;
  scope: string;
  source: string;
  required: boolean;
  tagsText: string;
  content: string;
}

function createSkillDraftFromItem(skill: AISkillCatalogItem): SkillDraft {
  return {
    path: skill.path,
    title: skill.title,
    scope: skill.scope,
    source: skill.source,
    required: skill.required,
    tagsText: skill.tags.join(', '),
    content: skill.content,
  };
}

const AI_CHAT_SPLIT_MIN = 24;
const AI_CHAT_SPLIT_MAX = 78;
const AI_CHAT_SCROLL_THRESHOLD = 120;
const AI_MODEL_SUGGESTIONS = ['gpt-4.1', 'gpt-4o', 'qwen-max', 'qwen-plus'];

function SettingsModal({
  open,
  settings,
  modelOptions,
  modelsLoading,
  modelsError,
  saving,
  testing,
  testSucceeded,
  error,
  onClose,
  onRefreshModels,
  onSave,
  onTest,
}: SettingsModalProps) {
  const [baseUrl, setBaseUrl] = useState('');
  const [model, setModel] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [clearApiKey, setClearApiKey] = useState(false);

  useEffect(() => {
    if (!open || !settings) return;
    setBaseUrl(settings.baseUrl);
    setModel(settings.model);
    setApiKey('');
    setClearApiKey(false);
  }, [open, settings]);

  if (!open) return null;

  return (
    <div className="modal-overlay">
      <div className="modal modal-form-card" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3 className="font-bold text-gray-800 text-lg m-0">AI 接口设置</h3>
            <p className="text-sm text-gray-500 mt-1 m-0">配置 OpenAI 兼容接口的 Base URL、Model 和 API Key。</p>
          </div>
          <button className="modal-close" onClick={onClose} aria-label="关闭">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="modal-body">
          <div className="form-group">
            <label className="form-label">Base URL</label>
            <input className="input" value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} placeholder="https://api.openai.com/v1" />
          </div>

          <div className="form-group">
            <div className="flex items-center justify-between gap-2">
              <label className="form-label">Model</label>
              <button className="btn btn-secondary btn-sm" onClick={() => void onRefreshModels()} disabled={modelsLoading || saving || testing}>
                <RefreshCw className={`w-4 h-4 ${modelsLoading ? 'animate-spin' : ''}`} />
                刷新模型
              </button>
            </div>
            <select
              className="select"
              value={modelOptions.includes(model) ? model : ''}
              onChange={(event) => {
                if (event.target.value) {
                  setModel(event.target.value);
                }
              }}
            >
              <option value="">手动输入模型（可选）</option>
              {modelOptions.map((item) => <option key={item} value={item}>{item}</option>)}
            </select>
            <input className="input mt-2" value={model} onChange={(event) => setModel(event.target.value)} placeholder="也可以手动输入模型名，例如 qwen3.5-plus" />
            <div className="text-sm text-gray-500 mt-2">
              已加载 {modelOptions.length} 个可用模型，可直接选择或手动输入。
            </div>
            {modelsError && <div className="text-danger text-sm mt-2">{modelsError}</div>}
          </div>

          <div className="form-group">
            <label className="form-label">API Key</label>
            <input
              className="input"
              type="password"
              value={apiKey}
              onChange={(event) => setApiKey(event.target.value)}
              placeholder={settings?.hasApiKey ? `当前已配置: ${settings.maskedApiKey || '已配置'}` : '输入新的 API Key'}
            />
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <input id="clear-api-key" type="checkbox" checked={clearApiKey} onChange={(event) => setClearApiKey(event.target.checked)} />
              <label htmlFor="clear-api-key">清空已保存的 API Key</label>
            </div>
          </div>

          {settings?.updatedAt && <div className="badge badge-gray">最近更新: {formatTime(settings.updatedAt)}</div>}

          {testSucceeded && !error && (
            <div className="card mt-4" style={{ borderColor: 'var(--success)', borderWidth: '1px' }}>
              <div className="card-body">
                <div className="flex items-center gap-2 text-success">
                  <Check className="w-5 h-5" />
                  <div>
                    <div className="font-semibold">测试连接成功</div>
                    <div className="text-sm">当前输入的接口配置可用，可以直接保存。</div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {error && (
            <div className="card mt-4" style={{ borderColor: 'var(--danger)', borderWidth: '1px' }}>
              <div className="card-body">
                <div className="text-danger text-sm">{error}</div>
              </div>
            </div>
          )}
        </div>

        <div className="modal-footer modal-footer-actions">
          <button className="btn btn-secondary" onClick={onClose}>取消</button>
          <button className={`btn ${testSucceeded && !testing ? 'btn-success' : 'btn-secondary'}`} onClick={() => void onTest({ baseUrl, model, apiKey, clearApiKey })} disabled={testing || saving}>
            {testing ? <LoaderCircle className="w-4 h-4 animate-spin" /> : testSucceeded ? <Check className="w-4 h-4" /> : <ShieldCheck className="w-4 h-4" />}
            {testSucceeded && !testing ? '测试通过' : '测试连接'}
          </button>
          <button className="btn btn-primary" onClick={() => void onSave({ baseUrl, model, apiKey, clearApiKey })} disabled={saving}>
            {saving ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
            保存设置
          </button>
        </div>
      </div>
    </div>
  );
}

function RulesModal({ open, content, saving, onClose, onChange, onSave }: RulesModalProps) {
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  if (!open) return null;

  return (
    <div className="modal-overlay">
      <div className="modal-fullscreen-content modal-form-card" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3 className="font-bold text-gray-800 text-lg m-0">全局模板规范</h3>
            <p className="text-sm text-gray-500 mt-1 m-0">这些规则会自动参与每次 AI 模板生成。</p>
          </div>
          <button className="modal-close" onClick={onClose} aria-label="关闭">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="modal-body">
          <div className="flex gap-2 mb-4">
            <button className="btn btn-secondary" onClick={() => fileInputRef.current?.click()}>
              <Upload className="w-4 h-4" />
              导入文本文件
            </button>
            <span className="badge badge-gray">支持 .md .txt .yaml .conf</span>
          </div>

          <textarea className="input ai-rules-textarea" value={content} onChange={(event) => onChange(event.target.value)} placeholder="输入全局模板规范补充..." />

          <input
            ref={fileInputRef}
            type="file"
            accept=".txt,.md,.yaml,.yml,.conf,.json,.xml,.properties"
            style={{ display: 'none' }}
            onChange={async (event) => {
              const file = event.target.files?.[0];
              if (!file) return;
              onChange(await file.text());
              event.target.value = '';
            }}
          />
        </div>

        <div className="modal-footer modal-footer-actions">
          <button className="btn btn-secondary" onClick={onClose}>取消</button>
          <button className="btn btn-primary" onClick={() => void onSave()} disabled={saving}>
            {saving ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
            保存规则
          </button>
        </div>
      </div>
    </div>
  );
}

function SkillManagerModal({
  open,
  skills,
  examples,
  loading,
  saving,
  error,
  onClose,
  onRefresh,
  onSave,
}: SkillManagerModalProps) {
  const [selectedSkillPath, setSelectedSkillPath] = useState<string | null>(null);
  const [draft, setDraft] = useState<SkillDraft | null>(null);

  useEffect(() => {
    if (!open) {
      setSelectedSkillPath(null);
      setDraft(null);
      return;
    }

    if (skills.length > 0 && (!selectedSkillPath || !skills.some((skill) => skill.path === selectedSkillPath))) {
      setSelectedSkillPath(skills[0].path);
    }
  }, [open, skills, selectedSkillPath]);

  useEffect(() => {
    if (!open) return;
    const selected = skills.find((skill) => skill.path === selectedSkillPath);
    if (!selected) return;
    setDraft(createSkillDraftFromItem(selected));
  }, [open, selectedSkillPath, skills]);

  const selectedExamples = useMemo(() => {
    if (!draft) {
      return examples.slice(0, 10);
    }

    const scope = draft.scope.toLowerCase();
    const title = draft.title.toLowerCase();
    const path = draft.path.toLowerCase();
    const tags = draft.tagsText
      .split(',')
      .map((item) => item.trim().toLowerCase())
      .filter(Boolean);

    const scored = examples.map((example) => {
      let score = 0;
      const examplePath = example.path.toLowerCase();
      const exampleKind = example.kind.toLowerCase();
      const exampleService = example.service.toLowerCase();

      if (scope === 'service' && exampleService && (title.includes(exampleService) || path.includes(exampleService))) {
        score += 4;
      }
      if (scope === 'task') {
        if (path.includes('install') && exampleKind === 'shell_script') score += 3;
        if (path.includes('config') && exampleKind === 'config_file') score += 3;
        if (path.includes('systemd') && exampleKind === 'systemd_service') score += 3;
      }
      if (draft.required) {
        score += 1;
      }
      for (const tag of tags) {
        if (examplePath.includes(tag) || exampleKind.includes(tag) || exampleService.includes(tag)) {
          score += 2;
        }
      }
      if (scope === 'core') {
        score += example.isCluster ? 1 : 0;
      }
      return { example, score };
    });

    return scored
      .sort((a, b) => b.score - a.score || a.example.path.localeCompare(b.example.path))
      .slice(0, 10)
      .map((item) => item.example);
  }, [draft, examples]);

  if (!open) return null;

  return (
    <div className="modal-overlay">
      <div className="modal-fullscreen-content modal-form-card" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3 className="font-bold text-gray-800 text-lg m-0">Skill 管理</h3>
            <p className="text-sm text-gray-500 mt-1 m-0">查看并编辑 `data/ai/skills/*/*.md` 下的分层 skill，同时参考默认模板示例。</p>
          </div>
          <div className="flex items-center gap-2">
            <button className="btn btn-secondary" onClick={() => void onRefresh()} disabled={loading || saving}>
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              刷新
            </button>
            <button className="modal-close" onClick={onClose} aria-label="关闭">
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div className="modal-body">
          {error && <div className="ai-inline-alert ai-inline-alert-danger mb-4"><span>{error}</span></div>}

          <div className="ai-skill-manager-grid">
            <div className="card">
              <div className="card-body">
                <div className="flex items-center justify-between gap-2 mb-3">
                  <div>
                    <div className="font-semibold text-gray-800">技能列表</div>
                    <div className="text-sm text-gray-500">共 {skills.length} 个</div>
                  </div>
                  <span className="badge badge-gray">data/ai/skills</span>
                </div>

                <div className="ai-skill-list">
                  {skills.length === 0 ? (
                    <div className="empty-state p-4">
                      <p>还没有可管理的 skill 文件</p>
                    </div>
                  ) : (
                    skills.map((skill) => (
                      <button
                        key={skill.path}
                        type="button"
                        className={`ai-skill-list-item ${selectedSkillPath === skill.path ? 'active' : ''}`}
                        onClick={() => {
                          setSelectedSkillPath(skill.path);
                        }}
                      >
                        <div className="font-medium text-gray-800">{skill.title}</div>
                        <div className="text-xs text-gray-500 mt-1">{skill.path}</div>
                        <div className="flex items-center gap-2 flex-wrap mt-2">
                          <span className="badge badge-gray">{skill.scope}</span>
                          <span className={`badge ${skill.source === 'builtin' ? 'badge-blue' : 'badge-gray'}`}>{skill.source === 'builtin' ? '内置' : '自定义'}</span>
                          {skill.required && <span className="badge badge-danger">默认启用</span>}
                        </div>
                      </button>
                    ))
                  )}
                </div>
              </div>
            </div>

            <div className="ai-skill-editor-stack">
              <div className="card">
                <div className="card-body">
                <div className="flex items-center justify-between gap-2 mb-4">
                  <div>
                    <div className="font-semibold text-gray-800">编辑 Skill</div>
                    <div className="text-sm text-gray-500">直接编辑 `data/ai/skills/*/*.md` 下的正文内容。</div>
                  </div>
                </div>

                {draft ? (
                  <>
                    <div className="ai-skill-summary-grid">
                      <div className="form-group">
                        <label className="form-label">路径</label>
                        <input className="input" value={draft.path} disabled />
                      </div>
                      <div className="form-group">
                        <label className="form-label">标题</label>
                        <input className="input" value={draft.title} disabled />
                      </div>
                    </div>

                    <div className="flex items-center gap-2 flex-wrap mb-4">
                      <span className="badge badge-gray">{draft.scope}</span>
                      <span className={`badge ${draft.source === 'builtin' ? 'badge-blue' : 'badge-gray'}`}>{draft.source === 'builtin' ? '内置规则' : '自定义规则'}</span>
                      {draft.required && <span className="badge badge-danger">默认引用</span>}
                      {draft.tagsText && <span className="badge badge-gray">{draft.tagsText}</span>}
                    </div>

                    <div className="form-group">
                      <label className="form-label">内容</label>
                      <textarea
                        className="input ai-rules-textarea"
                        value={draft.content}
                        onChange={(event) => setDraft((previous) => previous ? { ...previous, content: event.target.value } : previous)}
                        placeholder="输入 skill 规则正文..."
                      />
                    </div>

                    <div className="flex items-center justify-end gap-2">
                      <button className="btn btn-secondary" onClick={onClose}>
                        取消
                      </button>
                      <button
                        className="btn btn-primary"
                        onClick={() => void onSave({ path: draft.path, content: draft.content })}
                        disabled={saving}
                      >
                        {saving ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                        保存 Skill
                      </button>
                    </div>
                  </>
                ) : (
                  <div className="empty-state p-6">
                    <p>从左侧选择一个 skill 文件开始编辑。</p>
                  </div>
                )}
              </div>
            </div>

              <div className="card">
                <div className="card-body">
                  <div className="flex items-center justify-between gap-2 mb-4">
                    <div>
                      <div className="font-semibold text-gray-800">模板示例</div>
                      <div className="text-sm text-gray-500">展示和当前 skill 更相关的模板示例，方便对照编写规则。</div>
                    </div>
                    <span className="badge badge-gray">{selectedExamples.length} 个示例</span>
                  </div>

                  <div className="ai-skill-example-list">
                    {selectedExamples.map((example) => (
                      <div key={example.path} className="ai-skill-example-item">
                        <div className="flex items-center justify-between gap-2 flex-wrap">
                          <div className="font-medium text-gray-800">{example.path}</div>
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="badge badge-gray">{example.kind}</span>
                            <span className="badge badge-blue">{example.service || 'common'}</span>
                            {example.isCluster && <span className="badge badge-danger">cluster</span>}
                          </div>
                        </div>
                        <div className="text-sm text-gray-500 mt-2">{example.fileName}</div>
                        {example.tags.length > 0 && <div className="text-xs text-gray-500 mt-1">tags: {example.tags.join(', ')}</div>}
                        <pre className="ai-skill-example-snippet">{example.snippet}</pre>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function SessionHistoryModal({
  open,
  sessions,
  activeSessionId,
  loading,
  savingSessionId,
  onClose,
  onRefresh,
  onOpenSession,
  onRenameSession,
  onDeleteSession,
}: SessionHistoryModalProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftTitle, setDraftTitle] = useState('');
  const [keyword, setKeyword] = useState('');

  useEffect(() => {
    if (!open) {
      setEditingId(null);
      setDraftTitle('');
      setKeyword('');
    }
  }, [open]);

  const filteredSessions = useMemo(() => {
    const q = keyword.trim().toLowerCase();
    if (!q) return sessions;
    return sessions.filter((item) => {
      const haystack = [
        item.title,
        item.id,
        item.createdAt,
        item.updatedAt,
      ].join(' ').toLowerCase();
      return haystack.includes(q);
    });
  }, [keyword, sessions]);

  if (!open) return null;

  return (
    <div className="modal-overlay">
      <div className="modal modal-form-card ai-session-history-modal" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3 className="font-bold text-gray-800 text-lg m-0">历史会话</h3>
            <p className="text-sm text-gray-500 mt-1 m-0">切换回旧会话继续修改模板，也可以直接改会话名称。</p>
          </div>
          <div className="flex items-center gap-2">
            <button className="btn btn-secondary" onClick={() => void onRefresh()} disabled={loading}>
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              刷新
            </button>
            <button className="modal-close" onClick={onClose} aria-label="关闭">
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div className="modal-body">
          <div className="form-group mb-4">
            <label className="form-label">搜索会话</label>
            <input
              className="input"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder="按会话名、会话 ID 搜索"
            />
          </div>
          <div className="ai-session-history-list">
            {filteredSessions.length === 0 ? (
              <div className="empty-state p-4">
                <p>{sessions.length === 0 ? '当前还没有历史会话' : '没有匹配的会话'}</p>
              </div>
            ) : filteredSessions.map((item) => {
              const editing = editingId === item.id;
              const saving = savingSessionId === item.id;
              return (
                <div key={item.id} className={`ai-session-history-item ${activeSessionId === item.id ? 'active' : ''}`}>
                  <div className="ai-session-history-main">
                    {editing ? (
                      <input
                        className="input"
                        value={draftTitle}
                        onChange={(event) => setDraftTitle(event.target.value)}
                        placeholder="输入会话名称"
                      />
                    ) : (
                      <div className="ai-session-history-title">{item.title || '新会话'}</div>
                    )}
                    <div className="ai-session-history-meta">
                      <span>{item.messageCount} 条消息</span>
                      <span>{item.draftCount} 个草稿</span>
                      <span>更新于 {formatTime(item.updatedAt)}</span>
                    </div>
                  </div>
                  <div className="ai-session-history-actions">
                    {editing ? (
                      <>
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={() => {
                            setEditingId(null);
                            setDraftTitle('');
                          }}
                          disabled={saving}
                        >
                          取消
                        </button>
                        <button
                          className="btn btn-primary btn-sm"
                          onClick={() => void (async () => {
                            await onRenameSession(item.id, draftTitle);
                            setEditingId(null);
                            setDraftTitle('');
                          })()}
                          disabled={saving}
                        >
                          {saving ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                          保存名称
                        </button>
                      </>
                    ) : (
                      <>
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={() => {
                            setEditingId(item.id);
                            setDraftTitle(item.title || '');
                          }}
                        >
                          <PencilLine className="w-4 h-4" />
                          改名
                        </button>
                        <button
                          className="btn btn-primary btn-sm"
                          onClick={() => void onOpenSession(item.id)}
                          disabled={activeSessionId === item.id}
                        >
                          打开会话
                        </button>
                        <button
                          className="btn btn-secondary btn-sm btn-danger-soft"
                          onClick={() => void onDeleteSession(item.id)}
                          disabled={saving}
                        >
                          <Trash2 className="w-4 h-4" />
                          删除
                        </button>
                      </>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

export function AITemplatePage() {
  const [settings, setSettings] = useState<AISettings | null>(null);
  const [rules, setRules] = useState('');
  const [session, setSession] = useState<AITemplateSession | null>(null);
  const [historySessions, setHistorySessions] = useState<AISessionSummary[]>([]);
  const [optimisticMessages, setOptimisticMessages] = useState<AITemplateMessage[]>([]);
  const [skillCatalog, setSkillCatalog] = useState<AISkillCatalogItem[]>([]);
  const [skillExamples, setSkillExamples] = useState<AIExampleCatalogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [loadingSkills, setLoadingSkills] = useState(false);
  const [pageError, setPageError] = useState<string | null>(null);
  const [skillsError, setSkillsError] = useState<string | null>(null);
  const [message, setMessage] = useState('');
  const [sessionRules, setSessionRules] = useState('');
  const [selectedModel, setSelectedModel] = useState('');
  const [selectedDraftPath, setSelectedDraftPath] = useState<string | null>(null);
  const [editorMode, setEditorMode] = useState(false);
  const [isCompactViewport, setIsCompactViewport] = useState(() => window.innerWidth <= 1180);
  const [splitRatio, setSplitRatio] = useState(() => {
    const stored = window.localStorage.getItem('config-generator:ai-split-ratio');
    const parsed = stored ? Number(stored) : 62;
    return Number.isFinite(parsed) ? Math.min(AI_CHAT_SPLIT_MAX, Math.max(AI_CHAT_SPLIT_MIN, parsed)) : 62;
  });
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [rulesOpen, setRulesOpen] = useState(false);
  const [skillsOpen, setSkillsOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [availableModels, setAvailableModels] = useState<AIProviderModel[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [modelsError, setModelsError] = useState<string | null>(null);
  const [settingsError, setSettingsError] = useState<string | null>(null);
  const [settingsTestSucceeded, setSettingsTestSucceeded] = useState(false);
  const [savingSettings, setSavingSettings] = useState(false);
  const [testingSettings, setTestingSettings] = useState(false);
  const [savingRules, setSavingRules] = useState(false);
  const [savingSkill, setSavingSkill] = useState(false);
  const [savingSessionId, setSavingSessionId] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [inflightAttachmentIds, setInflightAttachmentIds] = useState<string[]>([]);
  const [composerDragging, setComposerDragging] = useState(false);
  const [savingDrafts, setSavingDrafts] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const textUploadRef = useRef<HTMLInputElement | null>(null);
  const imageUploadRef = useRef<HTMLInputElement | null>(null);
  const messageListEndRef = useRef<HTMLDivElement | null>(null);
  const layoutRef = useRef<HTMLDivElement | null>(null);
  const resizeSessionRef = useRef<{ startX: number; startRatio: number; width: number } | null>(null);
  const shouldAutoScrollRef = useRef(true);
  const previousMessageCountRef = useRef(0);

  useEffect(() => { void bootstrap(); }, []);
  useEffect(() => { setSessionRules(session?.sessionRules || ''); }, [session?.id, session?.sessionRules]);
  useEffect(() => {
    if (!session) return;
    if (!selectedDraftPath && session.draftFiles.length > 0) {
      setSelectedDraftPath(session.draftFiles[0].path);
      return;
    }
    if (selectedDraftPath && !session.draftFiles.some((draft) => draft.path === selectedDraftPath)) {
      setSelectedDraftPath(session.draftFiles[0]?.path ?? null);
    }
  }, [session, selectedDraftPath]);
  const threadMessages = useMemo(() => [...(session?.messages ?? []), ...optimisticMessages], [session?.messages, optimisticMessages]);
  useEffect(() => {
    const updateScrollPreference = () => {
      const scrollRoot = document.scrollingElement ?? document.documentElement;
      const distanceToBottom = scrollRoot.scrollHeight - scrollRoot.scrollTop - window.innerHeight;
      shouldAutoScrollRef.current = distanceToBottom <= AI_CHAT_SCROLL_THRESHOLD;
    };

    updateScrollPreference();
    window.addEventListener('scroll', updateScrollPreference, { passive: true });
    return () => window.removeEventListener('scroll', updateScrollPreference);
  }, [session?.id]);
  useEffect(() => {
    const nextMessageCount = threadMessages.length;
    const isNewMessage = nextMessageCount > previousMessageCountRef.current;
    previousMessageCountRef.current = nextMessageCount;
    if (!shouldAutoScrollRef.current && !isNewMessage) {
      return;
    }
    window.requestAnimationFrame(() => {
      const scrollRoot = document.scrollingElement ?? document.documentElement;
      window.scrollTo({
        top: scrollRoot.scrollHeight,
        behavior: isNewMessage ? 'smooth' : 'auto',
      });
    });
  }, [threadMessages]);
  useEffect(() => {
    const handleResize = () => {
      setIsCompactViewport(window.innerWidth <= 1180);
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);
  useEffect(() => {
    window.localStorage.setItem('config-generator:ai-split-ratio', String(splitRatio));
  }, [splitRatio]);
  useEffect(() => {
    if (isCompactViewport) {
      resizeSessionRef.current = null;
    }
  }, [isCompactViewport]);
  useEffect(() => {
    const handleMouseMove = (event: MouseEvent) => {
      if (!resizeSessionRef.current) return;
      const { startX, startRatio, width } = resizeSessionRef.current;
      const deltaRatio = ((event.clientX - startX) / width) * 100;
      setSplitRatio(Math.min(AI_CHAT_SPLIT_MAX, Math.max(AI_CHAT_SPLIT_MIN, startRatio + deltaRatio)));
    };
    const handleMouseUp = () => {
      if (!resizeSessionRef.current) return;
      resizeSessionRef.current = null;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, []);

  async function bootstrap() {
    try {
      setLoading(true);
      setPageError(null);
      const [settingsData, rulesData] = await Promise.all([fetchAISettings(), fetchAIRules()]);
      setSettings(settingsData);
      setRules(rulesData.content || '');
      setSelectedModel((settingsData.model || '').trim());
      void refreshAvailableModels();
      void loadSkillCatalog();
      void loadHistorySessions();
      const storedSessionId = window.localStorage.getItem(SESSION_STORAGE_KEY);
      if (storedSessionId) {
        try {
          const currentSession = await fetchAISession(storedSessionId);
          setSession(currentSession);
          setSelectedModel((currentSession.selectedModel || settingsData.model || '').trim());
          return;
        } catch {
          window.localStorage.removeItem(SESSION_STORAGE_KEY);
        }
      }
      const nextSession = await createAISession();
      window.localStorage.setItem(SESSION_STORAGE_KEY, nextSession.id);
      setSession(nextSession);
      setSelectedModel((nextSession.selectedModel || settingsData.model || '').trim());
      shouldAutoScrollRef.current = true;
    } catch (error) {
      setPageError(error instanceof Error ? error.message : 'AI 工作台初始化失败');
    } finally {
      setLoading(false);
    }
  }

  async function refreshAvailableModels() {
    try {
      setLoadingModels(true);
      setModelsError(null);
      const models = await fetchAIModels();
      setAvailableModels(models);
    } catch (error) {
      setAvailableModels([]);
      setModelsError(error instanceof Error ? error.message : '获取模型列表失败');
    } finally {
      setLoadingModels(false);
    }
  }

  async function loadHistorySessions() {
    try {
      setLoadingHistory(true);
      const sessions = await fetchAISessions();
      setHistorySessions(sessions);
    } finally {
      setLoadingHistory(false);
    }
  }

  async function loadSkillCatalog(openAfterLoad = false) {
    try {
      setLoadingSkills(true);
      setSkillsError(null);
      const catalog = await fetchAIPromptCatalog();
      setSkillCatalog(Array.isArray(catalog.skills) ? catalog.skills : []);
      setSkillExamples(Array.isArray(catalog.examples) ? catalog.examples : []);
      if (openAfterLoad) {
        setSkillsOpen(true);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载 Skill 失败';
      setSkillsError(message);
      if (openAfterLoad) {
        setSkillsOpen(true);
      }
    } finally {
      setLoadingSkills(false);
    }
  }

  const selectedDraft = useMemo(() => session?.draftFiles.find((draft) => draft.path === selectedDraftPath) ?? null, [session, selectedDraftPath]);
  const pendingAttachments = session?.attachments.filter((attachment) => attachment.pending) ?? [];
  const visiblePendingAttachments = useMemo(() => {
    if (inflightAttachmentIds.length === 0) return pendingAttachments;
    const hidden = new Set(inflightAttachmentIds);
    return pendingAttachments.filter((attachment) => !hidden.has(attachment.id));
  }, [pendingAttachments, inflightAttachmentIds]);
  const latestAssistantMessage = [...threadMessages].reverse().find((item) => item.role === 'assistant' && !item.pending) ?? null;
  const latestAssistantDrafts = latestAssistantMessage?.draftFiles?.length ? latestAssistantMessage.draftFiles : session?.draftFiles ?? [];
  const hasAnyDraftFiles = (session?.draftFiles.length ?? 0) > 0;
  const configPatchServiceNames = useMemo(() => {
    if (!session) return [];
    const names = new Set([...Object.keys(session.configPatch?.serviceTop || {}), ...Object.keys(session.configPatch?.serverConfig || {})]);
    return [...names].sort();
  }, [session]);
  const hasAnyConfigPatch = configPatchServiceNames.length > 0;
  const selectedDraftService = selectedDraft ? getServiceNameFromTemplatePath(selectedDraft.path) : null;
  const selectedServiceHasConfigPatch = selectedDraftService
    ? Boolean(session?.configPatch?.serviceTop?.[selectedDraftService] || session?.configPatch?.serverConfig?.[selectedDraftService])
    : false;
  const modelSuggestions = useMemo(() => {
    const merged = new Set<string>();
    AI_MODEL_SUGGESTIONS.forEach((item) => merged.add(item));
    availableModels.forEach((item) => item.id && merged.add(item.id));
    if (settings?.model) merged.add(settings.model);
    if (session?.selectedModel) merged.add(session.selectedModel);
    return [...merged].filter(Boolean);
  }, [availableModels, settings?.model, session?.selectedModel]);

  async function refreshSession(currentSessionId?: string) {
    const targetSessionId = currentSessionId ?? session?.id;
    if (!targetSessionId) return;
    const currentSession = await fetchAISession(targetSessionId);
    setSession(currentSession);
    setInflightAttachmentIds([]);
    setSelectedModel((currentSession.selectedModel || settings?.model || '').trim());
    setOptimisticMessages([]);
    shouldAutoScrollRef.current = true;
  }

  async function handleCreateSession() {
    try {
      const nextSession = await createAISession();
      window.localStorage.setItem(SESSION_STORAGE_KEY, nextSession.id);
      setSession(nextSession);
      setInflightAttachmentIds([]);
      setOptimisticMessages([]);
      setSelectedDraftPath(null);
      setEditorMode(false);
      setMessage('');
      setSessionRules('');
      setSelectedModel((nextSession.selectedModel || settings?.model || '').trim());
      setHistorySessions((previous) => [ {
        id: nextSession.id,
        title: nextSession.title,
        createdAt: nextSession.createdAt,
        updatedAt: nextSession.updatedAt,
        messageCount: nextSession.messages.length,
        draftCount: nextSession.draftFiles.length,
      }, ...previous.filter((item) => item.id !== nextSession.id)]);
      shouldAutoScrollRef.current = true;
      setToast('已创建新的 AI 模板会话');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '创建会话失败');
    }
  }

  async function handleSendMessage() {
    if (!session) return;
    if (uploading) return;
    if (!message.trim() && pendingAttachments.length === 0) return;
    const outgoingMessage = message;
    const modelForMessage = selectedModel.trim() || settings?.model?.trim() || '';
    const createdAt = new Date().toISOString();
    const optimisticUserId = `local-user-${Date.now()}`;
    const optimisticAssistantId = `local-assistant-${Date.now()}`;
    const optimisticUserContent = outgoingMessage.trim() || `发送了 ${pendingAttachments.length} 个附件材料`;
    const pendingAttachmentIds = pendingAttachments.map((attachment) => attachment.id);
    try {
      setSending(true);
      setInflightAttachmentIds(pendingAttachmentIds);
      setPageError(null);
      shouldAutoScrollRef.current = true;
      setOptimisticMessages([
        {
          id: optimisticUserId,
          role: 'user',
          content: optimisticUserContent,
          createdAt,
          attachmentIds: pendingAttachmentIds,
        },
        {
          id: optimisticAssistantId,
          role: 'assistant',
          content: '',
          createdAt,
          pending: true,
          streamStatus: ['正在等待服务端开始生成'],
        },
      ]);
      setMessage('');
      const updated = await sendAISessionMessageStream(session.id, {
        message: outgoingMessage,
        sessionRules,
        selectedDraftPaths: selectedDraftPath ? [selectedDraftPath] : [],
        model: modelForMessage || undefined,
      }, {
        onAccepted: (event) => {
          setOptimisticMessages((previous) => previous.map((item) => (
            item.id === optimisticAssistantId
              ? { ...item, streamStatus: appendUniqueStatus(item.streamStatus, event.message || '消息已发送') }
              : item
          )));
        },
        onTrace: (event) => {
          setOptimisticMessages((previous) => previous.map((item) => (
            item.id === optimisticAssistantId
              ? { ...item, promptTrace: mergePromptTrace(item.promptTrace, event.promptTrace) }
              : item
          )));
        },
        onStatus: (event) => {
          setOptimisticMessages((previous) => previous.map((item) => (
            item.id === optimisticAssistantId
              ? {
                ...item,
                streamStatus: appendUniqueStatus(item.streamStatus, event.message || ''),
                promptTrace: mergePromptTrace(item.promptTrace, event.promptTrace),
              }
              : item
          )));
        },
        onDelta: (event) => {
          setOptimisticMessages((previous) => previous.map((item) => (
            item.id === optimisticAssistantId
              ? {
                ...item,
                content: `${item.content || ''}${event.delta || ''}`,
                draftFiles: event.draftFiles && event.draftFiles.length > 0 ? event.draftFiles : item.draftFiles,
              }
              : item
          )));
        },
      });
      setSession(updated);
      setSelectedModel((updated.selectedModel || modelForMessage || settings?.model || '').trim());
      setHistorySessions((previous) => [
        {
          id: updated.id,
          title: updated.title,
          createdAt: updated.createdAt,
          updatedAt: updated.updatedAt,
          messageCount: updated.messages.length,
          draftCount: updated.draftFiles.length,
        },
        ...previous.filter((item) => item.id !== updated.id),
      ]);
      setOptimisticMessages([]);
      if (updated.draftFiles.length > 0 && !selectedDraftPath) {
        setSelectedDraftPath(updated.draftFiles[0].path);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '发送消息失败';
      setPageError(errorMessage);
      setOptimisticMessages((previous) => previous.map((item) => (
        item.id === optimisticAssistantId
          ? { ...item, content: `请求失败：${errorMessage}`, pending: false, failed: true }
          : item
      )));
    } finally {
      setSending(false);
      setInflightAttachmentIds([]);
    }
  }

  async function handleUpload(files: FileList | File[] | null) {
    const uploadFiles = Array.isArray(files) ? files : Array.from(files || []);
    if (!session || uploadFiles.length === 0) return;
    try {
      setUploading(true);
      setPageError(null);
      for (const file of uploadFiles) {
        const attachment = await uploadAISessionAttachment(session.id, file);
        setSession((previous) => previous ? { ...previous, attachments: [...previous.attachments, attachment] } : previous);
      }
      setToast('附件已加入当前消息上下文');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '上传附件失败');
    } finally {
      setUploading(false);
      setComposerDragging(false);
      if (textUploadRef.current) textUploadRef.current.value = '';
      if (imageUploadRef.current) imageUploadRef.current.value = '';
    }
  }

  async function handleDeleteAttachment(attachmentId: string) {
    if (!session) return;
    try {
      setPageError(null);
      const updated = await deleteAISessionAttachment(session.id, attachmentId);
      setSession(updated);
      setInflightAttachmentIds((previous) => previous.filter((id) => id !== attachmentId));
      setToast('已取消本轮待发送附件');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '取消附件失败');
    }
  }

  function handleComposerPaste(event: React.ClipboardEvent<HTMLTextAreaElement>) {
    const pastedFiles = extractFilesFromDataTransfer(event.clipboardData);
    if (pastedFiles.length === 0) return;
    event.preventDefault();
    void handleUpload(pastedFiles);
  }

  function handleComposerDragOver(event: React.DragEvent<HTMLDivElement>) {
    const dragFiles = extractFilesFromDataTransfer(event.dataTransfer);
    if (dragFiles.length === 0) return;
    event.preventDefault();
    if (!composerDragging) {
      setComposerDragging(true);
    }
  }

  function handleComposerDragLeave(event: React.DragEvent<HTMLDivElement>) {
    if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
      setComposerDragging(false);
    }
  }

  function handleComposerDrop(event: React.DragEvent<HTMLDivElement>) {
    const droppedFiles = extractFilesFromDataTransfer(event.dataTransfer);
    if (droppedFiles.length === 0) return;
    event.preventDefault();
    setComposerDragging(false);
    void handleUpload(droppedFiles);
  }

  function updateDraftContent(targetPath: string, updater: (draft: AIDraftFile) => AIDraftFile) {
    setSession((previous) => previous ? {
      ...previous,
      draftFiles: previous.draftFiles.map((draft) => draft.path === targetPath ? updater(draft) : draft),
    } : previous);
  }

  async function handleDeleteDraft(path: string) {
    if (!session) return;
    const confirmed = window.confirm('删除后会从当前会话草稿中移除，并尝试删除 templates 下已保存的同名模板文件。确定继续吗？');
    if (!confirmed) return;

    try {
      setSavingDrafts(true);
      setPageError(null);
      const result = await deleteAISessionDrafts(session.id, {
        paths: [path],
        removeFromDisk: true,
      });
      setSession(result.session);
      setHistorySessions((previous) => previous.map((item) => item.id === result.session.id ? {
        ...item,
        title: result.session.title,
        updatedAt: result.session.updatedAt,
        messageCount: result.session.messages.length,
        draftCount: result.session.draftFiles.length,
      } : item));
      if (selectedDraftPath === path) {
        setSelectedDraftPath(result.session.draftFiles[0]?.path ?? null);
      }
      setToast(result.deletedTemplates.length > 0 ? '草稿和已保存模板已删除' : '草稿已从当前会话移除');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '删除模板失败');
    } finally {
      setSavingDrafts(false);
    }
  }

  async function handleSaveDrafts(files: AIDraftFile[], applyConfigPatch = false) {
    if (!session || files.length === 0) return;
    try {
      setSavingDrafts(true);
      setPageError(null);
      const result = await saveAISessionDrafts(session.id, { files, applyConfigPatch });
      await refreshSession(session.id);
      void loadHistorySessions();
      if (applyConfigPatch) {
        setToast(result.configApplied
          ? `已保存 ${result.savedFiles.length} 个模板文件，并同步 ${result.appliedServices.length} 个服务配置`
          : `模板已保存，但配置未同步，请先处理 ${result.configIssues.length} 个配置问题`);
      } else {
        setToast(`已保存 ${result.savedFiles.length} 个模板文件`);
      }
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '保存模板失败');
    } finally {
      setSavingDrafts(false);
    }
  }

  async function handleSaveSettings(payload: { baseUrl: string; model: string; apiKey: string; clearApiKey: boolean }) {
    try {
      setSavingSettings(true);
      setSettingsError(null);
      setSettingsTestSucceeded(false);
      const updated = await saveAISettings(payload);
      setSettings(updated);
      await refreshAvailableModels();
      setSelectedModel((current) => current.trim() || (updated.model || '').trim());
      setToast('AI 设置已保存');
      setSettingsOpen(false);
    } catch (error) {
      setSettingsError(error instanceof Error ? error.message : '保存 AI 设置失败');
    } finally {
      setSavingSettings(false);
    }
  }

  async function handleTestSettings(payload: { baseUrl: string; model: string; apiKey: string; clearApiKey: boolean }) {
    try {
      setTestingSettings(true);
      setSettingsError(null);
      setSettingsTestSucceeded(false);
      const result = await testAISettings(payload);
      await refreshAvailableModels();
      setSettingsTestSucceeded(true);
      setToast(result.message || '测试连接成功');
    } catch (error) {
      setSettingsTestSucceeded(false);
      setSettingsError(error instanceof Error ? error.message : '测试连接失败');
    } finally {
      setTestingSettings(false);
    }
  }

  async function handleSaveRules() {
    try {
      setSavingRules(true);
      setPageError(null);
      const result = await saveAIRules(rules);
      setToast(result.message);
      setRulesOpen(false);
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '保存模板规则失败');
    } finally {
      setSavingRules(false);
    }
  }

  async function handleSaveSkillFile(payload: { path: string; content: string }) {
    try {
      setSavingSkill(true);
      setSkillsError(null);
      const updated = await saveAISkillFile(payload);
      setSkillCatalog((previous) => previous.map((item) => item.path === updated.path ? updated : item));
      setToast(`Skill ${updated.path} 已保存`);
    } catch (error) {
      setSkillsError(error instanceof Error ? error.message : '保存 Skill 失败');
    } finally {
      setSavingSkill(false);
    }
  }

  async function handleOpenHistory(sessionId: string) {
    try {
      setPageError(null);
      await refreshSession(sessionId);
      window.localStorage.setItem(SESSION_STORAGE_KEY, sessionId);
      setHistoryOpen(false);
      setEditorMode(false);
      setToast('已切换到历史会话');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '切换历史会话失败');
    }
  }

  async function handleRenameHistorySession(sessionId: string, title: string) {
    try {
      setSavingSessionId(sessionId);
      const updated = await updateAISessionMeta(sessionId, { title });
      setHistorySessions((previous) => previous.map((item) => item.id === updated.id ? {
        ...item,
        title: updated.title,
        updatedAt: updated.updatedAt,
      } : item));
      if (session?.id === updated.id) {
        setSession(updated);
      }
      setToast('会话名称已更新');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '更新会话名称失败');
    } finally {
      setSavingSessionId(null);
    }
  }

  async function handleDeleteHistorySession(sessionId: string) {
    const target = historySessions.find((item) => item.id === sessionId);
    const confirmed = window.confirm(`确认删除历史会话“${target?.title || sessionId}”吗？这只会删除 AI 会话记录和附件，不会删除已保存到 templates/ 的模板文件。`);
    if (!confirmed) return;

    try {
      setSavingSessionId(sessionId);
      await deleteAISession(sessionId);
      const remaining = historySessions.filter((item) => item.id !== sessionId);
      setHistorySessions(remaining);

      if (session?.id === sessionId) {
        const nextSessionId = remaining[0]?.id;
        if (nextSessionId) {
          await handleOpenHistory(nextSessionId);
        } else {
          await handleCreateSession();
          setHistoryOpen(false);
        }
      }
      setToast('历史会话已删除');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '删除历史会话失败');
    } finally {
      setSavingSessionId(null);
    }
  }

  function openDraftEditor(path: string) {
    setSelectedDraftPath(path);
    setEditorMode(true);
  }

  function handleResizerMouseDown(event: React.MouseEvent<HTMLDivElement>) {
    if (!layoutRef.current) return;
    const rect = layoutRef.current.getBoundingClientRect();
    resizeSessionRef.current = {
      startX: event.clientX,
      startRatio: splitRatio,
      width: rect.width,
    };
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  }

  if (loading) {
    return <div className="card"><div className="card-body ai-page-loading"><LoaderCircle className="w-6 h-6 animate-spin text-primary" /><span className="text-gray-600">正在初始化 AI 模板工作台...</span></div></div>;
  }

  if (!session) {
    return <div className="card"><div className="card-body text-center"><p className="text-danger">{pageError || 'AI 会话初始化失败'}</p><button className="btn btn-primary mt-4" onClick={() => void bootstrap()}><RefreshCw className="w-4 h-4" />重新加载</button></div></div>;
  }

  return (
    <div className={`ai-chat-shell ${editorMode ? 'ai-chat-shell-editor' : ''}`}>
      <div className="ai-chat-topbar">
        <div className="ai-chat-topbar-main">
          <div className="ai-chat-topbar-mark">AI 模板</div>
          <div className="ai-chat-topbar-copy">
            <h2 className="m-0">对话生成模板草稿，确认后再写入 templates/</h2>
            <p className="m-0">当前会话 {session.title || '新会话'} · {session.id} · {session.messages.length} 条消息 · {session.attachments.length} 个附件</p>
          </div>
        </div>
        <div className="ai-chat-topbar-actions">
          <button className="btn btn-secondary btn-sm ai-toolbar-btn" onClick={() => { setSettingsError(null); setSettingsTestSucceeded(false); setSettingsOpen(true); }}><Settings2 className="w-4 h-4" />AI 设置</button>
          <button className="btn btn-secondary btn-sm ai-toolbar-btn" onClick={() => setRulesOpen(true)}><ShieldCheck className="w-4 h-4" />全局规则</button>
          <button className="btn btn-secondary btn-sm ai-toolbar-btn" onClick={() => void loadSkillCatalog(true)}><Sparkles className="w-4 h-4" />Skill 管理</button>
          <button className="btn btn-secondary btn-sm ai-toolbar-btn" onClick={() => { void loadHistorySessions(); setHistoryOpen(true); }}><History className="w-4 h-4" />历史会话</button>
          <button className="btn btn-secondary btn-sm ai-toolbar-btn" onClick={() => void refreshSession()}><RefreshCw className="w-4 h-4" />刷新</button>
          <button className="btn btn-primary btn-sm ai-toolbar-btn" onClick={() => void handleCreateSession()}><CopyPlus className="w-4 h-4" />新建会话</button>
        </div>
      </div>

      <input ref={textUploadRef} type="file" multiple accept=".sh,.tmpl,.conf,.yaml,.yml,.xml,.properties,.json,.md,.txt" style={{ display: 'none' }} onChange={(event) => void handleUpload(event.target.files)} />
      <input ref={imageUploadRef} type="file" multiple accept=".png,.jpg,.jpeg,.webp,image/png,image/jpeg,image/webp" style={{ display: 'none' }} onChange={(event) => void handleUpload(event.target.files)} />

      <div
        ref={layoutRef}
        className={`ai-chat-layout ${editorMode ? 'is-editor-open' : ''}`}
        style={editorMode && !isCompactViewport ? { gridTemplateColumns: `minmax(0, ${splitRatio}fr) 14px minmax(320px, ${100 - splitRatio}fr)` } : undefined}
      >
        <main className="ai-chat-main">
          {pageError && <div className="ai-inline-alert ai-inline-alert-danger"><span>{pageError}</span><button className="modal-close" onClick={() => setPageError(null)} aria-label="关闭错误提示"><X className="w-4 h-4" /></button></div>}
          {toast && <div className="ai-inline-alert ai-inline-alert-success"><div className="flex items-center gap-2"><Check className="w-4 h-4" /><span>{toast}</span></div><button className="modal-close" onClick={() => setToast(null)} aria-label="关闭提示"><X className="w-4 h-4" /></button></div>}

          <div className="ai-chat-feed">
            {threadMessages.length === 0 ? (
              <div className="card ai-thread-empty-card"><div className="card-body empty-state ai-chat-empty"><Bot className="empty-state-icon" /><p>可以直接描述目标服务和输出文件，例如“生成 elasticsearch 3 节点集群安装脚本模板，目标是 templates/elasticsearch/install.sh.tmpl”。</p></div></div>
            ) : threadMessages.map((item) => {
              const messageDrafts = item.draftFiles ?? [];
              const messagePlans = item.plannedActions ?? [];
              const patchServiceNames = getMessagePatchServices(item);
              const messageIssues = item.configIssues ?? [];
              const isLatestAssistant = latestAssistantMessage?.id === item.id;
              const messageAttachments = getMessageAttachments(session, item);
              return (
                <article key={item.id} className={`ai-thread-message ${item.role === 'assistant' ? 'ai-thread-message-assistant' : 'ai-thread-message-user'}`}>
                  <div className="ai-thread-meta"><div className="ai-thread-meta-main"><div className={`ai-thread-avatar ${item.role === 'assistant' ? 'assistant' : 'user'}`}>{item.role === 'assistant' ? <Sparkles className="w-4 h-4" /> : <Wand2 className="w-4 h-4" />}</div><div><div className="ai-thread-author">{item.role === 'assistant' ? 'AI 模板助手' : '你'}</div><div className={`ai-thread-time ${item.failed ? 'text-danger' : ''}`}>{item.pending ? '处理中...' : item.failed ? '发送失败' : formatTime(item.createdAt)}</div></div></div></div>
                  {item.role === 'assistant' && (item.promptTrace || (item.streamStatus && item.streamStatus.length > 0)) && <details className="ai-thread-thinking" open><summary>{item.pending ? '生成中' : '查看生成过程'}</summary><div className="ai-thread-thinking-content">{item.streamStatus?.map((status) => <div key={`${item.id}-${status}`} className="ai-thread-thinking-line">{status}</div>)}{item.promptTrace && <><div className="ai-thread-thinking-line">已命中 {item.promptTrace.skillRefs.length} 条规则，参考 {item.promptTrace.exampleRefs.length} 个模板示例。</div><div className="ai-thread-thinking-line">模式: {item.promptTrace.mode} · 协议: {item.promptTrace.provider}</div></>}{item.attachmentIds && item.attachmentIds.length > 0 && <div className="ai-thread-thinking-line">本轮引用了 {item.attachmentIds.length} 个附件材料。</div>}</div></details>}
                  <div className="ai-thread-content">{item.pending ? <span className="flex items-start gap-2"><LoaderCircle className="w-4 h-4 animate-spin mt-0.5" /><span>{item.content || '正在思考并整理回复...'}</span></span> : (item.content || '(空消息)')}</div>
                  {messageAttachments.length > 0 && (
                    item.role === 'user'
                      ? <div className="ai-user-attachment-grid">{messageAttachments.map((attachment) => attachment.kind === 'image'
                        ? <figure key={`${item.id}-${attachment.id}`} className="ai-user-image-card"><img src={attachment.downloadUrl} alt={attachment.name} className="ai-user-image-preview" /><figcaption className="ai-user-image-caption">{attachment.name} · 图片已发送 · {formatBytes(attachment.size)}</figcaption></figure>
                        : <div key={`${item.id}-${attachment.id}`} className="ai-user-file-card"><FileText className="w-4 h-4 text-primary" /><div className="ai-context-chip-copy"><span className="ai-context-chip-name">{attachment.name}</span><span className="ai-context-chip-meta">文本附件已发送 · {formatBytes(attachment.size)}</span></div></div>)}</div>
                      : <div className="ai-context-badges">{messageAttachments.map((attachment) => <div key={`${item.id}-${attachment.id}`} className="ai-context-chip">{attachment.kind === 'image' ? <img src={attachment.downloadUrl} alt={attachment.name} className="ai-context-chip-thumb" /> : <FileText className="w-4 h-4 text-primary" />}<div className="ai-context-chip-copy"><span className="ai-context-chip-name">{attachment.name}</span><span className="ai-context-chip-meta">{attachment.kind === 'image' ? '图片识别已发送' : '文本附件已发送'} · {formatBytes(attachment.size)}</span></div></div>)}</div>
                  )}
                  {item.warnings && item.warnings.length > 0 && <div className="ai-thread-warning-block"><div className="ai-thread-warning-title">人工复核提示</div><div className="ai-thread-warning-list">{item.warnings.map((warning) => <div key={warning}>{warning}</div>)}</div></div>}
                  {messagePlans.length > 0 && <div className="ai-inline-plan-list">{messagePlans.map((action) => <div key={`${item.id}-${action.type}-${action.path}`} className="ai-inline-plan-item"><div className="ai-inline-plan-type">{action.type === 'mkdir' ? '创建目录' : '写入文件'}</div><div className="ai-inline-plan-path">{action.path}</div><div className="ai-inline-plan-reason">{action.reason}</div></div>)}</div>}
                  {patchServiceNames.length > 0 && <div className="ai-inline-patch-card"><div className="ai-inline-patch-header"><div><div className="ai-inline-patch-title">配置同步建议</div><div className="ai-inline-patch-subtitle">保存模板时可以一并同步到 serviceTop 和 serverConfig.vars。</div></div><div className="flex items-center gap-2"><span className="badge badge-gray">{patchServiceNames.length} 个服务</span>{messageIssues.some((issue) => issue.severity === 'error') && <span className="badge badge-danger">{messageIssues.filter((issue) => issue.severity === 'error').length} 个阻塞项</span>}</div></div><div className="ai-inline-patch-list">{patchServiceNames.map((serviceName) => { const topo = item.configPatch?.serviceTop?.[serviceName]; const serviceCfg = item.configPatch?.serverConfig?.[serviceName]; const serviceIssues = messageIssues.filter((issue) => issue.service === serviceName); return <div key={`${item.id}-${serviceName}`} className="ai-inline-patch-item"><div className="flex items-center justify-between gap-2"><div className="font-medium text-gray-800">{serviceName}</div><div className="flex items-center gap-2">{topo && <span className="badge badge-blue">serviceTop</span>}{serviceCfg && <span className="badge badge-gray">vars {Object.keys(serviceCfg.vars || {}).length}</span>}</div></div>{topo && <div className="text-sm text-gray-600">节点: {topo.nodes?.join(', ') || '未设置'} · id_auto_derive: {String(Boolean(topo.id_auto_derive))}</div>}{serviceCfg && Object.keys(serviceCfg.vars || {}).length > 0 && <div className="text-sm text-gray-600">变量: {Object.keys(serviceCfg.vars || {}).join(', ')}</div>}{serviceIssues.length > 0 && <div className="ai-inline-patch-issues">{serviceIssues.map((issue) => <div key={`${item.id}-${issue.service}-${issue.field}-${issue.message}`} className={issue.severity === 'error' ? 'text-danger' : issue.severity === 'warning' ? 'text-warning' : 'text-gray-600'}>{issue.severity === 'error' ? '阻塞' : issue.severity === 'warning' ? '注意' : '提示'}: {issue.message}</div>)}</div>}</div>; })}</div></div>}
                  {messageDrafts.length > 0 && <div className="ai-inline-draft-list">{messageDrafts.map((draft) => <div key={`${item.id}-${draft.path}`} className="ai-inline-draft-card"><div className="ai-inline-draft-header"><div className="ai-inline-draft-title-block"><div className="ai-inline-draft-title"><FileText className="w-4 h-4 text-primary" /><span>{draft.path}</span></div><div className="ai-inline-draft-meta">{draft.reason || 'AI 生成的模板草稿'}</div></div><div className="ai-inline-draft-actions">{draft.source === 'image' && <span className="badge badge-danger">图片识别</span>}<span className={`badge ${draft.needsReview ? 'badge-blue' : 'badge-green'}`}>{draft.needsReview ? '待审核' : '已保存'}</span><button className="btn btn-sm btn-secondary" onClick={() => openDraftEditor(draft.path)}>编辑</button><button className="btn btn-sm btn-secondary btn-danger-soft" onClick={() => void handleDeleteDraft(draft.path)} disabled={savingDrafts}><Trash2 className="w-4 h-4" />删除</button></div></div><pre className="ai-inline-code-block">{draft.content}</pre></div>)}</div>}
                  {item.followUpQuestions && item.followUpQuestions.length > 0 && <div className="ai-follow-up-list">{item.followUpQuestions.map((question) => <div key={question} className="ai-follow-up-chip">{question}</div>)}</div>}
                  {isLatestAssistant && hasAnyDraftFiles && <div className="ai-answer-actions"><button className="btn btn-primary" onClick={() => void handleSaveDrafts(session.draftFiles)} disabled={savingDrafts || session.draftFiles.length === 0}>{savingDrafts ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}保存全部</button><button className="btn btn-secondary" onClick={() => void handleSaveDrafts(session.draftFiles, true)} disabled={savingDrafts || session.draftFiles.length === 0 || !hasAnyConfigPatch}>{savingDrafts ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <ShieldCheck className="w-4 h-4" />}保存并同步配置</button><button className="btn btn-secondary" onClick={() => latestAssistantDrafts[0] && openDraftEditor(latestAssistantDrafts[0].path)} disabled={latestAssistantDrafts.length === 0}>编辑模板</button></div>}
                </article>
              );
            })}
            <div ref={messageListEndRef} />
          </div>

          <div className="ai-chat-footer">
            <div
              className={`ai-chat-composer ${composerDragging ? 'is-dragging' : ''}`}
              onDragOver={handleComposerDragOver}
              onDragLeave={handleComposerDragLeave}
              onDrop={handleComposerDrop}
            >
              <div className="ai-chat-context-row">
                <div className="ai-chat-context-label">附件与上下文</div>
                <div className="ai-chat-context-actions">
                  <div className="ai-chat-model-switch">
                    <Cpu className="w-4 h-4 text-primary" />
                    <select
                      className="select ai-chat-model-select"
                      value={modelSuggestions.includes(selectedModel) ? selectedModel : ''}
                      onChange={(event) => {
                        if (event.target.value) {
                          setSelectedModel(event.target.value);
                        }
                      }}
                    >
                      <option value="">选择模型</option>
                      {modelSuggestions.map((item) => <option key={item} value={item}>{item}</option>)}
                    </select>
                  </div>
                  <button className="btn btn-sm btn-secondary" onClick={() => textUploadRef.current?.click()} disabled={uploading}><Upload className="w-4 h-4" />上传文本</button>
                  <button className="btn btn-sm btn-secondary" onClick={() => imageUploadRef.current?.click()} disabled={uploading}><ImagePlus className="w-4 h-4" />上传图片</button>
                </div>
              </div>
              {uploading && <div className="ai-context-badges"><span className="badge badge-blue"><LoaderCircle className="w-3 h-3 animate-spin" />正在上传附件</span></div>}
              {visiblePendingAttachments.length > 0 && <div className="ai-context-badges"><span className="badge badge-blue">{visiblePendingAttachments.length} 个附件待发送</span>{visiblePendingAttachments.map((attachment) => <div key={attachment.id} className="ai-context-chip">{attachment.kind === 'image' ? <img src={attachment.downloadUrl} alt={attachment.name} className="ai-context-chip-thumb" /> : <FileText className="w-4 h-4 text-primary" />}<div className="ai-context-chip-copy"><span className="ai-context-chip-name">{attachment.name}</span><span className="ai-context-chip-meta">{attachment.kind === 'image' ? '图片识别' : '文本'} · {formatBytes(attachment.size)}</span></div><span className="badge badge-blue">待发送</span><button className="modal-close" onClick={() => void handleDeleteAttachment(attachment.id)} aria-label="取消附件"><X className="w-4 h-4" /></button></div>)}</div>}
              <details className="ai-session-rules-panel"><summary>本次补充规则</summary><textarea className="input ai-chat-rules-input" value={sessionRules} onChange={(event) => setSessionRules(event.target.value)} placeholder="例如：统一使用 run_as_root，安装脚本必须幂等，优先复用 .Global.user 和 .Global.install_base_dir。" /></details>
              <textarea className="input ai-chat-input" value={message} onPaste={handleComposerPaste} onChange={(event) => setMessage(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); void handleSendMessage(); } }} placeholder="描述你要生成或改写的模板。回车发送，Shift + Enter 换行。" />
              <div className="ai-chat-composer-actions"><div className="ai-chat-composer-hint">支持上传脚本、配置文件和文本截图，也支持 Ctrl+V 粘贴图片/文档或拖拽文件；AI 先生成草稿，再由你审核保存。</div><button className="btn btn-primary" onClick={() => void handleSendMessage()} disabled={sending || uploading || (!message.trim() && pendingAttachments.length === 0)}>{sending ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}发送</button></div>
            </div>
          </div>
        </main>

        {editorMode && !isCompactViewport && <div className="ai-chat-resizer" onMouseDown={handleResizerMouseDown} role="separator" aria-orientation="vertical" aria-label="调整对话与编辑器宽度"><span className="ai-chat-resizer-grip" /></div>}
        {editorMode && <aside className="ai-chat-editor-panel"><div className="ai-chat-editor-header"><div><div className="ai-chat-editor-title">编辑模板</div><div className="ai-chat-editor-subtitle">{selectedDraft ? selectedDraft.reason || '人工审核并修改草稿内容' : '选择一个草稿开始编辑'}</div></div><button className="btn btn-secondary" onClick={() => setEditorMode(false)}>退出编辑模式</button></div>{selectedDraft ? <><div className="ai-chat-editor-pathbar"><div className="form-group ai-draft-path-group"><label className="form-label">目标路径</label><input className="input" value={selectedDraft.path} onChange={(event) => { const currentPath = selectedDraft.path; const nextPath = event.target.value; updateDraftContent(currentPath, (draft) => ({ ...draft, path: nextPath, needsReview: true })); setSelectedDraftPath(nextPath); }} /></div><div className="ai-chat-editor-toolbar">{selectedDraft.source === 'image' && <span className="badge badge-danger">来自图片识别，需重点人工复核</span>}<span className="badge badge-gray">{getDraftLanguage(selectedDraft.path)}</span></div></div><div className="ai-chat-editor-monaco"><Editor height="100%" language={getDraftLanguage(selectedDraft.path)} value={selectedDraft.content} onChange={(value) => updateDraftContent(selectedDraft.path, (draft) => ({ ...draft, content: value ?? '', needsReview: true }))} options={{ minimap: { enabled: false }, fontSize: 13, lineNumbers: 'on', scrollBeyondLastLine: false, wordWrap: 'off', theme: 'vs', mouseWheelZoom: false, scrollbar: { alwaysConsumeMouseWheel: false } }} /></div><div className="ai-chat-editor-actions"><button className="btn btn-secondary btn-danger-soft" onClick={() => void handleDeleteDraft(selectedDraft.path)} disabled={savingDrafts}><Trash2 className="w-4 h-4" />删除模板</button><button className="btn btn-secondary" onClick={() => void handleSaveDrafts([selectedDraft], true)} disabled={savingDrafts || !selectedServiceHasConfigPatch}>{savingDrafts ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <ShieldCheck className="w-4 h-4" />}保存并同步配置</button><button className="btn btn-primary" onClick={() => void handleSaveDrafts([selectedDraft])} disabled={savingDrafts}>{savingDrafts ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}保存当前</button></div></> : <div className="empty-state ai-draft-empty"><FolderPlus className="empty-state-icon" /><p>当前还没有可编辑的模板草稿。先让 AI 生成一个模板文件，再点击“编辑”。</p></div>}</aside>}
      </div>

      <SettingsModal open={settingsOpen} settings={settings} modelOptions={modelSuggestions} modelsLoading={loadingModels} modelsError={modelsError} saving={savingSettings} testing={testingSettings} testSucceeded={settingsTestSucceeded} error={settingsError} onClose={() => { setSettingsOpen(false); setSettingsError(null); setSettingsTestSucceeded(false); }} onRefreshModels={refreshAvailableModels} onSave={handleSaveSettings} onTest={handleTestSettings} />
      <RulesModal open={rulesOpen} content={rules} saving={savingRules} onClose={() => setRulesOpen(false)} onChange={setRules} onSave={handleSaveRules} />
      <SkillManagerModal
        open={skillsOpen}
        skills={skillCatalog}
        examples={skillExamples}
        loading={loadingSkills}
        saving={savingSkill}
        error={skillsError}
        onClose={() => {
          setSkillsOpen(false);
          setSkillsError(null);
        }}
        onRefresh={() => loadSkillCatalog()}
        onSave={handleSaveSkillFile}
      />
      <SessionHistoryModal
        open={historyOpen}
        sessions={historySessions}
        activeSessionId={session?.id ?? null}
        loading={loadingHistory}
        savingSessionId={savingSessionId}
        onClose={() => setHistoryOpen(false)}
        onRefresh={loadHistorySessions}
        onOpenSession={handleOpenHistory}
        onRenameSession={handleRenameHistorySession}
        onDeleteSession={handleDeleteHistorySession}
      />
    </div>
  );
}
