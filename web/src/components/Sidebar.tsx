import { 
  LayoutDashboard, 
  Globe, 
  Server, 
  Layers, 
  Sparkles,
  FileCode, 
  Download,
  Settings,
  Play
} from 'lucide-react';
import type { EditorTab } from '../types/config';

interface SidebarProps {
  activeTab: EditorTab;
  onTabChange: (tab: EditorTab) => void;
}

const navItems: { id: EditorTab; label: string; icon: React.ElementType }[] = [
  { id: 'overview', label: '概览', icon: LayoutDashboard },
  { id: 'global', label: '全局配置', icon: Globe },
  { id: 'nodes', label: '节点管理', icon: Server },
  { id: 'services', label: '服务配置', icon: Layers },
  { id: 'ai', label: 'AI 模板', icon: Sparkles },
  { id: 'generate', label: '生成配置', icon: Play },
  { id: 'preview', label: 'YAML预览', icon: FileCode },
  { id: 'export', label: '导出配置', icon: Download },
];

export function Sidebar({ activeTab, onTabChange }: SidebarProps) {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="brand-lockup flex items-center gap-3">
          <div className="brand-mark w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
            <Settings className="w-5 h-5 text-white" />
          </div>
          <div className="sidebar-brand-copy">
            <h1 className="text-lg font-semibold text-gray-800">配置生成器</h1>
            <p className="text-xs text-gray-500">大数据平台离线配置</p>
          </div>
        </div>
      </div>
      
      <nav className="sidebar-nav">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = activeTab === item.id;
          return (
            <button
              key={item.id}
              className={`nav-item w-full text-left ${isActive ? 'active' : ''}`}
              onClick={() => onTabChange(item.id)}
            >
              <div className="nav-item-main">
                <Icon className="w-5 h-5" />
                <span>{item.label}</span>
              </div>
              {(item.id === 'generate' || item.id === 'ai') && (
                <span className="ml-auto badge badge-blue text-xs">核心</span>
              )}
            </button>
          );
        })}
      </nav>
      
      <div className="sidebar-footer p-4 border-t">
        <div className="text-xs text-gray-500">
          <p>版本: v1.0.0</p>
          <p className="mt-1">后端: Go + 前端: React</p>
        </div>
      </div>
    </aside>
  );
}
