import { useState, useEffect, useCallback } from 'react';
import { Sidebar } from './components/Sidebar';
import { OverviewPage } from './components/OverviewPage';
import { GlobalConfigPage } from './components/GlobalConfigPage';
import { NodesPage } from './components/NodesPage';
import { ServicesPage } from './components/ServicesPage';
import { PreviewPage } from './components/PreviewPage';
import { ExportPage } from './components/ExportPage';
import { Save } from 'lucide-react';
import type { AppConfig, EditorTab } from './types/config';
import { loadConfig, saveConfig } from './data/defaultConfig';

function App() {
  const [config, setConfig] = useState<AppConfig>(loadConfig());
  const [activeTab, setActiveTab] = useState<EditorTab>('overview');
  const [saveStatus, setSaveStatus] = useState<'saved' | 'saving' | null>(null);
  
  // 自动保存
  useEffect(() => {
    setSaveStatus('saving');
    const timer = setTimeout(() => {
      saveConfig(config);
      setSaveStatus('saved');
      
      // 2秒后清除状态
      setTimeout(() => setSaveStatus(null), 2000);
    }, 500);
    
    return () => clearTimeout(timer);
  }, [config]);
  
  const handleConfigChange = useCallback((newConfig: AppConfig) => {
    setConfig(newConfig);
  }, []);
  
  const renderContent = () => {
    switch (activeTab) {
      case 'overview':
        return <OverviewPage config={config} />;
      case 'global':
        return <GlobalConfigPage config={config} onChange={handleConfigChange} />;
      case 'nodes':
        return <NodesPage config={config} onChange={handleConfigChange} />;
      case 'services':
        return <ServicesPage config={config} onChange={handleConfigChange} />;
      case 'preview':
        return <PreviewPage config={config} />;
      case 'export':
        return <ExportPage config={config} />;
      default:
        return <OverviewPage config={config} />;
    }
  };
  
  return (
    <div className="min-h-screen bg-gray-50">
      <Sidebar activeTab={activeTab} onTabChange={setActiveTab} />
      
      <main className="main-content">
        {/* 顶部栏 */}
        <header className="header">
          <div className="flex items-center gap-4">
            <h1 className="text-lg font-semibold text-gray-800">
              {activeTab === 'overview' && '概览'}
              {activeTab === 'global' && '全局配置'}
              {activeTab === 'nodes' && '节点管理'}
              {activeTab === 'services' && '服务配置'}
              {activeTab === 'preview' && 'YAML 预览'}
              {activeTab === 'export' && '导出配置'}
            </h1>
          </div>
          
          <div className="flex items-center gap-4">
            {saveStatus && (
              <div className={`flex items-center gap-1 text-sm ${
                saveStatus === 'saved' ? 'text-success' : 'text-gray-400'
              }`}>
                <Save className="w-4 h-4" />
                <span>{saveStatus === 'saved' ? '已保存' : '保存中...'}</span>
              </div>
            )}
          </div>
        </header>
        
        {/* 页面内容 */}
        <div className="page-content">
          {renderContent()}
        </div>
      </main>
    </div>
  );
}

export default App;
