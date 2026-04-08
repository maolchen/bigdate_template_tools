import { useEffect, useMemo, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import {
  Bot,
  Check,
  CopyPlus,
  FileText,
  FolderPlus,
  ImagePlus,
  LoaderCircle,
  MessageSquareText,
  RefreshCw,
  Save,
  Send,
  Settings2,
  ShieldCheck,
  Sparkles,
  Upload,
  Wand2,
  X,
} from 'lucide-react';
import {
  createAISession,
  fetchAIRules,
  fetchAISettings,
  fetchAISession,
  saveAIRules,
  saveAISessionDrafts,
  saveAISettings,
  sendAISessionMessage,
  testAISettings,
  uploadAISessionAttachment,
  type AIDraftFile,
  type AISettings,
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

interface SettingsModalProps {
  open: boolean;
  settings: AISettings | null;
  saving: boolean;
  testing: boolean;
  testSucceeded: boolean;
  error: string | null;
  onClose: () => void;
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

function SettingsModal({
  open,
  settings,
  saving,
  testing,
  testSucceeded,
  error,
  onClose,
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
            <p className="text-sm text-gray-500 mt-1 m-0">
              配置 OpenAI 兼容接口的 Base URL、Model 和 API Key。
            </p>
          </div>
          <button className="modal-close" onClick={onClose} aria-label="关闭">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="modal-body">
          <div className="form-group">
            <label className="form-label">Base URL</label>
            <input
              className="input"
              value={baseUrl}
              onChange={(event) => setBaseUrl(event.target.value)}
              placeholder="https://api.openai.com/v1"
            />
          </div>

          <div className="form-group">
            <label className="form-label">Model</label>
            <input
              className="input"
              value={model}
              onChange={(event) => setModel(event.target.value)}
              placeholder="gpt-4.1"
            />
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
              <input
                id="clear-api-key"
                type="checkbox"
                checked={clearApiKey}
                onChange={(event) => setClearApiKey(event.target.checked)}
              />
              <label htmlFor="clear-api-key">清空已保存的 API Key</label>
            </div>
          </div>

          {settings?.updatedAt && (
            <div className="badge badge-gray">最近更新: {formatTime(settings.updatedAt)}</div>
          )}

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
          <button className="btn btn-secondary" onClick={onClose}>
            取消
          </button>
          <button
            className={`btn ${testSucceeded && !testing ? 'btn-success' : 'btn-secondary'}`}
            onClick={() => void onTest({ baseUrl, model, apiKey, clearApiKey })}
            disabled={testing || saving}
          >
            {testing ? (
              <LoaderCircle className="w-4 h-4 animate-spin" />
            ) : testSucceeded ? (
              <Check className="w-4 h-4" />
            ) : (
              <ShieldCheck className="w-4 h-4" />
            )}
            {testSucceeded && !testing ? '测试通过' : '测试连接'}
          </button>
          <button
            className="btn btn-primary"
            onClick={() => void onSave({ baseUrl, model, apiKey, clearApiKey })}
            disabled={saving}
          >
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
            <p className="text-sm text-gray-500 mt-1 m-0">
              这些规则会自动拼接到每次 AI 模板生成的系统提示词里。
            </p>
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

          <textarea
            className="input ai-rules-textarea"
            value={content}
            onChange={(event) => onChange(event.target.value)}
            placeholder="输入全局模板规范补充..."
          />

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
          <button className="btn btn-secondary" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" onClick={() => void onSave()} disabled={saving}>
            {saving ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
            保存规则
          </button>
        </div>
      </div>
    </div>
  );
}

export function AITemplatePage() {
  const [settings, setSettings] = useState<AISettings | null>(null);
  const [rules, setRules] = useState('');
  const [session, setSession] = useState<AITemplateSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [pageError, setPageError] = useState<string | null>(null);
  const [message, setMessage] = useState('');
  const [sessionRules, setSessionRules] = useState('');
  const [selectedDraftPath, setSelectedDraftPath] = useState<string | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [rulesOpen, setRulesOpen] = useState(false);
  const [settingsError, setSettingsError] = useState<string | null>(null);
  const [settingsTestSucceeded, setSettingsTestSucceeded] = useState(false);
  const [savingSettings, setSavingSettings] = useState(false);
  const [testingSettings, setTestingSettings] = useState(false);
  const [savingRules, setSavingRules] = useState(false);
  const [sending, setSending] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [savingDrafts, setSavingDrafts] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const textUploadRef = useRef<HTMLInputElement | null>(null);
  const imageUploadRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    void bootstrap();
  }, []);

  useEffect(() => {
    setSessionRules(session?.sessionRules || '');
  }, [session?.id]);

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

  async function bootstrap() {
    try {
      setLoading(true);
      setPageError(null);
      const [settingsData, rulesData] = await Promise.all([fetchAISettings(), fetchAIRules()]);
      setSettings(settingsData);
      setRules(rulesData.content || '');

      const storedSessionId = window.localStorage.getItem(SESSION_STORAGE_KEY);
      if (storedSessionId) {
        try {
          const currentSession = await fetchAISession(storedSessionId);
          setSession(currentSession);
          return;
        } catch {
          window.localStorage.removeItem(SESSION_STORAGE_KEY);
        }
      }

      const nextSession = await createAISession();
      window.localStorage.setItem(SESSION_STORAGE_KEY, nextSession.id);
      setSession(nextSession);
    } catch (error) {
      setPageError(error instanceof Error ? error.message : 'AI 工作台初始化失败');
    } finally {
      setLoading(false);
    }
  }

  const selectedDraft = useMemo(
    () => session?.draftFiles.find((draft) => draft.path === selectedDraftPath) ?? null,
    [session, selectedDraftPath],
  );

  const pendingAttachments = session?.attachments.filter((attachment) => attachment.pending) ?? [];
  const latestAssistantMessage = [...(session?.messages ?? [])]
    .reverse()
    .find((item) => item.role === 'assistant') ?? null;
  const configPatchServiceNames = useMemo(() => {
    if (!session) return [];
    const names = new Set([
      ...Object.keys(session.configPatch?.serviceTop || {}),
      ...Object.keys(session.configPatch?.serverConfig || {}),
    ]);
    return [...names].sort();
  }, [session]);
  const hasAnyConfigPatch = configPatchServiceNames.length > 0;
  const blockingConfigIssues = session?.configIssues.filter((issue) => issue.severity === 'error') ?? [];
  const selectedDraftService = selectedDraft ? getServiceNameFromTemplatePath(selectedDraft.path) : null;
  const selectedServiceHasConfigPatch = selectedDraftService
    ? Boolean(session?.configPatch?.serviceTop?.[selectedDraftService] || session?.configPatch?.serverConfig?.[selectedDraftService])
    : false;

  async function refreshSession(currentSessionId?: string) {
    const targetSessionId = currentSessionId ?? session?.id;
    if (!targetSessionId) return;
    const currentSession = await fetchAISession(targetSessionId);
    setSession(currentSession);
  }

  async function handleCreateSession() {
    try {
      setPageError(null);
      const nextSession = await createAISession();
      window.localStorage.setItem(SESSION_STORAGE_KEY, nextSession.id);
      setSession(nextSession);
      setSelectedDraftPath(null);
      setMessage('');
      setSessionRules('');
      setToast('已创建新的 AI 模板会话');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '创建会话失败');
    }
  }

  async function handleSendMessage() {
    if (!session) return;
    if (!message.trim() && pendingAttachments.length === 0) return;

    try {
      setSending(true);
      setPageError(null);
      const updated = await sendAISessionMessage(session.id, {
        message,
        sessionRules,
        selectedDraftPaths: selectedDraftPath ? [selectedDraftPath] : [],
      });
      setSession(updated);
      setMessage('');
      if (updated.draftFiles.length > 0 && !selectedDraftPath) {
        setSelectedDraftPath(updated.draftFiles[0].path);
      }
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '发送消息失败');
    } finally {
      setSending(false);
    }
  }

  async function handleUpload(files: FileList | null) {
    if (!session || !files || files.length === 0) return;

    try {
      setUploading(true);
      setPageError(null);
      for (const file of Array.from(files)) {
        const attachment = await uploadAISessionAttachment(session.id, file);
        setSession((previous) => {
          if (!previous) return previous;
          return {
            ...previous,
            attachments: [...previous.attachments, attachment],
          };
        });
      }
      setToast('附件已加入当前消息上下文');
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '上传附件失败');
    } finally {
      setUploading(false);
      if (textUploadRef.current) textUploadRef.current.value = '';
      if (imageUploadRef.current) imageUploadRef.current.value = '';
    }
  }

  function updateDraftContent(targetPath: string, updater: (draft: AIDraftFile) => AIDraftFile) {
    setSession((previous) => {
      if (!previous) return previous;
      return {
        ...previous,
        draftFiles: previous.draftFiles.map((draft) => (
          draft.path === targetPath ? updater(draft) : draft
        )),
      };
    });
  }

  async function handleSaveDrafts(files: AIDraftFile[], applyConfigPatch = false) {
    if (!session || files.length === 0) return;

    try {
      setSavingDrafts(true);
      setPageError(null);
      const result = await saveAISessionDrafts(session.id, { files, applyConfigPatch });
      await refreshSession(session.id);
      if (applyConfigPatch) {
        if (result.configApplied) {
          setToast(`已保存 ${result.savedFiles.length} 个模板文件，并同步 ${result.appliedServices.length} 个服务配置`);
        } else {
          setToast(`模板已保存，但配置未同步，请先处理 ${result.configIssues.length} 个配置问题`);
        }
      } else {
        setToast(`已保存 ${result.savedFiles.length} 个模板文件`);
      }
    } catch (error) {
      setPageError(error instanceof Error ? error.message : '保存模板失败');
    } finally {
      setSavingDrafts(false);
    }
  }

  async function handleSaveSettings(payload: {
    baseUrl: string;
    model: string;
    apiKey: string;
    clearApiKey: boolean;
  }) {
    try {
      setSavingSettings(true);
      setSettingsError(null);
      setSettingsTestSucceeded(false);
      const updated = await saveAISettings(payload);
      setSettings(updated);
      setToast('AI 设置已保存');
      setSettingsOpen(false);
    } catch (error) {
      setSettingsError(error instanceof Error ? error.message : '保存 AI 设置失败');
    } finally {
      setSavingSettings(false);
    }
  }

  async function handleTestSettings(payload: {
    baseUrl: string;
    model: string;
    apiKey: string;
    clearApiKey: boolean;
  }) {
    try {
      setTestingSettings(true);
      setSettingsError(null);
      setSettingsTestSucceeded(false);
      const result = await testAISettings(payload);
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

  if (loading) {
    return (
      <div className="card">
        <div className="card-body ai-page-loading">
          <LoaderCircle className="w-6 h-6 animate-spin text-primary" />
          <span className="text-gray-600">正在初始化 AI 模板工作台...</span>
        </div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="card">
        <div className="card-body text-center">
          <p className="text-danger">{pageError || 'AI 会话初始化失败'}</p>
          <button className="btn btn-primary mt-4" onClick={() => void bootstrap()}>
            <RefreshCw className="w-4 h-4" />
            重新加载
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="ai-workbench">
      <div className="card mb-6 ai-banner-card">
        <div className="card-body ai-banner-body">
          <div>
            <div className="badge badge-blue mb-2">AI 模板生成工作台</div>
            <h2 className="text-2xl font-bold text-gray-800 m-0">
              先生成草稿，再人工确认保存到 templates/
            </h2>
            <p className="text-gray-500 mt-2 m-0">
              支持对话生成、文本改写、图片识别辅助改写，以及模板草稿的人工审核与编辑。
            </p>
          </div>
          <div className="ai-banner-actions">
            <button
              className="btn btn-secondary"
              onClick={() => {
                setSettingsError(null);
                setSettingsTestSucceeded(false);
                setSettingsOpen(true);
              }}
            >
              <Settings2 className="w-4 h-4" />
              AI 设置
            </button>
            <button className="btn btn-secondary" onClick={() => setRulesOpen(true)}>
              <ShieldCheck className="w-4 h-4" />
              全局规则
            </button>
            <button className="btn btn-primary" onClick={() => void handleCreateSession()}>
              <CopyPlus className="w-4 h-4" />
              新建会话
            </button>
          </div>
        </div>
      </div>

      {pageError && (
        <div className="card mb-6" style={{ borderColor: 'var(--danger)', borderWidth: '1px' }}>
          <div className="card-body text-danger text-sm">{pageError}</div>
        </div>
      )}

      {toast && (
        <div className="card mb-6" style={{ borderColor: 'var(--success)', borderWidth: '1px' }}>
          <div className="card-body flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 text-success">
              <Check className="w-4 h-4" />
              <span>{toast}</span>
            </div>
            <button className="modal-close" onClick={() => setToast(null)} aria-label="关闭提示">
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}

      <div className="ai-workbench-grid">
        <section className="ai-chat-column">
          <div className="card ai-session-card">
            <div className="card-header">
              <div>
                <div className="flex items-center gap-2">
                  <MessageSquareText className="w-4 h-4 text-primary" />
                  <span className="font-semibold text-gray-800">对话与上下文</span>
                </div>
                <p className="text-sm text-gray-500 mt-1 m-0">会话 ID: {session.id}</p>
              </div>
              <div className="flex items-center gap-2">
                <span className="badge badge-gray">{session.messages.length} 条消息</span>
                <span className="badge badge-gray">{session.attachments.length} 个附件</span>
              </div>
            </div>

            <div className="card-body ai-session-body">
              <div className="form-group">
                <label className="form-label">本次补充规则</label>
                <textarea
                  className="input ai-session-rules"
                  value={sessionRules}
                  onChange={(event) => setSessionRules(event.target.value)}
                  placeholder="例如：统一使用 run_as_root，安装脚本需要包含 systemd 服务创建。"
                />
              </div>

              <div className="ai-message-list">
                {session.messages.length === 0 ? (
                  <div className="empty-state ai-chat-empty">
                    <Bot className="empty-state-icon" />
                    <p>输入需求开始生成模板草稿，例如“创建 elasticsearch 目录并生成 3 节点安装脚本模板”。</p>
                  </div>
                ) : (
                  session.messages.map((item) => (
                    <article key={item.id} className={`ai-message ai-message-${item.role}`}>
                      <div className="ai-message-meta">
                        <div className="flex items-center gap-2">
                          {item.role === 'assistant' ? <Sparkles className="w-4 h-4" /> : <Wand2 className="w-4 h-4" />}
                          <span>{item.role === 'assistant' ? 'AI' : '你'}</span>
                        </div>
                        <span>{formatTime(item.createdAt)}</span>
                      </div>
                      <div className="ai-message-content">{item.content || '(空消息)'}</div>

                      {item.warnings && item.warnings.length > 0 && (
                        <div className="ai-message-warnings">
                          {item.warnings.map((warning) => (
                            <span key={warning} className="badge badge-danger">
                              {warning}
                            </span>
                          ))}
                        </div>
                      )}

                      {item.followUpQuestions && item.followUpQuestions.length > 0 && (
                        <div className="ai-follow-up-list">
                          {item.followUpQuestions.map((question) => (
                            <div key={question} className="text-sm text-gray-600">
                              · {question}
                            </div>
                          ))}
                        </div>
                      )}
                    </article>
                  ))
                )}
              </div>

              <div className="ai-compose">
                {pendingAttachments.length > 0 && (
                  <div className="ai-pending-bar">
                    <span className="badge badge-blue">{pendingAttachments.length} 个附件将随下一条消息发送</span>
                  </div>
                )}

                <textarea
                  className="input ai-compose-input"
                  value={message}
                  onChange={(event) => setMessage(event.target.value)}
                  placeholder="描述你要创建或改写的模板。可以先让 AI 规划目录，再生成 install.sh.tmpl 或其他配置模板。"
                />

                <div className="flex items-center justify-between gap-2 mt-3">
                  <span className="text-xs text-gray-500">
                    支持纯对话、文本附件改写，以及截图/拍照识别后改写。
                  </span>
                  <button
                    className="btn btn-primary"
                    onClick={() => void handleSendMessage()}
                    disabled={sending || uploading}
                  >
                    {sending ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                    发送
                  </button>
                </div>
              </div>

              <div className="ai-attachments-panel ai-attachments-panel-compact">
                <div className="flex items-center justify-between gap-2 mb-3">
                  <div className="flex items-center gap-2">
                    <Upload className="w-4 h-4 text-primary" />
                    <span className="font-semibold text-gray-800">附件上下文</span>
                  </div>
                  <div className="flex gap-2">
                    <button
                      className="btn btn-sm btn-secondary"
                      onClick={() => textUploadRef.current?.click()}
                      disabled={uploading}
                    >
                      <FileText className="w-4 h-4" />
                      上传文本
                    </button>
                    <button
                      className="btn btn-sm btn-secondary"
                      onClick={() => imageUploadRef.current?.click()}
                      disabled={uploading}
                    >
                      <ImagePlus className="w-4 h-4" />
                      上传图片
                    </button>
                  </div>
                </div>

                <input
                  ref={textUploadRef}
                  type="file"
                  multiple
                  accept=".sh,.tmpl,.conf,.yaml,.yml,.xml,.properties,.json,.md,.txt"
                  style={{ display: 'none' }}
                  onChange={(event) => void handleUpload(event.target.files)}
                />
                <input
                  ref={imageUploadRef}
                  type="file"
                  multiple
                  accept=".png,.jpg,.jpeg,.webp,image/png,image/jpeg,image/webp"
                  style={{ display: 'none' }}
                  onChange={(event) => void handleUpload(event.target.files)}
                />

                {uploading && (
                  <div className="badge badge-blue">
                    <LoaderCircle className="w-3 h-3 animate-spin" />
                    正在上传附件
                  </div>
                )}

                <div className="ai-attachment-list ai-attachment-list-compact">
                  {session.attachments.length === 0 ? (
                    <div className="empty-state p-4">
                      <p>还没有上传附件</p>
                    </div>
                  ) : (
                    session.attachments.map((attachment) => (
                      <div key={attachment.id} className="ai-attachment-item ai-attachment-item-compact">
                        <div className="ai-attachment-item-main">
                          {attachment.kind === 'image' ? (
                            <img
                              className="ai-attachment-preview ai-attachment-preview-compact"
                              src={attachment.downloadUrl}
                              alt={attachment.name}
                            />
                          ) : (
                            <div className="ai-attachment-icon-chip">
                              <FileText className="w-4 h-4 text-primary" />
                            </div>
                          )}

                          <div className="ai-attachment-copy">
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="font-medium text-gray-800 text-sm">{attachment.name}</span>
                              {attachment.pending && <span className="badge badge-blue">待发送</span>}
                              {attachment.kind === 'image' ? (
                                <span className="badge badge-danger">需复核</span>
                              ) : (
                                <span className="badge badge-gray">文本</span>
                              )}
                            </div>
                            <div className="text-xs text-gray-500">
                              {attachment.kind === 'image' ? '图片识别上下文' : '文本上下文'} · {formatBytes(attachment.size)} ·{' '}
                              {formatTime(attachment.createdAt)}
                            </div>
                            <div className="ai-attachment-inline-preview">
                              {attachment.kind === 'image'
                                ? '图片将用于文字提取和模板改写，请重点人工复核。'
                                : attachment.previewText || '文本预览不可用'}
                            </div>
                          </div>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="ai-draft-column">
          <div className="card ai-draft-card">
            <div className="card-header">
              <div>
                <div className="flex items-center gap-2">
                  <FolderPlus className="w-4 h-4 text-primary" />
                  <span className="font-semibold text-gray-800">目录计划与草稿文件</span>
                </div>
                <p className="text-sm text-gray-500 mt-1 m-0">
                  AI 只生成草稿，点击保存后才会真正写入 templates/。
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button className="btn btn-sm btn-secondary" onClick={() => void refreshSession()}>
                  <RefreshCw className="w-4 h-4" />
                  刷新
                </button>
                <button
                  className="btn btn-sm btn-primary"
                  onClick={() => void handleSaveDrafts(session.draftFiles)}
                  disabled={savingDrafts || session.draftFiles.length === 0}
                >
                  {savingDrafts ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                  保存全部
                </button>
                <button
                  className="btn btn-sm btn-secondary"
                  onClick={() => void handleSaveDrafts(session.draftFiles, true)}
                  disabled={savingDrafts || session.draftFiles.length === 0 || !hasAnyConfigPatch}
                >
                  {savingDrafts ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <ShieldCheck className="w-4 h-4" />}
                  保存并同步配置
                </button>
              </div>
            </div>

            <div className="card-body ai-draft-body">
              <div className="ai-plan-strip">
                {session.plannedActions.length === 0 ? (
                  <div className="empty-state p-4">
                    <p>AI 还没有产出目录或文件计划</p>
                  </div>
                ) : (
                  session.plannedActions.map((action) => (
                    <div key={`${action.type}-${action.path}`} className="ai-plan-item">
                      <span className="badge badge-blue">{action.type === 'mkdir' ? '创建目录' : '写入文件'}</span>
                      <div className="font-medium text-gray-800 text-sm">{action.path}</div>
                      <div className="text-xs text-gray-500">{action.reason}</div>
                    </div>
                  ))
                )}
              </div>

              {hasAnyConfigPatch && (
                <div className="card">
                  <div className="card-body">
                    <div className="flex items-center justify-between gap-2 mb-3">
                      <div>
                        <div className="font-semibold text-gray-800">配置同步草案</div>
                        <div className="text-sm text-gray-500">
                          保存模板时可一并同步到 serviceTop 和 serverConfig.vars。
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="badge badge-gray">{configPatchServiceNames.length} 个服务</span>
                        {blockingConfigIssues.length > 0 && (
                          <span className="badge badge-danger">{blockingConfigIssues.length} 个阻塞项</span>
                        )}
                      </div>
                    </div>

                    <div className="ai-config-sync-list">
                      {configPatchServiceNames.map((serviceName) => {
                        const topo = session.configPatch?.serviceTop?.[serviceName];
                        const serviceCfg = session.configPatch?.serverConfig?.[serviceName];
                        const serviceIssues = session.configIssues.filter((issue) => issue.service === serviceName);

                        return (
                          <div key={serviceName} className="ai-config-sync-item">
                            <div className="flex items-center justify-between gap-2">
                              <div className="font-medium text-gray-800">{serviceName}</div>
                              <div className="flex items-center gap-2">
                                {topo && <span className="badge badge-blue">serviceTop</span>}
                                {serviceCfg && <span className="badge badge-gray">vars {Object.keys(serviceCfg.vars || {}).length}</span>}
                              </div>
                            </div>

                            {topo && (
                              <div className="text-sm text-gray-600">
                                节点: {topo.nodes?.join(', ') || '未设置'} · id_auto_derive: {String(Boolean(topo.id_auto_derive))}
                              </div>
                            )}

                            {serviceCfg && Object.keys(serviceCfg.vars || {}).length > 0 && (
                              <div className="text-sm text-gray-600">
                                变量: {Object.keys(serviceCfg.vars || {}).join(', ')}
                              </div>
                            )}

                            {serviceIssues.length > 0 && (
                              <div className="ai-config-issue-list">
                                {serviceIssues.map((issue) => (
                                  <div
                                    key={`${issue.service}-${issue.field}-${issue.message}`}
                                    className={`text-sm ${issue.severity === 'error' ? 'text-danger' : issue.severity === 'warning' ? 'text-warning' : 'text-gray-600'}`}
                                  >
                                    {issue.severity === 'error' ? '阻塞' : issue.severity === 'warning' ? '注意' : '提示'}: {issue.message}
                                  </div>
                                ))}
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                </div>
              )}

              {latestAssistantMessage?.warnings && latestAssistantMessage.warnings.length > 0 && (
                <div className="card mt-4" style={{ borderColor: 'var(--warning)', borderWidth: '1px' }}>
                  <div className="card-body">
                    <div className="flex items-center gap-2 mb-2" style={{ color: 'var(--warning)' }}>
                      <Sparkles className="w-4 h-4" />
                      <span className="font-semibold">AI 风险提示</span>
                    </div>
                    <div className="space-y-2">
                      {latestAssistantMessage.warnings.map((warning) => (
                        <div key={warning} className="text-sm text-gray-700">
                          · {warning}
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              <div className="ai-draft-layout">
                <div className="ai-draft-list">
                  {session.draftFiles.length === 0 ? (
                    <div className="empty-state">
                      <FileText className="empty-state-icon" />
                      <p>草稿会显示在这里。可以让 AI 输出 install.sh.tmpl、配置文件模板或 README 草稿。</p>
                    </div>
                  ) : (
                    session.draftFiles.map((draft) => (
                      <button
                        key={draft.path}
                        className={`ai-draft-item ${selectedDraftPath === draft.path ? 'active' : ''}`}
                        onClick={() => setSelectedDraftPath(draft.path)}
                      >
                        <div className="flex items-center gap-2">
                          <FileText className="w-4 h-4 text-primary" />
                          <span className="font-medium text-gray-800 text-sm">{draft.path}</span>
                        </div>
                        <div className="flex items-center gap-2">
                          {draft.source === 'image' && <span className="badge badge-danger">图片识别</span>}
                          {draft.needsReview ? (
                            <span className="badge badge-blue">待审核</span>
                          ) : (
                            <span className="badge badge-green">已保存</span>
                          )}
                        </div>
                        <p className="text-xs text-gray-500 m-0">{draft.reason}</p>
                      </button>
                    ))
                  )}
                </div>

                <div className="ai-draft-editor-panel">
                  {selectedDraft ? (
                    <>
                      <div className="ai-draft-toolbar">
                        <div className="form-group ai-draft-path-group">
                          <label className="form-label">目标路径</label>
                          <input
                            className="input"
                            value={selectedDraft.path}
                            onChange={(event) => {
                              const currentPath = selectedDraft.path;
                              const nextPath = event.target.value;
                              updateDraftContent(currentPath, (draft) => ({
                                ...draft,
                                path: nextPath,
                                needsReview: true,
                              }));
                              setSelectedDraftPath(nextPath);
                            }}
                          />
                        </div>
                        <div className="ai-draft-toolbar-actions">
                          {selectedDraft.source === 'image' && (
                            <span className="badge badge-danger">来自图片识别，需重点人工复核</span>
                          )}
                          <button
                            className="btn btn-sm btn-secondary"
                            onClick={() => void handleSaveDrafts([selectedDraft], true)}
                            disabled={savingDrafts || !selectedServiceHasConfigPatch}
                          >
                            {savingDrafts ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <ShieldCheck className="w-4 h-4" />}
                            保存并同步配置
                          </button>
                          <button
                            className="btn btn-sm btn-primary"
                            onClick={() => void handleSaveDrafts([selectedDraft])}
                            disabled={savingDrafts}
                          >
                            {savingDrafts ? <LoaderCircle className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                            保存当前
                          </button>
                        </div>
                      </div>

                      <div className="card mb-4">
                        <div className="card-body flex items-center justify-between gap-2">
                          <div className="text-sm text-gray-600">{selectedDraft.reason || 'AI 生成的模板草稿'}</div>
                          <span className="badge badge-gray">{getDraftLanguage(selectedDraft.path)}</span>
                        </div>
                      </div>

                      <div className="ai-draft-editor">
                        <Editor
                          height="100%"
                          language={getDraftLanguage(selectedDraft.path)}
                          value={selectedDraft.content}
                          onChange={(value) => updateDraftContent(selectedDraft.path, (draft) => ({
                            ...draft,
                            content: value ?? '',
                            needsReview: true,
                          }))}
                          options={{
                            minimap: { enabled: false },
                            fontSize: 13,
                            lineNumbers: 'on',
                            scrollBeyondLastLine: false,
                            wordWrap: 'off',
                            theme: 'vs',
                          }}
                        />
                      </div>
                    </>
                  ) : (
                    <div className="empty-state ai-draft-empty">
                      <Sparkles className="empty-state-icon" />
                      <p>选择一个草稿开始审核，或者先发送一条对话请求。</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <SettingsModal
        open={settingsOpen}
        settings={settings}
        saving={savingSettings}
        testing={testingSettings}
        testSucceeded={settingsTestSucceeded}
        error={settingsError}
        onClose={() => {
          setSettingsOpen(false);
          setSettingsError(null);
          setSettingsTestSucceeded(false);
        }}
        onSave={handleSaveSettings}
        onTest={handleTestSettings}
      />

      <RulesModal
        open={rulesOpen}
        content={rules}
        saving={savingRules}
        onClose={() => setRulesOpen(false)}
        onChange={setRules}
        onSave={handleSaveRules}
      />
    </div>
  );
}
