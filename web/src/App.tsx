import { useState, useEffect, useCallback } from 'react';
import { Sidebar } from './components/Sidebar';
import { OverviewPage } from './components/OverviewPage';
import { GlobalConfigPage } from './components/GlobalConfigPage';
import { ConfigManagementPage } from './components/ConfigManagementPage';
import { AITemplatePage } from './components/AITemplatePage';
import { PreviewPage } from './components/PreviewPage';
import { ExportPage } from './components/ExportPage';
import { GeneratePage } from './components/GeneratePage';
import { LoginPage } from './components/LoginPage';
import { UserManagementModal } from './components/UserManagementModal';
import { ChangePasswordModal } from './components/ChangePasswordModal';
import './App.css';
import { Save, RefreshCw, AlertCircle, Check, LogOut, Shield, UserCog, KeyRound } from 'lucide-react';
import type { EditorTab } from './types/config';
import type { AppConfig } from './api/config';
import { fetchConfig, saveConfig, reloadConfig } from './api/config';
import { fetchMe, login, logout, type AuthUser } from './api/auth';

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

  const bootstrapAuth = useCallback(async () => {
    setAuthLoading(true);
    try {
      const me = await fetchMe();
      setAuthUser(me.user);
      setActiveTemplateId(me.activeTemplateId || 'config');
      await loadConfig();
    } catch {
      setAuthUser(null);
      setConfig(null);
    } finally {
      setAuthLoading(false);
    }
  }, [loadConfig]);

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

  const handleLogin = async (username: string, password: string) => {
    setLoginLoading(true);
    setAuthError(null);
    try {
      const result = await login(username, password);
      setAuthUser(result.user);
      const me = await fetchMe();
      setActiveTemplateId(me.activeTemplateId || 'config');
      await loadConfig();
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
    }
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
        return <OverviewPage config={config} onReload={handleReload} />;
      case 'global':
        return <GlobalConfigPage config={config} onChange={handleConfigChange} />;
      case 'config':
        return (
          <ConfigManagementPage
            config={config}
            onChange={handleConfigChange}
            activeTemplateId={activeTemplateId}
            onSwitchTemplate={(nextTemplateId, nextConfig) => {
              setActiveTemplateId(nextTemplateId);
              setConfig(nextConfig);
            }}
          />
        );
      case 'ai':
        return <AITemplatePage canSaveTemplate={authUser?.role === 'admin'} />;
      case 'preview':
        return <PreviewPage config={config} />;
      case 'generate':
        return <GeneratePage />;
      case 'export':
        return <ExportPage config={config} />;
      default:
        return <OverviewPage config={config} onReload={handleReload} />;
    }
  };

  const currentTitle =
    activeTab === 'overview'
      ? '概览'
      : activeTab === 'global'
        ? '全局配置'
        : activeTab === 'config'
          ? '配置管理'
          : activeTab === 'ai'
            ? 'AI模板'
            : activeTab === 'preview'
              ? 'YAML预览'
              : activeTab === 'generate'
                ? '生成配置'
                : '导出配置';

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

  return (
    <div className={`app-shell min-h-screen bg-gray-50 ${sidebarCollapsed ? 'sidebar-collapsed' : ''}`}>
      <Sidebar
        activeTab={activeTab}
        onTabChange={setActiveTab}
        collapsed={sidebarCollapsed}
        onToggleCollapse={() => setSidebarCollapsed((previous) => !previous)}
      />

      <main className="main-content">
        <header className="header">
          <div className="header-title-group">
            <span className="header-kicker">Config Studio</span>
            <h1 className="page-title text-lg font-semibold text-gray-800">{currentTitle}</h1>
            <div className="flex items-center gap-2 text-sm text-gray-600">
              <span className="badge badge-gray">用户: {authUser.username}</span>
              <span className={`badge ${authUser.role === 'admin' ? 'badge-blue' : 'badge-gray'}`}>
                {authUser.role === 'admin' ? '管理员' : '普通用户'}
              </span>
              <span className="badge badge-gray">主模板: {activeTemplateId}</span>
            </div>
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

            <button className="btn btn-sm btn-secondary" onClick={() => setChangePasswordOpen(true)} title="修改密码">
              <KeyRound className="w-4 h-4" />
            </button>
            {authUser.role === 'admin' && (
              <button className="btn btn-sm btn-secondary" onClick={() => setUserManageOpen(true)} title="用户管理">
                <UserCog className="w-4 h-4" />
              </button>
            )}
            <button className="btn btn-sm btn-secondary" onClick={handleLogout} title="退出登录">
              <LogOut className="w-4 h-4" />
            </button>
            <button className="btn btn-sm btn-secondary header-refresh" onClick={() => void handleReload()} title="重新加载配置">
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </header>

        {authUser.mustChangePassword && (
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

        <div className="page-content">{renderContent()}</div>
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
    </div>
  );
}

export default App;
