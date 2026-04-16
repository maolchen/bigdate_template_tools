import { useState, useEffect, useCallback } from 'react';
import { Sidebar } from './components/Sidebar';
import { OverviewPage } from './components/OverviewPage';
import { GlobalConfigPage } from './components/GlobalConfigPage';
import { ConfigManagementPage } from './components/ConfigManagementPage';
import { AITemplatePage } from './components/AITemplatePage';
import { PreviewPage } from './components/PreviewPage';
import { ExportPage } from './components/ExportPage';
import { GeneratePage } from './components/GeneratePage';
import { TemplateEditorPage } from './components/TemplateEditorPage';
import { LoginPage } from './components/LoginPage';
import { UserManagementModal } from './components/UserManagementModal';
import { ChangePasswordModal } from './components/ChangePasswordModal';
import './App.css';
import { Save, RefreshCw, AlertCircle, Check, Shield } from 'lucide-react';
import type { EditorTab } from './types/config';
import type { AppConfig, RootConfigVersion, ConfigSyncSummary } from './api/config';
import { fetchConfig, saveConfig, reloadConfig, fetchRootConfigVersion, fetchSyncMainPreview, syncMainConfig, syncMainDelete } from './api/config';
import { fetchMe, login, logout, type AuthUser } from './api/auth';

function isSameRootConfigVersion(a: RootConfigVersion | null, b: RootConfigVersion | null): boolean {
  if (!a || !b) return false;
  return a.mtimeUnixNano === b.mtimeUnixNano && a.size === b.size && a.sha256 === b.sha256;
}

function getRootConfigContentSignature(version: RootConfigVersion | null): string {
  if (!version) return '';
  return version.sha256 || `${version.size}:${version.mtimeUnixNano}`;
}

function isSameRootConfigContent(a: RootConfigVersion | null, b: RootConfigVersion | null): boolean {
  const left = getRootConfigContentSignature(a);
  const right = getRootConfigContentSignature(b);
  return Boolean(left && right && left === right);
}

function getIgnoredRootConfigStorageKey(username: string, templateId: string, version: RootConfigVersion): string {
  const safeTemplateId = templateId || 'config';
  return `config-generator:root-config-sync-ignored:${username}:${safeTemplateId}:${getRootConfigContentSignature(version)}`;
}

function isRootConfigVersionIgnored(username: string | undefined, templateId: string, version: RootConfigVersion | null): boolean {
  if (!username || !version) return false;
  try {
    return window.localStorage.getItem(getIgnoredRootConfigStorageKey(username, templateId, version)) === '1';
  } catch {
    return false;
  }
}

function markRootConfigVersionIgnored(username: string | undefined, templateId: string, version: RootConfigVersion | null) {
  if (!username || !version) return;
  try {
    window.localStorage.setItem(getIgnoredRootConfigStorageKey(username, templateId, version), '1');
  } catch (err) {
    console.warn('[App] failed to persist ignored root config sync version:', err);
  }
}

function buildSyncSummary(sync: {
  addedGlobalKeys: string[];
  removedGlobalKeys?: string[];
  addedServiceTopServices?: string[];
  removedServiceTopServices?: string[];
  addedServerConfigServices?: string[];
  removedServerConfigServices?: string[];
  addedServices?: string[];
  removedServices?: string[];
}): string {
  const parts: string[] = [];
  const globalKeys = Array.isArray(sync.addedGlobalKeys) ? sync.addedGlobalKeys : [];
  const removedGlobalKeys = Array.isArray(sync.removedGlobalKeys) ? sync.removedGlobalKeys : [];
  const topServices = Array.isArray(sync.addedServiceTopServices) ? sync.addedServiceTopServices : [];
  const removedTopServices = Array.isArray(sync.removedServiceTopServices) ? sync.removedServiceTopServices : [];
  const cfgServices = Array.isArray(sync.addedServerConfigServices) ? sync.addedServerConfigServices : [];
  const removedCfgServices = Array.isArray(sync.removedServerConfigServices) ? sync.removedServerConfigServices : [];

  if (globalKeys.length > 0) {
    parts.push(`全局新增: ${globalKeys.join('、')}`);
  }
  if (removedGlobalKeys.length > 0) {
    parts.push(`全局删除: ${removedGlobalKeys.join('、')}`);
  }
  if (topServices.length > 0) {
    parts.push(`serviceTop 新增: ${topServices.join('、')}`);
  }
  if (removedTopServices.length > 0) {
    parts.push(`serviceTop 删除: ${removedTopServices.join('、')}`);
  }
  if (cfgServices.length > 0) {
    parts.push(`serverConfig 新增: ${cfgServices.join('、')}`);
  }
  if (removedCfgServices.length > 0) {
    parts.push(`serverConfig 删除: ${removedCfgServices.join('、')}`);
  }
  if (parts.length === 0 && Array.isArray(sync.addedServices) && sync.addedServices.length > 0) {
    parts.push(`新增服务: ${sync.addedServices.join('、')}`);
  }
  if (parts.length === 0 && Array.isArray(sync.removedServices) && sync.removedServices.length > 0) {
    parts.push(`删除服务: ${sync.removedServices.join('、')}`);
  }
  return parts.join('；');
}

function hasSyncDelta(sync: ConfigSyncSummary | null): boolean {
  if (!sync) return false;
  return sync.globalAdded > 0
    || sync.globalRemoved > 0
    || sync.serviceTopAdded > 0
    || sync.serviceTopRemoved > 0
    || sync.serverConfigAdded > 0
    || sync.serverConfigRemoved > 0;
}

function App() {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [activeTab, setActiveTab] = useState<EditorTab>('overview');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    const stored = window.localStorage.getItem('config-generator:sidebar-collapsed');
    return stored === '1';
  });
  const [saveStatus, setSaveStatus] = useState<'saved' | 'saving' | 'error' | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [authLoading, setAuthLoading] = useState(true);
  const [loginLoading, setLoginLoading] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [authUser, setAuthUser] = useState<AuthUser | null>(null);
  const [activeTemplateId, setActiveTemplateId] = useState('config');
  const [userManageOpen, setUserManageOpen] = useState(false);
  const [changePasswordOpen, setChangePasswordOpen] = useState(false);
  const [rootConfigBaseVersion, setRootConfigBaseVersion] = useState<RootConfigVersion | null>(null);
  const [rootConfigLatestVersion, setRootConfigLatestVersion] = useState<RootConfigVersion | null>(null);
  const [rootConfigSyncLoading, setRootConfigSyncLoading] = useState(false);
  const [rootConfigDeleteSyncLoading, setRootConfigDeleteSyncLoading] = useState(false);
  const [rootConfigSyncError, setRootConfigSyncError] = useState<string | null>(null);
  const [rootConfigSyncInfo, setRootConfigSyncInfo] = useState<string | null>(null);
  const [rootConfigLastSummary, setRootConfigLastSummary] = useState<string | null>(null);
  const [rootConfigPreview, setRootConfigPreview] = useState<ConfigSyncSummary | null>(null);
  const [rootConfigPreviewLoading, setRootConfigPreviewLoading] = useState(false);
  const [rootConfigModalOpen, setRootConfigModalOpen] = useState(false);
  const [ignoredRootConfigVersion, setIgnoredRootConfigVersion] = useState<RootConfigVersion | null>(null);
  const [aiEditorMode, setAiEditorMode] = useState(false);
  const isNormalUser = authUser?.role === 'user';

  const loadConfig = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchConfig();
      setConfig(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载配置失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const initRootConfigVersion = useCallback(async () => {
    try {
      const version = await fetchRootConfigVersion();
      setRootConfigBaseVersion(version);
      setRootConfigLatestVersion(version);
      setRootConfigSyncError(null);
      setRootConfigLastSummary(null);
      setRootConfigPreview(null);
      setRootConfigModalOpen(false);
      setIgnoredRootConfigVersion(null);
    } catch (err) {
      console.warn('[App] init root config version failed:', err);
    }
  }, []);

  useEffect(() => {
    if (!authUser || authUser.role !== 'user' || !rootConfigLatestVersion) {
      setIgnoredRootConfigVersion(null);
      return;
    }

    if (isRootConfigVersionIgnored(authUser.username, activeTemplateId, rootConfigLatestVersion)) {
      setIgnoredRootConfigVersion(rootConfigLatestVersion);
      return;
    }

    setIgnoredRootConfigVersion((current) => {
      if (current && isSameRootConfigContent(current, rootConfigLatestVersion)) {
        return null;
      }
      return current;
    });
  }, [activeTemplateId, authUser, rootConfigLatestVersion]);

  const rootConfigOutdated = isNormalUser && Boolean(rootConfigBaseVersion && rootConfigLatestVersion)
    && !isSameRootConfigVersion(rootConfigBaseVersion, rootConfigLatestVersion);
  const rootConfigIgnored = Boolean(rootConfigLatestVersion && ignoredRootConfigVersion)
    && isSameRootConfigContent(rootConfigLatestVersion, ignoredRootConfigVersion);

  const openRootConfigDiffModal = useCallback(async (options?: { forceOpen?: boolean; latestVersion?: RootConfigVersion | null; refreshWhenNoDelta?: boolean }) => {
    if (!authUser || authUser.role !== 'user') return;

    const forceOpen = Boolean(options?.forceOpen);
    setRootConfigPreviewLoading(true);
    setRootConfigSyncError(null);
    try {
      const result = await fetchSyncMainPreview();
      const hasDelta = hasSyncDelta(result.sync);
      setRootConfigPreview(result.sync);
      setRootConfigLastSummary(buildSyncSummary(result.sync) || null);
      setRootConfigModalOpen(forceOpen || hasDelta);
      if (!hasDelta && !forceOpen) {
        if (options?.refreshWhenNoDelta) {
          await loadConfig();
        }
        if (options?.latestVersion) {
          setRootConfigBaseVersion(options.latestVersion);
          setRootConfigLatestVersion(options.latestVersion);
        }
      }
    } catch (err) {
      setRootConfigSyncError(err instanceof Error ? err.message : '获取主配置差异失败');
      setRootConfigModalOpen(true);
    } finally {
      setRootConfigPreviewLoading(false);
    }
  }, [authUser, loadConfig]);

  const bootstrapAuth = useCallback(async () => {
    setAuthLoading(true);
    try {
      const me = await fetchMe();
      setAuthUser(me.user);
      setActiveTemplateId(me.activeTemplateId || 'config');
      await loadConfig();
      await initRootConfigVersion();
    } catch {
      setAuthUser(null);
      setConfig(null);
      setRootConfigBaseVersion(null);
      setRootConfigLatestVersion(null);
      setRootConfigLastSummary(null);
      setRootConfigPreview(null);
      setRootConfigModalOpen(false);
      setIgnoredRootConfigVersion(null);
    } finally {
      setAuthLoading(false);
    }
  }, [initRootConfigVersion, loadConfig]);

  useEffect(() => {
    void bootstrapAuth();
  }, [bootstrapAuth]);

  useEffect(() => {
    const handleNavigate = (event: Event) => {
      const customEvent = event as CustomEvent<EditorTab>;
      if (customEvent.detail) {
        setActiveTab(customEvent.detail);
      }
    };

    window.addEventListener('navigate', handleNavigate as EventListener);
    return () => {
      window.removeEventListener('navigate', handleNavigate as EventListener);
    };
  }, []);

  useEffect(() => {
    window.localStorage.setItem('config-generator:sidebar-collapsed', sidebarCollapsed ? '1' : '0');
  }, [sidebarCollapsed]);

  useEffect(() => {
    if (!authUser || authUser.role !== 'user') return;

    let stopped = false;
    const poll = async () => {
      try {
        const latest = await fetchRootConfigVersion();
        if (stopped) return;
        setRootConfigLatestVersion(latest);
      } catch (err) {
        if (!stopped) {
          console.warn('[App] root config version polling failed:', err);
        }
      }
    };

    void poll();
    const timer = window.setInterval(() => {
      void poll();
    }, 30000);

    return () => {
      stopped = true;
      window.clearInterval(timer);
    };
  }, [authUser]);

  useEffect(() => {
    if (!rootConfigSyncInfo) return;
    const timer = window.setTimeout(() => setRootConfigSyncInfo(null), 5000);
    return () => window.clearTimeout(timer);
  }, [rootConfigSyncInfo]);

  useEffect(() => {
    if (!rootConfigOutdated || rootConfigIgnored || !authUser || authUser.role !== 'user') {
      return;
    }

    void openRootConfigDiffModal({
      latestVersion: rootConfigLatestVersion,
      refreshWhenNoDelta: true,
    });
  }, [authUser, openRootConfigDiffModal, rootConfigIgnored, rootConfigOutdated, rootConfigLatestVersion]);

  useEffect(() => {
    const isStandaloneTemplateEditor = new URLSearchParams(window.location.search).get('view') === 'template-editor';
    const allowWindowScroll = Boolean(authUser) && activeTab !== 'ai' && !isStandaloneTemplateEditor;
    const root = document.getElementById('root');
    const targets = [document.documentElement, document.body].filter(Boolean) as HTMLElement[];
    const previous = targets.map((target) => ({
      target,
      height: target.style.height,
      overflowX: target.style.overflowX,
      overflowY: target.style.overflowY,
    }));
    const previousRoot = root
      ? {
        height: root.style.height,
        overflowX: root.style.overflowX,
        overflowY: root.style.overflowY,
      }
      : null;

    if (allowWindowScroll) {
      for (const target of targets) {
        target.style.height = 'auto';
        target.style.overflowX = 'hidden';
        target.style.overflowY = 'auto';
      }
      if (root) {
        root.style.height = 'auto';
        root.style.overflowX = 'hidden';
        root.style.overflowY = 'visible';
      }
    }

    return () => {
      for (const item of previous) {
        item.target.style.height = item.height;
        item.target.style.overflowX = item.overflowX;
        item.target.style.overflowY = item.overflowY;
      }
      if (root && previousRoot) {
        root.style.height = previousRoot.height;
        root.style.overflowX = previousRoot.overflowX;
        root.style.overflowY = previousRoot.overflowY;
      }
    };
  }, [activeTab, aiEditorMode, authUser]);

  const handleSyncMainFromRoot = async () => {
    setRootConfigSyncLoading(true);
    setRootConfigSyncError(null);
    try {
      const result = await syncMainConfig();
      setConfig(result.config);
      setRootConfigBaseVersion(result.version);
      setRootConfigLatestVersion(result.version);
      setIgnoredRootConfigVersion(null);
      const hasDelta = result.sync.globalAdded > 0 || result.sync.serviceTopAdded > 0 || result.sync.serverConfigAdded > 0;
      const hasRemoved = (result.sync.globalRemoved || 0) > 0
        || (result.sync.serviceTopRemoved || 0) > 0
        || (result.sync.serverConfigRemoved || 0) > 0;
      const summary = buildSyncSummary(result.sync);
      setRootConfigLastSummary(summary || null);
      setRootConfigPreview(result.sync);
      setRootConfigModalOpen(hasSyncDelta(result.sync));
      setRootConfigSyncInfo(
        (hasDelta || hasRemoved)
          ? `主配置已同步。${summary || '检测到增量变更。'}`
          : '主配置已是最新，无需同步',
      );
    } catch (err) {
      setRootConfigSyncError(err instanceof Error ? err.message : '同步主配置失败');
    } finally {
      setRootConfigSyncLoading(false);
    }
  };

  const handleDangerDeleteSync = async (confirmBeforeSync = true) => {
    if (confirmBeforeSync) {
      const confirmed = window.confirm('将按主配置删除当前用户配置中的缺失项（全局键、serviceTop、serverConfig）。此操作有风险，是否继续？');
      if (!confirmed) return;
    }
    setRootConfigDeleteSyncLoading(true);
    setRootConfigSyncError(null);
    try {
      const result = await syncMainDelete();
      setConfig(result.config);
      setRootConfigBaseVersion(result.version);
      setRootConfigLatestVersion(result.version);
      setIgnoredRootConfigVersion(null);
      const summary = buildSyncSummary(result.sync);
      setRootConfigLastSummary(summary || null);
      setRootConfigSyncInfo(summary ? `删除清理已完成。${summary}` : '删除清理已完成，无需变更');
      setRootConfigPreview(result.sync);
      setRootConfigModalOpen(false);
    } catch (err) {
      setRootConfigSyncError(err instanceof Error ? err.message : '清理删除项失败');
    } finally {
      setRootConfigDeleteSyncLoading(false);
    }
  };

  const handleLogin = async (username: string, password: string) => {
    setLoginLoading(true);
    setAuthError(null);
    try {
      const result = await login(username, password);
      setAuthUser(result.user);
      const me = await fetchMe();
      setActiveTemplateId(me.activeTemplateId || 'config');
      await loadConfig();
      await initRootConfigVersion();
    } catch (err) {
      setAuthError(err instanceof Error ? err.message : '登录失败');
    } finally {
      setLoginLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      await logout();
    } finally {
      setAuthUser(null);
      setConfig(null);
      setAuthError(null);
      setActiveTemplateId('config');
      setActiveTab('overview');
      setRootConfigBaseVersion(null);
      setRootConfigLatestVersion(null);
      setRootConfigSyncError(null);
      setRootConfigSyncInfo(null);
      setRootConfigLastSummary(null);
      setRootConfigPreview(null);
      setRootConfigModalOpen(false);
      setIgnoredRootConfigVersion(null);
    }
  };

  const handleIgnoreRootConfigSync = () => {
    if (authUser?.role === 'user' && rootConfigLatestVersion) {
      markRootConfigVersionIgnored(authUser.username, activeTemplateId, rootConfigLatestVersion);
      setIgnoredRootConfigVersion(rootConfigLatestVersion);
    }
    setRootConfigModalOpen(false);
  };

  const handleConfigChange = useCallback(async (newConfig: AppConfig) => {
    setConfig(newConfig);
    setSaveStatus('saving');

    try {
      await saveConfig(newConfig);
      setSaveStatus('saved');
      setTimeout(() => setSaveStatus(null), 2000);
    } catch (err) {
      setSaveStatus('error');
      console.error('保存配置失败:', err);
    }
  }, []);

  const handleReload = async () => {
    try {
      setError(null);
      const result = await reloadConfig();
      if (result.config) {
        setConfig(result.config);
      } else {
        await loadConfig();
        return;
      }
      setSaveStatus('saved');
      setTimeout(() => setSaveStatus(null), 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : '重新加载配置失败');
    }
  };

  const renderContent = () => {
    if (!config) {
      return (
        <div className="flex items-center justify-center h-full">
          <p className="text-gray-500">加载配置中...</p>
        </div>
      );
    }

    switch (activeTab) {
      case 'overview':
        return <OverviewPage config={config} onReload={handleReload} hidePageTitle />;
      case 'global':
        return <GlobalConfigPage config={config} onChange={handleConfigChange} hidePageTitle />;
      case 'config':
        return (
          <ConfigManagementPage
            config={config}
            onChange={handleConfigChange}
            activeTemplateId={activeTemplateId}
            hidePageTitle
            onSwitchTemplate={(nextTemplateId, nextConfig) => {
              setActiveTemplateId(nextTemplateId);
              setConfig(nextConfig);
              if (currentAuthUser?.role === 'user') {
                void openRootConfigDiffModal();
              }
            }}
          />
        );
      case 'ai':
        return (
          <AITemplatePage
            canSaveTemplate={currentAuthUser?.role === 'admin'}
            onEditorModeChange={setAiEditorMode}
            username={currentAuthUser?.username}
            activeTemplateId={activeTemplateId}
          />
        );
      case 'template_editor':
        return <TemplateEditorPage />;
      case 'preview':
        return <PreviewPage config={config} hidePageTitle />;
      case 'generate':
        return <GeneratePage hidePageTitle />;
      case 'export':
        return <ExportPage config={config} hidePageTitle />;
      default:
        return <OverviewPage config={config} onReload={handleReload} hidePageTitle />;
    }
  };

  const headerMeta = activeTab === 'overview'
    ? { title: '配置概览', subtitle: '查看当前配置的整体状态和统计信息' }
    : activeTab === 'global'
      ? { title: '全局配置管理', subtitle: '管理全局配置项，包括目录路径、用户、JDK 等' }
      : activeTab === 'config'
        ? { title: '配置管理', subtitle: '模板管理、节点管理、服务拓扑与服务配置统一维护' }
        : activeTab === 'ai'
          ? { title: 'AI模板', subtitle: '对话生成模板草稿，确认后再写入 templates/' }
          : activeTab === 'template_editor'
            ? { title: '模板编辑器', subtitle: '手动编辑 templates 目录（管理员可写，普通用户只读）' }
          : activeTab === 'preview'
            ? { title: 'YAML 预览', subtitle: '预览当前配置生成的 YAML 内容' }
            : activeTab === 'generate'
              ? { title: '生成配置', subtitle: '根据配置生成部署脚本与配置文件' }
              : { title: '导出配置', subtitle: '导出配置与生成产物，供交付使用' };

  if (authLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <RefreshCw className="w-8 h-8 text-primary animate-spin mx-auto mb-4" />
          <p className="text-gray-500">正在检查登录状态...</p>
        </div>
      </div>
    );
  }

  if (!authUser) {
    return <LoginPage loading={loginLoading} error={authError} onLogin={handleLogin} />;
  }

  if (new URLSearchParams(window.location.search).get('view') === 'template-editor') {
    return <TemplateEditorPage standalone />;
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <RefreshCw className="w-8 h-8 text-primary animate-spin mx-auto mb-4" />
          <p className="text-gray-500">加载配置中...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="card max-w-md">
          <div className="card-body text-center">
            <AlertCircle className="w-12 h-12 text-danger mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-gray-800 mb-2">加载失败</h2>
            <p className="text-gray-500 mb-4">{error}</p>
            <button className="btn btn-primary" onClick={() => void loadConfig()}>
              <RefreshCw className="w-4 h-4" />
              重新加载
            </button>
          </div>
        </div>
      </div>
    );
  }

  const currentAuthUser = authUser;
  const useWindowScrollLayout = activeTab !== 'ai';
  const currentSyncSummary = rootConfigPreview ? buildSyncSummary(rootConfigPreview) : rootConfigLastSummary;
  const rootConfigHasDelta = hasSyncDelta(rootConfigPreview);
  const rootConfigHasRemovals = Boolean(
    rootConfigPreview
    && (rootConfigPreview.globalRemoved > 0
      || rootConfigPreview.serviceTopRemoved > 0
      || rootConfigPreview.serverConfigRemoved > 0),
  );

  return (
    <div className={`app-shell min-h-screen bg-gray-50 ${sidebarCollapsed ? 'sidebar-collapsed' : ''} ${useWindowScrollLayout ? 'app-shell-window-scroll' : 'app-shell-panel-scroll'} ${activeTab === 'ai' && aiEditorMode ? 'app-shell-ai-editor' : ''}`}>
      <Sidebar
        activeTab={activeTab}
        onTabChange={(tab) => {
          if (tab === 'template_editor') {
            const target = `${window.location.origin}${window.location.pathname}?view=template-editor`;
            window.open(target, '_blank', 'noopener,noreferrer');
            return;
          }
          setActiveTab(tab);
        }}
        collapsed={sidebarCollapsed}
        onToggleCollapse={() => setSidebarCollapsed((previous) => !previous)}
        canManageUsers={currentAuthUser.role === 'admin'}
        onOpenUserManagement={() => setUserManageOpen(true)}
        onOpenChangePassword={() => setChangePasswordOpen(true)}
        onLogout={handleLogout}
      />

      <main className={`main-content ${useWindowScrollLayout ? 'main-content-window-scroll' : ''}`}>
        <header className="header header-compact">
          <div className={`header-title-group ${activeTab === 'ai' ? 'header-title-group-ai' : ''}`}>
            <div className="page-header-title-row">
              <h1 className="page-title text-lg font-semibold text-gray-800">{headerMeta.title}</h1>
              <div className="page-header-badges text-sm text-gray-600">
                <span className="badge badge-gray">用户: {currentAuthUser.username}</span>
                {activeTab !== 'ai' && (
                  <span className={`badge ${currentAuthUser.role === 'admin' ? 'badge-blue' : 'badge-gray'}`}>
                    {currentAuthUser.role === 'admin' ? '管理员' : '普通用户'}
                  </span>
                )}
                <span className="badge badge-gray">主模板: {activeTemplateId}</span>
              </div>
            </div>
            <p className="header-subtitle">{headerMeta.subtitle}</p>
          </div>

          <div className="header-actions flex items-center gap-2">
            {saveStatus && (
              <div className={`status-chip flex items-center gap-1 text-sm ${
                saveStatus === 'saved' ? 'text-success' : saveStatus === 'error' ? 'text-danger' : 'text-gray-400'
              }`}>
                {saveStatus === 'saved' ? (
                  <>
                    <Check className="w-4 h-4" />
                    <span>已保存</span>
                  </>
                ) : saveStatus === 'error' ? (
                  <>
                    <AlertCircle className="w-4 h-4" />
                    <span>保存失败</span>
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4" />
                    <span>保存中...</span>
                  </>
                )}
              </div>
            )}
            <button className="btn btn-sm btn-secondary header-refresh" onClick={() => void handleReload()} title="重新加载配置">
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
          {activeTab === 'ai' && <div id="ai-header-extra" className="ai-header-extra" />}
        </header>

        {currentAuthUser.mustChangePassword && (
          <div className="card mb-4" style={{ borderColor: '#f59e0b', borderWidth: '1px' }}>
            <div className="card-body flex items-center justify-between gap-4">
              <div className="flex items-center gap-2 text-amber-700">
                <Shield className="w-4 h-4" />
                <span>首次登录必须修改密码后再继续使用。</span>
              </div>
              <button className="btn btn-sm btn-secondary" onClick={() => setChangePasswordOpen(true)}>立即修改</button>
            </div>
          </div>
        )}

        <div className={`page-content ${activeTab === 'ai' ? 'page-content-ai' : ''} ${useWindowScrollLayout ? 'page-content-window-scroll' : ''}`}>{renderContent()}</div>
      </main>

      <UserManagementModal open={userManageOpen} onClose={() => setUserManageOpen(false)} />
      <ChangePasswordModal
        open={changePasswordOpen || authUser.mustChangePassword}
        force={authUser.mustChangePassword}
        onClose={() => setChangePasswordOpen(false)}
        onChanged={async () => {
          setChangePasswordOpen(false);
          const me = await fetchMe();
          setAuthUser(me.user);
          setActiveTemplateId(me.activeTemplateId || 'config');
        }}
      />

      {isNormalUser && rootConfigModalOpen && (
        <div className="sync-modal-backdrop" role="presentation">
          <div className="sync-modal card" role="dialog" aria-modal="true" aria-labelledby="root-config-sync-title">
            <div className="card-body sync-modal-body">
              <div className="sync-modal-header">
                <div>
                  <h2 id="root-config-sync-title" className="sync-modal-title">检测到主配置已更新</h2>
                  <p className="sync-modal-subtitle">新增项仍按现有增量同步逻辑处理；删除项默认不动，由你在这里决定是否同步删除。</p>
                </div>
                <button
                  type="button"
                  className="btn btn-sm btn-secondary"
                  onClick={handleIgnoreRootConfigSync}
                  disabled={rootConfigSyncLoading || rootConfigDeleteSyncLoading}
                >
                  关闭
                </button>
              </div>

              {rootConfigPreviewLoading ? (
                <div className="sync-modal-loading">
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  <span>正在分析主配置差异...</span>
                </div>
              ) : rootConfigSyncError ? (
                <div className="sync-modal-error">
                  <AlertCircle className="w-4 h-4" />
                  <span>{rootConfigSyncError}</span>
                </div>
              ) : (
                <>
                  <div className="sync-modal-summary">
                    <div className="sync-modal-summary-title">变更摘要</div>
                    <div className="sync-modal-summary-text">
                      {currentSyncSummary || (rootConfigHasDelta
                        ? '检测到主配置差异，但暂无可展示摘要。'
                        : '主配置文件版本已变化，但未检测到新增/删除项；可能是已有配置值发生修改，系统不会自动覆盖用户已有配置值。')}
                    </div>
                  </div>

                  <div className="sync-modal-grid">
                    <div className="sync-modal-section">
                      <div className="sync-modal-section-title">全局配置</div>
                      <div className="sync-modal-section-row">
                        <span>新增</span>
                        <span>{rootConfigPreview?.addedGlobalKeys?.length ? rootConfigPreview.addedGlobalKeys.join('、') : '无'}</span>
                      </div>
                      <div className="sync-modal-section-row danger">
                        <span>删除</span>
                        <span>{rootConfigPreview?.removedGlobalKeys?.length ? rootConfigPreview.removedGlobalKeys.join('、') : '无'}</span>
                      </div>
                    </div>

                    <div className="sync-modal-section">
                      <div className="sync-modal-section-title">serviceTop</div>
                      <div className="sync-modal-section-row">
                        <span>新增服务</span>
                        <span>{rootConfigPreview?.addedServiceTopServices?.length ? rootConfigPreview.addedServiceTopServices.join('、') : '无'}</span>
                      </div>
                      <div className="sync-modal-section-row danger">
                        <span>删除服务</span>
                        <span>{rootConfigPreview?.removedServiceTopServices?.length ? rootConfigPreview.removedServiceTopServices.join('、') : '无'}</span>
                      </div>
                    </div>

                    <div className="sync-modal-section">
                      <div className="sync-modal-section-title">serverConfig</div>
                      <div className="sync-modal-section-row">
                        <span>新增服务</span>
                        <span>{rootConfigPreview?.addedServerConfigServices?.length ? rootConfigPreview.addedServerConfigServices.join('、') : '无'}</span>
                      </div>
                      <div className="sync-modal-section-row danger">
                        <span>删除服务</span>
                        <span>{rootConfigPreview?.removedServerConfigServices?.length ? rootConfigPreview.removedServerConfigServices.join('、') : '无'}</span>
                      </div>
                    </div>
                  </div>

                  <div className="sync-modal-actions">
                    <button
                      className="btn btn-sm btn-secondary"
                      onClick={handleIgnoreRootConfigSync}
                      disabled={rootConfigSyncLoading || rootConfigDeleteSyncLoading}
                    >
                      保持现状
                    </button>
                    {rootConfigHasRemovals && (
                      <button
                        className="btn btn-sm btn-danger"
                        onClick={() => void handleDangerDeleteSync(false)}
                        disabled={rootConfigDeleteSyncLoading || rootConfigSyncLoading}
                      >
                        <AlertCircle className={`w-4 h-4 ${rootConfigDeleteSyncLoading ? 'animate-spin' : ''}`} />
                        {rootConfigDeleteSyncLoading ? '同步删除中...' : '同步删除'}
                      </button>
                    )}
                    <button
                      className="btn btn-sm btn-primary"
                      onClick={() => void handleSyncMainFromRoot()}
                      disabled={rootConfigSyncLoading || rootConfigDeleteSyncLoading}
                    >
                      <RefreshCw className={`w-4 h-4 ${rootConfigSyncLoading ? 'animate-spin' : ''}`} />
                      {rootConfigSyncLoading ? '同步中...' : '同步新增'}
                    </button>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
