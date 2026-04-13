import { useEffect, useMemo, useState } from 'react';
import { Layers, Plus, Server, Settings, Trash2 } from 'lucide-react';
import {
  createConfigTemplate,
  deleteConfigTemplate,
  fetchConfigTemplates,
  switchConfigTemplate,
  type AppConfig,
  type ConfigTemplateEntry,
} from '../api/config';
import { NodesPage } from './NodesPage';
import { ServicesPage } from './ServicesPage';

type ConfigSubTab = 'templates' | 'nodes' | 'services';

interface ConfigManagementPageProps {
  config: AppConfig;
  onChange: (next: AppConfig) => void;
  activeTemplateId: string;
  onSwitchTemplate: (activeTemplateId: string, config: AppConfig) => void;
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'short' }).format(date);
}

export function ConfigManagementPage({
  config,
  onChange,
  activeTemplateId,
  onSwitchTemplate,
}: ConfigManagementPageProps) {
  const [activeTab, setActiveTab] = useState<ConfigSubTab>('templates');
  const [templates, setTemplates] = useState<ConfigTemplateEntry[]>([]);
  const [loadingTemplates, setLoadingTemplates] = useState(false);
  const [templateError, setTemplateError] = useState<string | null>(null);
  const [templateIdInput, setTemplateIdInput] = useState('');
  const [templateRemarkInput, setTemplateRemarkInput] = useState('');
  const [creatingTemplate, setCreatingTemplate] = useState(false);
  const [switchingTemplateId, setSwitchingTemplateId] = useState<string | null>(null);
  const [deletingTemplateId, setDeletingTemplateId] = useState<string | null>(null);
  const [syncMessage, setSyncMessage] = useState<string | null>(null);

  const activeTemplate = useMemo(
    () => templates.find((item) => item.id === activeTemplateId) ?? null,
    [activeTemplateId, templates],
  );

  const loadTemplates = async () => {
    setLoadingTemplates(true);
    setTemplateError(null);
    try {
      setTemplates(await fetchConfigTemplates());
    } catch (error) {
      setTemplateError(error instanceof Error ? error.message : '获取模板失败');
    } finally {
      setLoadingTemplates(false);
    }
  };

  useEffect(() => {
    void loadTemplates();
  }, []);

  const handleCreateTemplate = async () => {
    if (!templateIdInput.trim()) {
      setTemplateError('请输入模板ID');
      return;
    }
    setCreatingTemplate(true);
    setTemplateError(null);
    try {
      const updated = await createConfigTemplate(templateIdInput.trim(), templateRemarkInput.trim());
      setTemplates(updated);
      setTemplateIdInput('');
      setTemplateRemarkInput('');
    } catch (error) {
      setTemplateError(error instanceof Error ? error.message : '保存模板失败');
    } finally {
      setCreatingTemplate(false);
    }
  };

  const handleSwitchTemplate = async (templateId: string) => {
    setSwitchingTemplateId(templateId);
    setTemplateError(null);
    try {
      const response = await switchConfigTemplate(templateId);
      onSwitchTemplate(response.activeTemplateId, response.config);
      setSyncMessage(
        response.sync.serviceTopAdded > 0 || response.sync.serverConfigAdded > 0
          ? `已增量同步 ${response.sync.addedServices.length} 个新增服务`
          : '已切换主模板',
      );
      await loadTemplates();
    } catch (error) {
      setTemplateError(error instanceof Error ? error.message : '切换模板失败');
    } finally {
      setSwitchingTemplateId(null);
    }
  };

  const handleDeleteTemplate = async (templateId: string) => {
    if (!confirm(`确认删除模板 ${templateId} 吗？`)) return;
    setDeletingTemplateId(templateId);
    setTemplateError(null);
    try {
      setTemplates(await deleteConfigTemplate(templateId));
    } catch (error) {
      setTemplateError(error instanceof Error ? error.message : '删除模板失败');
    } finally {
      setDeletingTemplateId(null);
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">配置管理</h2>
          <p className="text-gray-500 mt-1">模板管理、节点管理、服务拓扑与服务配置统一在这里维护</p>
        </div>
      </div>

      <div className="tabs mb-6">
        <button className={`tab ${activeTab === 'templates' ? 'active' : ''}`} onClick={() => setActiveTab('templates')}>
          <Layers className="w-4 h-4 inline mr-2" />
          模板管理
          <span className="ml-2 badge badge-gray">{templates.length}</span>
        </button>
        <button className={`tab ${activeTab === 'nodes' ? 'active' : ''}`} onClick={() => setActiveTab('nodes')}>
          <Server className="w-4 h-4 inline mr-2" />
          节点管理
        </button>
        <button className={`tab ${activeTab === 'services' ? 'active' : ''}`} onClick={() => setActiveTab('services')}>
          <Settings className="w-4 h-4 inline mr-2" />
          服务配置
        </button>
      </div>

      {activeTab === 'templates' && (
        <div className="space-y-6">
          <div className="card">
            <div className="card-body">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="form-label">模板ID</label>
                  <input
                    className="input"
                    value={templateIdInput}
                    onChange={(event) => setTemplateIdInput(event.target.value)}
                    placeholder="例如: project_a"
                  />
                </div>
                <div>
                  <label className="form-label">备注</label>
                  <input
                    className="input"
                    value={templateRemarkInput}
                    onChange={(event) => setTemplateRemarkInput(event.target.value)}
                    placeholder="例如: 生产环境A"
                  />
                </div>
              </div>
              <div className="mt-4 flex items-center gap-2">
                <button className="btn btn-primary" onClick={() => void handleCreateTemplate()} disabled={creatingTemplate}>
                  <Plus className="w-4 h-4" />
                  保存当前为配置模板
                </button>
                <button className="btn btn-secondary" onClick={() => void loadTemplates()} disabled={loadingTemplates}>
                  刷新列表
                </button>
              </div>
              {templateError && <div className="text-danger text-sm mt-3">{templateError}</div>}
              {syncMessage && <div className="text-success text-sm mt-3">{syncMessage}</div>}
            </div>
          </div>

          <div className="card">
            <div className="table-container">
              <table className="table">
                <thead>
                  <tr>
                    <th>模板ID</th>
                    <th>文件名</th>
                    <th>备注</th>
                    <th>配置摘要</th>
                    <th>更新时间</th>
                    <th style={{ width: '220px' }}>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {templates.map((item) => (
                    <tr key={item.id}>
                      <td>
                        <div className="flex items-center gap-2">
                          <span className="font-medium">{item.id}</span>
                          {item.id === activeTemplateId && <span className="badge badge-green text-xs">当前主模板</span>}
                        </div>
                      </td>
                      <td>{item.fileName}</td>
                      <td>{item.remark || '-'}</td>
                      <td>{item.nodeSummary || '-'}</td>
                      <td>{formatTime(item.updatedAt)}</td>
                      <td>
                        <div className="flex items-center gap-2">
                          <button
                            className="btn btn-sm btn-secondary"
                            onClick={() => void handleSwitchTemplate(item.id)}
                            disabled={switchingTemplateId === item.id || item.id === activeTemplateId}
                          >
                            切换主模板
                          </button>
                          <button
                            className="btn btn-sm btn-danger"
                            onClick={() => void handleDeleteTemplate(item.id)}
                            disabled={deletingTemplateId === item.id || item.id === activeTemplateId || item.id === 'config'}
                            title={item.id === 'config' ? '默认模板不可删除' : item.id === activeTemplateId ? '当前主模板不可删除' : '删除模板'}
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {templates.length === 0 && (
              <div className="empty-state">
                <Layers className="empty-state-icon" />
                <p>暂无配置模板</p>
              </div>
            )}
          </div>

          <div className="card">
            <div className="card-body text-sm text-gray-600">
              <p>当前主模板: <span className="font-medium text-gray-800">{activeTemplate?.id ?? activeTemplateId}</span></p>
              <p className="mt-1">登录或切换主模板时会自动补齐主配置中新增加的服务（仅增量，不覆盖你已修改的服务）。</p>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'nodes' && <NodesPage config={config} onChange={onChange} />}
      {activeTab === 'services' && <ServicesPage config={config} onChange={onChange} />}
    </div>
  );
}

