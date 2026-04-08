import { useState, useEffect, useCallback } from 'react';
import { Sidebar } from './components/Sidebar';
import { OverviewPage } from './components/OverviewPage';
import { GlobalConfigPage } from './components/GlobalConfigPage';
import { NodesPage } from './components/NodesPage';
import { ServicesPage } from './components/ServicesPage';
import { AITemplatePage } from './components/AITemplatePage';
import { PreviewPage } from './components/PreviewPage';
import { ExportPage } from './components/ExportPage';
import { GeneratePage } from './components/GeneratePage';
import './App.css';
import { Save, RefreshCw, AlertCircle, Check } from 'lucide-react';
import type { EditorTab } from './types/config';
import type { AppConfig } from './api/config';
import { fetchConfig, saveConfig, reloadConfig } from './api/config';

function App() {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [activeTab, setActiveTab] = useState<EditorTab>('overview');
  const [saveStatus, setSaveStatus] = useState<'saved' | 'saving' | 'error' | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  // 加载配置
  useEffect(() => {
    loadConfig();
  }, []);

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
  
  const loadConfig = async () => {
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
      case 'nodes':
        return <NodesPage config={config} onChange={handleConfigChange} />;
      case 'services':
        return <ServicesPage config={config} onChange={handleConfigChange} />;
      case 'ai':
        return <AITemplatePage />;
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
            <button className="btn btn-primary" onClick={loadConfig}>
              <RefreshCw className="w-4 h-4" />
              重新加载
            </button>
          </div>
        </div>
      </div>
    );
  }
  
  return (
    <div className="app-shell min-h-screen bg-gray-50">
      <Sidebar activeTab={activeTab} onTabChange={setActiveTab} />
      
      <main className="main-content">
        <header className="header">
          <div className="header-title-group">
            <span className="header-kicker">Config Studio</span>
            <h1 className="page-title text-lg font-semibold text-gray-800">
              {activeTab === 'overview' && '概览'}
              {activeTab === 'global' && '全局配置'}
              {activeTab === 'nodes' && '节点管理'}
              {activeTab === 'services' && '服务配置'}
              {activeTab === 'ai' && 'AI 模板'}
              {activeTab === 'preview' && 'YAML 预览'}
              {activeTab === 'generate' && '生成配置'}
              {activeTab === 'export' && '导出配置'}
            </h1>
          </div>
          
          <div className="header-actions flex items-center gap-4">
            {saveStatus && (
              <div className={`status-chip flex items-center gap-1 text-sm ${
                saveStatus === 'saved' ? 'text-success' : 
                saveStatus === 'error' ? 'text-danger' : 'text-gray-400'
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
            
            <button className="btn btn-sm btn-secondary header-refresh" onClick={handleReload} title="从配置文件重新加载">
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </header>
        
        <div className="page-content">
          {renderContent()}
        </div>
      </main>
    </div>
  );
}

export default App;
