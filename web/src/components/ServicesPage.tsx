import { useState } from 'react';
import { Plus, Edit2, Trash2, Layers, Settings, Check, X, ChevronRight } from 'lucide-react';
import type { AppConfig, ServiceTopoItem, ServiceConfigItem } from '../api/config';
import { VariablesEditor } from './VariablesEditor';

type ServiceTab = 'topo' | 'config';
type ServiceConfigView = 'list' | 'detail';

interface ServicesPageProps {
  config: AppConfig;
  onChange: (config: AppConfig) => void;
}

export function ServicesPage({ config, onChange }: ServicesPageProps) {
  const [activeTab, setActiveTab] = useState<ServiceTab>('topo');

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">服务配置</h2>
          <p className="text-gray-500 mt-1">管理服务拓扑（部署位置）和服务详细配置</p>
        </div>
      </div>

      {/* 标签页切换 */}
      <div className="tabs mb-6">
        <button
          className={`tab ${activeTab === 'topo' ? 'active' : ''}`}
          onClick={() => setActiveTab('topo')}
        >
          <Layers className="w-4 h-4 inline mr-2" />
          服务拓扑
          <span className="ml-2 badge badge-gray">{Object.keys(config.serviceTop).length}</span>
        </button>
        <button
          className={`tab ${activeTab === 'config' ? 'active' : ''}`}
          onClick={() => setActiveTab('config')}
        >
          <Settings className="w-4 h-4 inline mr-2" />
          服务配置
          <span className="ml-2 badge badge-gray">{Object.keys(config.serverConfig).length}</span>
        </button>
      </div>

      {activeTab === 'topo' ? (
        <ServiceTopoTab config={config} onChange={onChange} />
      ) : (
        <ServiceConfigTab config={config} onChange={onChange} />
      )}
    </div>
  );
}

// 服务拓扑标签页
function ServiceTopoTab({ config, onChange }: ServicesPageProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingService, setEditingService] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    nodes: [] as string[],
    description: '',
    id_auto_derive: false
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const allNodes = Object.keys(config.nodes);

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = '服务名称不能为空';
    } else if (editingService !== formData.name && config.serviceTop[formData.name]) {
      newErrors.name = '服务名称已存在';
    }

    if (formData.nodes.length === 0) {
      newErrors.nodes = '至少选择一个节点';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = () => {
    if (!validate()) return;

    const newServiceTop = { ...config.serviceTop };

    if (editingService && editingService !== formData.name) {
      delete newServiceTop[editingService];
    }

    newServiceTop[formData.name] = {
      nodes: formData.nodes,
      description: formData.description,
      id_auto_derive: formData.id_auto_derive
    };

    onChange({ ...config, serviceTop: newServiceTop });
    closeModal();
  };

  const handleDelete = (name: string) => {
    if (!confirm(`确定要删除服务拓扑 "${name}" 吗？`)) return;

    const newServiceTop = { ...config.serviceTop };
    delete newServiceTop[name];
    onChange({ ...config, serviceTop: newServiceTop });
  };

  const openAddModal = () => {
    setEditingService(null);
    setFormData({ name: '', nodes: [], description: '', id_auto_derive: false });
    setErrors({});
    setIsModalOpen(true);
  };

  const openEditModal = (name: string, service: ServiceTopoItem) => {
    setEditingService(name);
    setFormData({
      name,
      nodes: service.nodes,
      description: service.description || '',
      id_auto_derive: service.id_auto_derive || false
    });
    setErrors({});
    setIsModalOpen(true);
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setEditingService(null);
    setFormData({ name: '', nodes: [], description: '', id_auto_derive: false });
    setErrors({});
  };

  const toggleNode = (node: string) => {
    const newNodes = formData.nodes.includes(node)
      ? formData.nodes.filter(n => n !== node)
      : [...formData.nodes, node];
    setFormData({ ...formData, nodes: newNodes });
  };

  const selectAllNodes = () => {
    setFormData({ ...formData, nodes: ['*'] });
  };

  const clearAllNodes = () => {
    setFormData({ ...formData, nodes: [] });
  };

  return (
    <>
      <div className="flex justify-end mb-4">
        <button className="btn btn-primary" onClick={openAddModal}>
          <Plus className="w-4 h-4" />
          添加服务拓扑
        </button>
      </div>

      <div className="card">
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>服务名称</th>
                <th>部署节点</th>
                <th>描述</th>
                <th>自增ID</th>
                <th style={{ width: '120px' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(config.serviceTop).map(([name, service]) => (
                <tr key={name}>
                  <td>
                    <div className="flex items-center gap-2">
                      <Layers className="w-4 h-4 text-gray-400" />
                      <span className="font-medium">{name}</span>
                    </div>
                  </td>
                  <td>
                    {service.nodes.includes('*') ? (
                      <span className="badge badge-green">所有节点 (*)</span>
                    ) : (
                      <div className="flex flex-wrap gap-1">
                        {service.nodes.slice(0, 4).map(node => (
                          <span key={node} className="badge badge-blue text-xs">{node}</span>
                        ))}
                        {service.nodes.length > 4 && (
                          <span className="badge badge-gray text-xs">+{service.nodes.length - 4}</span>
                        )}
                      </div>
                    )}
                  </td>
                  <td className="text-gray-600">{service.description || '-'}</td>
                  <td>
                    {service.id_auto_derive ? (
                      <span className="badge badge-blue text-xs">自动ID</span>
                    ) : (
                      <span className="text-gray-400 text-sm">-</span>
                    )}
                  </td>
                  <td>
                    <div className="flex gap-1.5">
                      <button
                        className="btn btn-sm btn-secondary"
                        onClick={() => openEditModal(name, service)}
                      >
                        <Edit2 className="w-3.5 h-3.5" />
                      </button>
                      <button
                        className="btn btn-sm btn-danger"
                        onClick={() => handleDelete(name)}
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

        {Object.keys(config.serviceTop).length === 0 && (
          <div className="empty-state">
            <Layers className="empty-state-icon" />
            <p>暂无服务拓扑配置</p>
          </div>
        )}
      </div>

      {/* 弹窗 - 全屏模式 */}
      {isModalOpen && (
        <div className="modal-fullscreen">
          <div className="modal-fullscreen-content">
            <div className="modal-header">
              <h3 className="font-bold text-gray-800 text-lg">
                {editingService ? '编辑服务拓扑' : '添加服务拓扑'}
              </h3>
              <button className="btn btn-sm btn-secondary px-4 py-2 text-base font-medium border-2 border-gray-300 hover:border-gray-400" onClick={closeModal}>关闭</button>
            </div>
            <div className="modal-body">
              <div className="form-group max-w-3xl mx-auto">
                <label className="form-label text-base font-medium">
                  服务名称 <span className="text-danger">*</span>
                </label>
                <input
                  type="text"
                  className={`input ${errors.name ? 'border-danger' : ''} text-base`}
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  placeholder="例如: zookeeper 或 hadoop/hdfs_namenode"
                  disabled={editingService !== null}
                />
                {errors.name && <p className="text-danger text-base mt-1">{errors.name}</p>}
                <p className="form-hint text-base">可以使用斜杠表示层级关系，如 doris/fe</p>
              </div>

              <div className="form-group max-w-3xl mx-auto">
                <label className="form-label text-base font-medium">
                  部署节点 <span className="text-danger">*</span>
                </label>
                <div className="flex gap-2 mb-3">
                  <button type="button" className="btn btn-sm btn-secondary text-base" onClick={selectAllNodes}>
                    所有节点
                  </button>
                  <button type="button" className="btn btn-sm btn-secondary text-base" onClick={clearAllNodes}>
                    清空
                  </button>
                </div>

                {formData.nodes.includes('*') ? (
                  <div className="p-3 bg-green-50 border border-green-200 rounded-lg flex items-center gap-2">
                    <Check className="w-4 h-4 text-green-500" />
                    <span className="text-green-700 text-base">已选择所有节点 (*)</span>
                    <button className="ml-auto text-green-600 hover:text-green-800" onClick={clearAllNodes}>
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ) : (
                  <div className="grid grid-cols-4 gap-2 max-h-60 overflow-y-auto p-3 border rounded-lg">
                    {allNodes.map(node => (
                      <label key={node} className="flex items-center gap-2 cursor-pointer text-base">
                        <input
                          type="checkbox"
                          checked={formData.nodes.includes(node)}
                          onChange={() => toggleNode(node)}
                        />
                        <span className="text-sm">{node}</span>
                      </label>
                    ))}
                  </div>
                )}
                {errors.nodes && <p className="text-danger text-base mt-1">{errors.nodes}</p>}
              </div>

              <div className="form-group max-w-3xl mx-auto">
                <label className="form-label text-base font-medium">描述</label>
                <input
                  type="text"
                  className="input text-base"
                  value={formData.description}
                  onChange={e => setFormData({ ...formData, description: e.target.value })}
                  placeholder="例如: ZooKeeper 集群"
                />
              </div>

              <div className="form-group max-w-3xl mx-auto">
                <label className="form-label text-base font-medium">自增ID配置</label>
                <label className="flex items-center gap-2 cursor-pointer text-base">
                  <input
                    type="checkbox"
                    checked={formData.id_auto_derive}
                    onChange={e => setFormData({ ...formData, id_auto_derive: e.target.checked })}
                  />
                  <span className="text-base">启用自动推导ID（需在服务配置中设置 id_field 和 id_format）</span>
                </label>
                <p className="form-hint text-base">启用后，系统会自动为每个节点实例生成唯一ID，从服务器配置的 id_format 字段读取格式模板</p>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary text-base px-6 py-2.5" onClick={closeModal}>取消</button>
              <button className="btn btn-primary text-base px-6 py-2.5" onClick={handleSubmit}>
                {editingService ? '保存' : '添加'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

// 服务配置标签页
function ServiceConfigTab({ config, onChange }: ServicesPageProps) {
  const [viewMode, setViewMode] = useState<ServiceConfigView>('list');
  const [selectedService, setSelectedService] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingConfig, setEditingConfig] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    type: '' as '' | 'global',
    description: '',
    vars: {} as Record<string, any>
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = '配置名称不能为空';
    } else if (editingConfig !== formData.name && config.serverConfig[formData.name]) {
      newErrors.name = '配置名称已存在';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = () => {
    if (!validate()) return;

    const newServerConfig = { ...config.serverConfig };

    if (editingConfig && editingConfig !== formData.name) {
      delete newServerConfig[editingConfig];
    }

    const configItem: ServiceConfigItem = {
      description: formData.description,
      vars: formData.vars || {}
    };

    if (formData.type) {
      configItem.type = formData.type;
    }

    newServerConfig[formData.name] = configItem;
    onChange({ ...config, serverConfig: newServerConfig });
    closeModal();
  };

  const handleDelete = (name: string) => {
    if (!confirm(`确定要删除服务配置 "${name}" 吗？`)) return;

    const newServerConfig = { ...config.serverConfig };
    delete newServerConfig[name];
    onChange({ ...config, serverConfig: newServerConfig });
  };

  const openAddModal = () => {
    setEditingConfig(null);
    setFormData({ name: '', type: '', description: '', vars: {} });
    setErrors({});
    setIsModalOpen(true);
  };

  const openEditModal = (name: string, cfg: ServiceConfigItem) => {
    setEditingConfig(name);
    setFormData({
      name,
      type: cfg.type || '',
      description: cfg.description || '',
      vars: cfg.vars || {}
    });
    setErrors({});
    setIsModalOpen(true);
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setEditingConfig(null);
    setFormData({ name: '', type: '', description: '', vars: {} });
    setErrors({});
  };

  const handleViewDetail = (name: string) => {
    setSelectedService(name);
    setViewMode('detail');
  };

  const handleBackToList = () => {
    setSelectedService(null);
    setViewMode('list');
  };

  // 列表视图
  if (viewMode === 'list') {
    return (
      <>
        <div className="flex justify-end mb-4">
          <button className="btn btn-primary" onClick={openAddModal}>
            <Plus className="w-4 h-4" />
            添加服务配置
          </button>
        </div>

        <div className="card">
          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th style={{ width: '25%' }}>服务名称</th>
                  <th style={{ width: '15%' }}>服务类型</th>
                  <th style={{ width: '35%' }}>描述</th>
                  <th style={{ width: '15%' }}>配置项数</th>
                  <th style={{ width: '10%' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {Object.entries(config.serverConfig).map(([name, cfg]) => (
                  <tr key={name}>
                    <td>
                      <button
                        className="flex items-center gap-2 text-primary hover:underline cursor-pointer text-base font-semibold"
                        onClick={() => handleViewDetail(name)}
                      >
                        <span>{name}</span>
                        <ChevronRight className="w-4 h-4" />
                      </button>
                    </td>
                    <td>
                      {cfg.type === 'global' ? (
                        <span className="badge badge-green text-xs">全局</span>
                      ) : (
                        <span className="badge badge-gray text-xs">普通</span>
                      )}
                    </td>
                    <td>
                      <span className="text-sm text-gray-600">{cfg.description || '-'}</span>
                    </td>
                    <td>
                      <span className="badge badge-blue text-xs">
                        {Object.keys(cfg.vars || {}).length} 项
                      </span>
                    </td>
                    <td>
                      <div className="flex gap-1.5">
                        <button
                          className="btn btn-sm btn-secondary"
                          onClick={() => openEditModal(name, cfg)}
                          title="编辑"
                        >
                          <Edit2 className="w-3.5 h-3.5" />
                        </button>
                        <button
                          className="btn btn-sm btn-danger"
                          onClick={() => handleDelete(name)}
                          title="删除"
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

          {Object.keys(config.serverConfig).length === 0 && (
            <div className="empty-state">
              <Settings className="empty-state-icon" />
              <p>暂无服务配置</p>
            </div>
          )}
        </div>

        {/* 弹窗 - 全屏模式 */}
        {isModalOpen && (
          <div className="modal-fullscreen">
            <div className="modal-fullscreen-content">
              <div className="modal-header">
                <h3 className="font-bold text-gray-800 text-lg">
                  {editingConfig ? '编辑服务配置' : '添加服务配置'}
                </h3>
                <button className="btn btn-sm btn-secondary px-4 py-2 text-base font-medium border-2 border-gray-300 hover:border-gray-400" onClick={closeModal}>关闭</button>
              </div>
              <div className="modal-body">
                <div className="form-group max-w-3xl mx-auto">
                  <label className="form-label text-base font-medium">
                    配置名称 <span className="text-danger">*</span>
                  </label>
                  <input
                    type="text"
                    className={`input ${errors.name ? 'border-danger' : ''} text-base`}
                    value={formData.name}
                    onChange={e => setFormData({ ...formData, name: e.target.value })}
                    placeholder="例如: zookeeper 或 doris/fe"
                    disabled={editingConfig !== null}
                  />
                  {errors.name && <p className="text-danger text-base mt-1">{errors.name}</p>}
                  <p className="form-hint text-base">建议与服务拓扑中的名称保持一致</p>
                </div>

                <div className="form-group max-w-3xl mx-auto">
                  <label className="form-label text-base font-medium">服务类型</label>
                  <select
                    className="select text-base"
                    value={formData.type}
                    onChange={e => setFormData({ ...formData, type: e.target.value as '' | 'global' })}
                  >
                    <option value="">普通服务</option>
                    <option value="global">全局服务（所有节点）</option>
                  </select>
                  <p className="form-hint text-base">全局服务会在所有节点上执行</p>
                </div>

                <div className="form-group max-w-3xl mx-auto">
                  <label className="form-label text-base font-medium">描述</label>
                  <input
                    type="text"
                    className="input text-base"
                    value={formData.description}
                    onChange={e => setFormData({ ...formData, description: e.target.value })}
                    placeholder="例如: ZooKeeper 详细配置"
                  />
                </div>

                <div className="form-group max-w-full mx-auto">
                  <label className="form-label text-base font-medium">变量</label>
                  <VariablesEditor
                    vars={formData.vars}
                    onChange={(newVars) => setFormData({ ...formData, vars: newVars })}
                  />
                  <p className="form-hint text-base">服务的详细配置参数，支持字符串、数字、布尔、对象和数组</p>
                </div>
              </div>
              <div className="modal-footer">
                <button className="btn btn-secondary text-base px-6 py-2.5" onClick={closeModal}>取消</button>
                <button className="btn btn-primary text-base px-6 py-2.5" onClick={handleSubmit}>
                  {editingConfig ? '保存' : '添加'}
                </button>
              </div>
            </div>
          </div>
        )}
      </>
    );
  }

  // 详情视图
  return (
    <ServiceConfigDetail
      serviceName={selectedService!}
      config={config}
      onChange={onChange}
      onBack={handleBackToList}
    />
  );
}

// 服务配置详情组件
interface ServiceConfigDetailProps {
  serviceName: string;
  config: AppConfig;
  onBack: () => void;
}

function ServiceConfigDetail({ serviceName, config, onBack }: ServiceConfigDetailProps) {
  const serviceConfig = config.serverConfig[serviceName];
  const serviceTopo = config.serviceTop[serviceName];

  if (!serviceConfig) {
    return (
      <div className="card">
        <div className="card-body text-center py-12">
          <p className="text-gray-500 text-lg">服务配置不存在</p>
          <button className="btn btn-secondary mt-4 text-base px-6 py-2" onClick={onBack}>返回列表</button>
        </div>
      </div>
    );
  }

  // 生成 YAML 格式的配置展示
  const generateYAML = () => {
    let yaml = `# ${serviceName} 配置\n`;
    yaml += `# 服务类型: ${serviceConfig.type === 'global' ? '全局服务' : '普通服务'}\n`;
    if (serviceConfig.description) {
      yaml += `# 描述: ${serviceConfig.description}\n`;
    }
    if (serviceTopo) {
      yaml += `# 部署节点: ${serviceTopo.nodes.includes('*') ? '所有节点 (*)' : serviceTopo.nodes.join(', ')}\n`;
    }
    yaml += '\n';

    // 添加配置变量
    const vars = serviceConfig.vars || {};
    const keys = Object.keys(vars);
    if (keys.length > 0) {
      yaml += `vars:\n`;
      keys.forEach(key => {
        const value = vars[key];
        if (typeof value === 'object' && value !== null) {
          yaml += `  ${key}:\n`;
          yaml += formatObjectAsYAML(value, 4);
        } else if (typeof value === 'string') {
          yaml += `  ${key}: "${value}"\n`;
        } else if (typeof value === 'boolean') {
          yaml += `  ${key}: ${value ? 'true' : 'false'}\n`;
        } else {
          yaml += `  ${key}: ${value}\n`;
        }
      });
    } else {
      yaml += `vars: {}\n`;
    }

    return yaml;
  };

  const formatObjectAsYAML = (obj: any, indent: number): string => {
    let result = '';
    const indentStr = ' '.repeat(indent);
    Object.keys(obj).forEach(key => {
      const value = obj[key];
      if (typeof value === 'object' && value !== null) {
        if (Array.isArray(value)) {
          result += `${indentStr}${key}:\n`;
          value.forEach((item: any) => {
            if (typeof item === 'object') {
              result += `${indentStr}  -\n`;
              result += formatObjectAsYAML(item, indent + 4);
            } else {
              const valStr = typeof item === 'string' ? `"${item}"` : String(item);
              result += `${indentStr}  - ${valStr}\n`;
            }
          });
        } else {
          result += `${indentStr}${key}:\n`;
          result += formatObjectAsYAML(value, indent + 2);
        }
      } else {
        const valStr = typeof value === 'string' ? `"${value}"` : String(value);
        result += `${indentStr}${key}: ${valStr}\n`;
      }
    });
    return result;
  };

  const yamlContent = generateYAML();

  return (
    <div className="space-y-6">
      {/* 面包屑导航 */}
      <div className="flex items-center gap-2 text-base">
        <button
          className="text-primary hover:underline cursor-pointer font-medium"
          onClick={onBack}
        >
          服务配置
        </button>
        <ChevronRight className="w-5 h-5 text-gray-400" />
        <span className="font-semibold text-gray-800 text-base">{serviceName}</span>
      </div>

      {/* 基本信息 */}
      <div className="card">
        <div className="card-header">
          <h3 className="font-bold text-gray-800 text-lg">基本信息</h3>
        </div>
        <div className="card-body">
          <div className="grid grid-cols-2 gap-6">
            <div>
              <label className="text-base text-gray-500 font-medium">服务名称</label>
              <p className="font-semibold text-gray-800 text-lg">{serviceName}</p>
            </div>
            <div>
              <label className="text-base text-gray-500 font-medium">服务类型</label>
              <p>
                {serviceConfig.type === 'global' ? (
                  <span className="badge badge-green text-sm">全局服务</span>
                ) : (
                  <span className="badge badge-gray text-sm">普通服务</span>
                )}
              </p>
            </div>
            <div className="col-span-2">
              <label className="text-base text-gray-500 font-medium">描述</label>
              <p className="text-gray-800 text-base">{serviceConfig.description || '-'}</p>
            </div>
          </div>

          {/* 部署节点信息 */}
          {serviceTopo && (
            <div className="mt-6 pt-6 border-t">
              <label className="text-base text-gray-500 font-medium">部署节点</label>
              <div className="mt-3">
                {serviceTopo.nodes.includes('*') ? (
                  <span className="badge badge-green text-sm">所有节点 (*)</span>
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {serviceTopo.nodes.map(node => (
                      <span key={node} className="badge badge-blue text-sm">{node}</span>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* YAML 配置展示 */}
      <div className="card">
        <div className="card-header">
          <h3 className="font-bold text-gray-800 text-lg">
            配置 YAML ({Object.keys(serviceConfig.vars || {}).length} 项)
          </h3>
        </div>
        <div className="card-body">
          <pre className="bg-gray-50 rounded-lg p-6 text-sm text-gray-800 font-mono overflow-x-auto whitespace-pre">
            {yamlContent}
          </pre>
        </div>
      </div>

      {/* 操作按钮 */}
      <div className="flex justify-end gap-3 pb-4">
        <button className="btn btn-secondary text-base px-6 py-2.5" onClick={onBack}>返回列表</button>
      </div>
    </div>
  );
}
