import { useRef, useState } from 'react';
import {
  Download,
  FileCode,
  Globe,
  KeyRound,
  LayoutDashboard,
  LogOut,
  PanelLeftClose,
  PanelLeftOpen,
  Play,
  Settings,
  SlidersHorizontal,
  Sparkles,
  UserCog,
  Wrench,
} from 'lucide-react';
import type { EditorTab } from '../types/config';

interface SidebarProps {
  activeTab: EditorTab;
  onTabChange: (tab: EditorTab) => void;
  collapsed: boolean;
  onToggleCollapse: () => void;
  canManageUsers: boolean;
  onOpenUserManagement: () => void;
  onOpenChangePassword: () => void;
  onLogout: () => void;
}

const navItems: { id: EditorTab; label: string; icon: React.ElementType }[] = [
  { id: 'overview', label: '概览', icon: LayoutDashboard },
  { id: 'global', label: '全局配置', icon: Globe },
  { id: 'config', label: '配置管理', icon: Wrench },
  { id: 'ai', label: 'AI模板', icon: Sparkles },
  { id: 'template_editor', label: '模板编辑器', icon: FileCode },
  { id: 'generate', label: '生成配置', icon: Play },
  { id: 'preview', label: 'YAML预览', icon: FileCode },
  { id: 'export', label: '导出配置', icon: Download },
];

export function Sidebar({
  activeTab,
  onTabChange,
  collapsed,
  onToggleCollapse,
  canManageUsers,
  onOpenUserManagement,
  onOpenChangePassword,
  onLogout,
}: SidebarProps) {
  const [systemMenuOpen, setSystemMenuOpen] = useState(false);
  const closeTimerRef = useRef<number | null>(null);

  const openSystemMenu = () => {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
    setSystemMenuOpen(true);
  };

  const scheduleCloseSystemMenu = () => {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current);
    }
    closeTimerRef.current = window.setTimeout(() => {
      setSystemMenuOpen(false);
      closeTimerRef.current = null;
    }, 140);
  };

  return (
    <aside className={`sidebar ${collapsed ? 'collapsed' : ''} ${systemMenuOpen && !collapsed ? 'sidebar-system-open' : ''}`}>
      <div className="sidebar-header">
        <div className="sidebar-header-top">
          <div className="brand-lockup flex items-center gap-3">
            <div className="brand-mark w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <Settings className="w-5 h-5 text-white" />
            </div>
            <div className="sidebar-brand-copy">
              <h1 className="text-lg font-semibold text-gray-800">配置生成器</h1>
              <p className="text-xs text-gray-500">大数据平台离线配置</p>
            </div>
          </div>
          <button
            className="sidebar-toggle"
            type="button"
            onClick={onToggleCollapse}
            aria-label={collapsed ? '展开侧边栏' : '收起侧边栏'}
            title={collapsed ? '展开侧边栏' : '收起侧边栏'}
          >
            {collapsed ? <PanelLeftOpen className="w-4 h-4" /> : <PanelLeftClose className="w-4 h-4" />}
          </button>
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
              title={collapsed ? item.label : undefined}
            >
              <div className="nav-item-main">
                <Icon className="w-5 h-5" />
                <span className="nav-item-label">{item.label}</span>
              </div>
              {(item.id === 'generate' || item.id === 'ai') && (
                <span className="ml-auto badge badge-blue text-xs nav-item-badge">核心</span>
              )}
            </button>
          );
        })}

        <div
          className="sidebar-system-wrap"
          onMouseEnter={openSystemMenu}
          onMouseLeave={scheduleCloseSystemMenu}
        >
          <button
            className={`nav-item w-full text-left ${systemMenuOpen ? 'active' : ''}`}
            type="button"
            title={collapsed ? '系统设置' : undefined}
            onFocus={openSystemMenu}
            onBlur={scheduleCloseSystemMenu}
          >
            <div className="nav-item-main">
              <SlidersHorizontal className="w-5 h-5" />
              <span className="nav-item-label">系统设置</span>
            </div>
          </button>

          {systemMenuOpen && !collapsed && (
            <div
              className="sidebar-system-menu"
              onMouseEnter={openSystemMenu}
              onMouseLeave={scheduleCloseSystemMenu}
            >
              <button className="sidebar-system-item" type="button" onClick={onOpenChangePassword}>
                <KeyRound className="w-4 h-4" />
                修改密码
              </button>
              {canManageUsers && (
                <button className="sidebar-system-item" type="button" onClick={onOpenUserManagement}>
                  <UserCog className="w-4 h-4" />
                  用户管理
                </button>
              )}
            </div>
          )}
        </div>

        <button
          className="nav-item nav-item-logout w-full text-left"
          type="button"
          onClick={onLogout}
          title={collapsed ? '退出系统' : undefined}
          aria-label="退出系统"
        >
          <div className="nav-item-main">
            <LogOut className="w-5 h-5" />
            <span className="nav-item-label">退出系统</span>
          </div>
        </button>
      </nav>

      <div className="sidebar-footer p-4 border-t">
        <div className="text-xs text-gray-500">
          <p>{collapsed ? 'v1.0.0' : '版本: v1.0.0'}</p>
          {!collapsed && <p className="mt-1">后端: Go + 前端: React</p>}
        </div>
      </div>
    </aside>
  );
}
