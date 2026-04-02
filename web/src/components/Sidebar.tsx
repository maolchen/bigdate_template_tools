import { 
  LayoutDashboard, 
  Globe, 
  Server, 
  Layers, 
  FileCode, 
  Download,
  Settings
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
  { id: 'preview', label: 'YAML预览', icon: FileCode },
  { id: 'export', label: '导出配置', icon: Download },
];

export function Sidebar({ activeTab, onTabChange }: SidebarProps) {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
            <Settings className="w-5 h-5 text-white" />
          </div>
          <div>
            <h1 className="text-lg font-semibold text-gray-800">配置编辑器</h1>
            <p className="text-xs text-gray-500">大数据平台离线配置</p>
          </div>
        </div>
      </div>
      
      <nav className="sidebar-nav">
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <button
              key={item.id}
              className={`nav-item w-full text-left ${activeTab === item.id ? 'active' : ''}`}
              onClick={() => onTabChange(item.id)}
            >
              <Icon className="w-5 h-5" />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>
      
      <div className="p-4 border-t">
        <div className="text-xs text-gray-500">
          <p>版本: v1.0.0</p>
          <p className="mt-1">分支: web-config-editor</p>
        </div>
      </div>
    </aside>
  );
}
