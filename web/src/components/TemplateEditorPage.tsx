import { useEffect, useMemo, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import {
  FilePlus,
  FileText,
  Folder,
  FolderOpen,
  FolderPlus,
  Moon,
  RefreshCw,
  Save,
  Sun,
  Trash2,
} from 'lucide-react';
import {
  createTemplateEditorItem,
  deleteTemplateEditorItem,
  fetchTemplateEditorFile,
  fetchTemplateEditorTree,
  saveTemplateEditorFile,
  type TemplateEditorTreeNode,
} from '../api/config';

interface TemplateEditorPageProps {
  standalone?: boolean;
}

interface TreeContextMenuState {
  path: string;
  type: 'file' | 'dir' | 'root';
  x: number;
  y: number;
}

function getLanguageFromPath(path: string): string {
  const lower = path.toLowerCase();
  if (lower.endsWith('.sh.tmpl')) return 'shell';
  if (lower.endsWith('.yaml.tmpl') || lower.endsWith('.yml.tmpl')) return 'yaml';
  if (lower.endsWith('.json.tmpl')) return 'json';
  if (lower.endsWith('.xml.tmpl')) return 'xml';
  if (lower.endsWith('.md.tmpl')) return 'markdown';
  if (lower.endsWith('.conf.tmpl') || lower.endsWith('.properties.tmpl')) return 'ini';
  return 'plaintext';
}

function collectDirPaths(nodes: TemplateEditorTreeNode[]): string[] {
  const result: string[] = [];
  for (const node of nodes) {
    if (node.type !== 'dir') continue;
    result.push(node.path);
    if (node.children?.length) {
      result.push(...collectDirPaths(node.children));
    }
  }
  return result;
}

export function TemplateEditorPage({ standalone = false }: TemplateEditorPageProps) {
  const [loading, setLoading] = useState(true);
  const [tree, setTree] = useState<TemplateEditorTreeNode[]>([]);
  const [readonly, setReadonly] = useState(true);
  const [openDirs, setOpenDirs] = useState<Record<string, boolean>>({});
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [fileUpdatedAt, setFileUpdatedAt] = useState('');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [info, setInfo] = useState<string | null>(null);
  const [theme, setTheme] = useState<'dark' | 'light'>(() => (
    window.localStorage.getItem('config-generator:template-editor-theme') === 'light' ? 'light' : 'dark'
  ));
  const [contextMenu, setContextMenu] = useState<TreeContextMenuState | null>(null);

  const shellRef = useRef<HTMLDivElement | null>(null);
  const sidebarRef = useRef<HTMLElement | null>(null);
  const hasSelection = Boolean(selectedPath);
  const language = useMemo(() => getLanguageFromPath(selectedPath || ''), [selectedPath]);

  const closeContextMenu = () => setContextMenu(null);

  const loadTree = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchTemplateEditorTree();
      const nextNodes = result.nodes || [];
      setTree(nextNodes);
      setReadonly(Boolean(result.readonly));
      setOpenDirs((prev) => {
        const next = { ...prev };
        for (const path of collectDirPaths(nextNodes)) {
          if (typeof next[path] === 'undefined') {
            next[path] = true;
          }
        }
        return next;
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载模板目录失败');
    } finally {
      setLoading(false);
    }
  };

  const loadFile = async (path: string) => {
    setError(null);
    setInfo(null);
    try {
      const file = await fetchTemplateEditorFile(path);
      setSelectedPath(file.path);
      setFileContent(file.content || '');
      setFileUpdatedAt(file.updatedAt || '');
      setDirty(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载模板文件失败');
    }
  };

  useEffect(() => {
    void loadTree();
  }, []);

  useEffect(() => {
    if (!standalone) return;
    const oldTitle = document.title;
    document.title = '模板编辑器';
    return () => {
      document.title = oldTitle;
    };
  }, [standalone]);

  useEffect(() => {
    window.localStorage.setItem('config-generator:template-editor-theme', theme);
  }, [theme]);

  useEffect(() => {
    if (!info) return;
    const timer = window.setTimeout(() => setInfo(null), 2000);
    return () => window.clearTimeout(timer);
  }, [info]);

  useEffect(() => {
    const handleWindowClick = () => closeContextMenu();
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') closeContextMenu();
    };
    window.addEventListener('click', handleWindowClick);
    window.addEventListener('contextmenu', handleWindowClick);
    window.addEventListener('keydown', handleEscape);
    return () => {
      window.removeEventListener('click', handleWindowClick);
      window.removeEventListener('contextmenu', handleWindowClick);
      window.removeEventListener('keydown', handleEscape);
    };
  }, []);

  const toggleDir = (path: string) => {
    setOpenDirs((prev) => ({ ...prev, [path]: !prev[path] }));
  };

  const expandAllDirs = () => {
    const next: Record<string, boolean> = {};
    for (const path of collectDirPaths(tree)) {
      next[path] = true;
    }
    setOpenDirs(next);
  };

  const collapseAllDirs = () => {
    setOpenDirs({});
  };

  const createItem = async (parentPath: string, type: 'file' | 'dir') => {
    if (readonly) return;
    const name = window.prompt(type === 'dir' ? '请输入目录名称' : '请输入文件名称（建议以 .tmpl 结尾）');
    if (!name) return;

    const trimmed = name.trim().replace(/\\/g, '/').replace(/^\/+|\/+$/g, '');
    if (!trimmed) return;

    const nextPath = parentPath ? `${parentPath}/${trimmed}` : trimmed;
    setError(null);
    setInfo(null);
    try {
      await createTemplateEditorItem(nextPath, type, type === 'file' ? '{{/* new template */}}\n' : '');
      setOpenDirs((prev) => ({
        ...prev,
        ...(parentPath ? { [parentPath]: true } : {}),
        ...(type === 'dir' ? { [nextPath]: true } : {}),
      }));
      await loadTree();
      if (type === 'file') {
        await loadFile(nextPath);
      }
      setInfo(type === 'dir' ? '目录已创建' : '文件已创建');
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败');
    }
  };

  const deleteItem = async (path: string, type: 'file' | 'dir') => {
    if (readonly) return;
    const confirmed = window.confirm(
      `确定删除${type === 'dir' ? '目录' : '文件'} "${path}" 吗？${type === 'dir' ? '目录下所有内容都会被删除。' : ''}`,
    );
    if (!confirmed) return;

    setError(null);
    setInfo(null);
    try {
      await deleteTemplateEditorItem(path);
      if (selectedPath === path || (type === 'dir' && selectedPath?.startsWith(`${path}/`))) {
        setSelectedPath(null);
        setFileContent('');
        setFileUpdatedAt('');
        setDirty(false);
      }
      await loadTree();
      setInfo(type === 'dir' ? '目录已删除' : '文件已删除');
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除失败');
    }
  };

  const openContextMenu = (event: React.MouseEvent, item: { path: string; type: 'file' | 'dir' | 'root' }) => {
    event.preventDefault();
    event.stopPropagation();
    const menuWidth = 188;
    const menuHeight = 220;
    const padding = 8;
    const rawX = event.clientX;
    const rawY = event.clientY;
    const width = window.innerWidth;
    const height = window.innerHeight;
    setContextMenu({
      path: item.path,
      type: item.type,
      x: Math.max(padding, Math.min(rawX, width - menuWidth - padding)),
      y: Math.max(padding, Math.min(rawY, height - menuHeight - padding)),
    });
  };

  const renderTree = (nodes: TemplateEditorTreeNode[], depth = 0) => (
    <div>
      {nodes.map((node) => {
        const isDir = node.type === 'dir';
        const opened = Boolean(openDirs[node.path]);
        return (
          <div key={node.path}>
            <button
              type="button"
              className={`template-editor-tree-item ${selectedPath === node.path ? 'active' : ''}`}
              style={{ paddingLeft: `${12 + depth * 14}px` }}
              onContextMenu={(event) => openContextMenu(event, { path: node.path, type: node.type })}
              onClick={() => {
                closeContextMenu();
                if (isDir) {
                  toggleDir(node.path);
                } else {
                  void loadFile(node.path);
                }
              }}
            >
              {isDir ? (
                opened ? <FolderOpen className="w-4 h-4" /> : <Folder className="w-4 h-4" />
              ) : (
                <FileText className="w-4 h-4" />
              )}
              <span>{node.name}</span>
              {!readonly && (
                <span className="template-editor-tree-actions" onClick={(event) => event.stopPropagation()}>
                  {isDir && (
                    <>
                      <button
                        type="button"
                        className="template-editor-tree-action"
                        title="新增文件"
                        onClick={() => void createItem(node.path, 'file')}
                      >
                        <FilePlus className="w-3 h-3" />
                      </button>
                      <button
                        type="button"
                        className="template-editor-tree-action"
                        title="新增目录"
                        onClick={() => void createItem(node.path, 'dir')}
                      >
                        <FolderPlus className="w-3 h-3" />
                      </button>
                    </>
                  )}
                  <button
                    type="button"
                    className="template-editor-tree-action danger"
                    title="删除"
                    onClick={() => void deleteItem(node.path, node.type)}
                  >
                    <Trash2 className="w-3 h-3" />
                  </button>
                </span>
              )}
            </button>
            {isDir && opened && node.children && node.children.length > 0 && renderTree(node.children, depth + 1)}
          </div>
        );
      })}
    </div>
  );

  const handleSave = async () => {
    if (!selectedPath || readonly || !dirty) return;
    setSaving(true);
    setError(null);
    setInfo(null);
    try {
      const file = await saveTemplateEditorFile(selectedPath, fileContent);
      setFileUpdatedAt(file.updatedAt || '');
      setDirty(false);
      setInfo('模板已保存');
      void loadTree();
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleContextAction = async (action: 'newFile' | 'newDir' | 'delete' | 'expand' | 'collapse') => {
    if (!contextMenu) return;
    closeContextMenu();
    switch (action) {
      case 'newFile':
        await createItem(contextMenu.type === 'file' ? '' : contextMenu.path, 'file');
        break;
      case 'newDir':
        await createItem(contextMenu.type === 'file' ? '' : contextMenu.path, 'dir');
        break;
      case 'delete':
        if (contextMenu.type !== 'root') {
          await deleteItem(contextMenu.path, contextMenu.type);
        }
        break;
      case 'expand':
        if (contextMenu.type === 'root') {
          expandAllDirs();
        } else if (contextMenu.type === 'dir') {
          setOpenDirs((prev) => ({ ...prev, [contextMenu.path]: true }));
        }
        break;
      case 'collapse':
        if (contextMenu.type === 'root') {
          collapseAllDirs();
        } else if (contextMenu.type === 'dir') {
          setOpenDirs((prev) => ({ ...prev, [contextMenu.path]: false }));
        }
        break;
      default:
        break;
    }
  };

  return (
    <div
      ref={shellRef}
      className={`template-editor-shell ${standalone ? 'standalone' : ''} ${theme === 'light' ? 'template-editor-light' : 'template-editor-dark'}`}
    >
      <aside ref={sidebarRef} className="template-editor-sidebar">
        <div className="template-editor-sidebar-header">
          <div className="template-editor-title" onContextMenu={(event) => openContextMenu(event, { path: '', type: 'root' })}>
            模板目录
          </div>
          <div className="template-editor-sidebar-actions">
            {!readonly && (
              <>
                <button className="template-editor-icon-btn" onClick={() => void createItem('', 'file')} title="在根目录新增文件">
                  <FilePlus className="w-4 h-4" />
                </button>
                <button className="template-editor-icon-btn" onClick={() => void createItem('', 'dir')} title="在根目录新增目录">
                  <FolderPlus className="w-4 h-4" />
                </button>
              </>
            )}
            <button className="template-editor-icon-btn" onClick={expandAllDirs} title="全部展开目录">
              <FolderOpen className="w-4 h-4" />
            </button>
            <button className="template-editor-icon-btn" onClick={collapseAllDirs} title="全部收起目录">
              <Folder className="w-4 h-4" />
            </button>
            <button className="template-editor-icon-btn" onClick={() => setTheme((value) => (value === 'dark' ? 'light' : 'dark'))} title="切换黑白模式">
              {theme === 'dark' ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
            </button>
            <button className="template-editor-icon-btn" onClick={() => void loadTree()} title="刷新目录">
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>
        <div className="template-editor-tree" onContextMenu={(event) => {
          if (event.target === event.currentTarget) {
            openContextMenu(event, { path: '', type: 'root' });
          }
        }}
        >
          {loading ? <div className="template-editor-empty">加载中...</div> : renderTree(tree)}
        </div>
      </aside>

      <section className="template-editor-main">
        <div className="template-editor-main-header">
          <div className="template-editor-main-meta">
            <div className="template-editor-file-path">{selectedPath || '请选择左侧模板文件'}</div>
            <div className="template-editor-file-sub">
              {readonly ? '只读模式（普通用户）' : '可编辑（管理员）'}
              {fileUpdatedAt ? ` · 更新于 ${new Date(fileUpdatedAt).toLocaleString('zh-CN')}` : ''}
              {dirty ? ' · 未保存' : ''}
            </div>
          </div>
          <button
            className="btn btn-sm btn-primary"
            onClick={() => void handleSave()}
            disabled={readonly || !hasSelection || !dirty || saving}
          >
            <Save className={`w-4 h-4 ${saving ? 'animate-spin' : ''}`} />
            {saving ? '保存中...' : '保存'}
          </button>
        </div>

        {(error || info) && (
          <div className="template-editor-toast-layer">
            {error && <div className="template-editor-alert error">{error}</div>}
            {info && <div className="template-editor-alert success">{info}</div>}
          </div>
        )}

        {contextMenu && (
          <div
            className="template-editor-context-menu"
            style={{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px`, position: 'fixed' }}
            onClick={(event) => event.stopPropagation()}
          >
            {!readonly && (
              <>
                <button type="button" className="template-editor-context-item" onClick={() => void handleContextAction('newFile')}>
                  <FilePlus className="w-4 h-4" />
                  新增文件
                </button>
                <button type="button" className="template-editor-context-item" onClick={() => void handleContextAction('newDir')}>
                  <FolderPlus className="w-4 h-4" />
                  新增目录
                </button>
              </>
            )}
            {(contextMenu.type === 'root' || contextMenu.type === 'dir') && (
              <>
                <button type="button" className="template-editor-context-item" onClick={() => void handleContextAction('expand')}>
                  <FolderOpen className="w-4 h-4" />
                  {contextMenu.type === 'root' ? '全部展开' : '展开目录'}
                </button>
                <button type="button" className="template-editor-context-item" onClick={() => void handleContextAction('collapse')}>
                  <Folder className="w-4 h-4" />
                  {contextMenu.type === 'root' ? '全部收起' : '收起目录'}
                </button>
              </>
            )}
            {!readonly && contextMenu.type !== 'root' && (
              <button type="button" className="template-editor-context-item danger" onClick={() => void handleContextAction('delete')}>
                <Trash2 className="w-4 h-4" />
                删除
              </button>
            )}
          </div>
        )}

        <div className="template-editor-editor-wrap">
          {hasSelection ? (
            <Editor
              height="100%"
              language={language}
              value={fileContent}
              onChange={(value) => {
                setFileContent(value ?? '');
                setDirty(true);
              }}
              options={{
                readOnly: readonly,
                minimap: { enabled: true },
                fontSize: 14,
                lineNumbers: 'on',
                scrollBeyondLastLine: true,
                automaticLayout: true,
                wordWrap: 'off',
              }}
              theme={theme === 'dark' ? 'vs-dark' : 'vs'}
              loading={<div className="template-editor-empty">编辑器加载中...</div>}
            />
          ) : (
            <div className="template-editor-empty">请选择左侧文件开始查看或编辑</div>
          )}
        </div>
      </section>
    </div>
  );
}
