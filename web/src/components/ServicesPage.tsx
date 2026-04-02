import { useState } from 'react';
import { Plus, Edit2, Trash2, Layers, Settings, Check, X } from 'lucide-react';
import type { AppConfig, ServiceTopoItem, ServiceConfigItem } from '../types/config';

type ServiceTab = 'topo' | 'config';

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
    vars: '{}'
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
    
    try {
      JSON.parse(formData.vars);
    } catch {
      newErrors.vars = '变量必须是有效的 JSON 格式';
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
      vars: JSON.parse(formData.vars)
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
    setFormData({ name: '', nodes: [], description: '', vars: '{}' });
    setErrors({});
    setIsModalOpen(true);
  };
  
  const openEditModal = (name: string, service: ServiceTopoItem) => {
    setEditingService(name);
    setFormData({
      name,
      nodes: service.nodes,
      description: service.description || '',
      vars: JSON.stringify(service.vars || {}, null, 2)
    });
    setErrors({});
    setIsModalOpen(true);
  };
  
  const closeModal = () => {
    setIsModalOpen(false);
    setEditingService(null);
    setFormData({ name: '', nodes: [], description: '', vars: '{}' });
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
                <th>变量</th>
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
                    {service.vars && Object.keys(service.vars).length > 0 ? (
                      <span className="badge badge-gray text-xs">
                        {Object.keys(service.vars).length} 个变量
                      </span>
                    ) : (
                      <span className="text-gray-400 text-sm">-</span>
                    )}
                  </td>
                  <td>
                    <div className="flex gap-1">
                      <button 
                        className="btn btn-sm btn-secondary"
                        onClick={() => openEditModal(name, service)}
                      >
                        <Edit2 className="w-3 h-3" />
                      </button>
                      <button 
                        className="btn btn-sm btn-danger"
                        onClick={() => handleDelete(name)}
                      >
                        <Trash2 className="w-3 h-3" />
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
      
      {/* 弹窗 */}
      {isModalOpen && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: '700px' }}>
            <div className="modal-header">
              <h3 className="font-semibold text-gray-800">
                {editingService ? '编辑服务拓扑' : '添加服务拓扑'}
              </h3>
              <button className="text-gray-400 hover:text-gray-600" onClick={closeModal}>×</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">
                  服务名称 <span className="text-danger">*</span>
                </label>
                <input
                  type="text"
                  className={`input ${errors.name ? 'border-danger' : ''}`}
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  placeholder="例如: zookeeper 或 hadoop/hdfs_namenode"
                  disabled={editingService !== null}
                />
                {errors.name && <p className="text-danger text-sm mt-1">{errors.name}</p>}
                <p className="form-hint">可以使用斜杠表示层级关系，如 doris/fe</p>
              </div>
              
              <div className="form-group">
                <label className="form-label">
                  部署节点 <span className="text-danger">*</span>
                </label>
                <div className="flex gap-2 mb-3">
                  <button type="button" className="btn btn-sm btn-secondary" onClick={selectAllNodes}>
                    所有节点
                  </button>
                  <button type="button" className="btn btn-sm btn-secondary" onClick={clearAllNodes}>
                    清空
                  </button>
                </div>
                
                {formData.nodes.includes('*') ? (
                  <div className="p-3 bg-green-50 border border-green-200 rounded-lg flex items-center gap-2">
                    <Check className="w-4 h-4 text-green-500" />
                    <span className="text-green-700">已选择所有节点 (*)</span>
                    <button className="ml-auto text-green-600 hover:text-green-800" onClick={clearAllNodes}>
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ) : (
                  <div className="grid grid-cols-3 gap-2 max-h-40 overflow-y-auto p-3 border rounded-lg">
                    {allNodes.map(node => (
                      <label key={node} className="flex items-center gap-2 cursor-pointer">
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
                {errors.nodes && <p className="text-danger text-sm mt-1">{errors.nodes}</p>}
              </div>
              
              <div className="form-group">
                <label className="form-label">描述</label>
                <input
                  type="text"
                  className="input"
                  value={formData.description}
                  onChange={e => setFormData({ ...formData, description: e.target.value })}
                  placeholder="例如: ZooKeeper 集群"
                />
              </div>
              
              <div className="form-group">
                <label className="form-label">变量 (JSON)</label>
                <textarea
                  className={`input font-mono text-sm ${errors.vars ? 'border-danger' : ''}`}
                  value={formData.vars}
                  onChange={e => setFormData({ ...formData, vars: e.target.value })}
                  rows={6}
                  placeholder='{"version": "3.7.1", "port": 2181}'
                />
                {errors.vars && <p className="text-danger text-sm mt-1">{errors.vars}</p>}
                <p className="form-hint">服务的自定义变量，以 JSON 格式输入</p>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={closeModal}>取消</button>
              <button className="btn btn-primary" onClick={handleSubmit}>
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
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingConfig, setEditingConfig] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    type: '' as '' | 'global',
    description: '',
    vars: '{}'
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  
  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};
    
    if (!formData.name.trim()) {
      newErrors.name = '配置名称不能为空';
    } else if (editingConfig !== formData.name && config.serverConfig[formData.name]) {
      newErrors.name = '配置名称已存在';
    }
    
    try {
      JSON.parse(formData.vars);
    } catch {
      newErrors.vars = '变量必须是有效的 JSON 格式';
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
      vars: JSON.parse(formData.vars)
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
    setFormData({ name: '', type: '', description: '', vars: '{}' });
    setErrors({});
    setIsModalOpen(true);
  };
  
  const openEditModal = (name: string, cfg: ServiceConfigItem) => {
    setEditingConfig(name);
    setFormData({
      name,
      type: cfg.type || '',
      description: cfg.description || '',
      vars: JSON.stringify(cfg.vars || {}, null, 2)
    });
    setErrors({});
    setIsModalOpen(true);
  };
  
  const closeModal = () => {
    setIsModalOpen(false);
    setEditingConfig(null);
    setFormData({ name: '', type: '', description: '', vars: '{}' });
    setErrors({});
  };
  
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
                <th>配置名称</th>
                <th>类型</th>
                <th>描述</th>
                <th>变量数</th>
                <th style={{ width: '120px' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(config.serverConfig).map(([name, cfg]) => (
                <tr key={name}>
                  <td>
                    <div className="flex items-center gap-2">
                      <Settings className="w-4 h-4 text-gray-400" />
                      <span className="font-medium">{name}</span>
                    </div>
                  </td>
                  <td>
                    {cfg.type === 'global' ? (
                      <span className="badge badge-green">全局服务</span>
                    ) : (
                      <span className="badge badge-gray">普通</span>
                    )}
                  </td>
                  <td className="text-gray-600">{cfg.description || '-'}</td>
                  <td>
                    {cfg.vars ? (
                      <span className="badge badge-blue text-xs">
                        {Object.keys(cfg.vars).length} 个
                      </span>
                    ) : (
                      <span className="text-gray-400 text-sm">-</span>
                    )}
                  </td>
                  <td>
                    <div className="flex gap-1">
                      <button 
                        className="btn btn-sm btn-secondary"
                        onClick={() => openEditModal(name, cfg)}
                      >
                        <Edit2 className="w-3 h-3" />
                      </button>
                      <button 
                        className="btn btn-sm btn-danger"
                        onClick={() => handleDelete(name)}
                      >
                        <Trash2 className="w-3 h-3" />
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
      
      {/* 弹窗 */}
      {isModalOpen && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="font-semibold text-gray-800">
                {editingConfig ? '编辑服务配置' : '添加服务配置'}
              </h3>
              <button className="text-gray-400 hover:text-gray-600" onClick={closeModal}>×</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">
                  配置名称 <span className="text-danger">*</span>
                </label>
                <input
                  type="text"
                  className={`input ${errors.name ? 'border-danger' : ''}`}
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  placeholder="例如: zookeeper 或 doris/fe"
                  disabled={editingConfig !== null}
                />
                {errors.name && <p className="text-danger text-sm mt-1">{errors.name}</p>}
                <p className="form-hint">建议与服务拓扑中的名称保持一致</p>
              </div>
              
              <div className="form-group">
                <label className="form-label">服务类型</label>
                <select 
                  className="select"
                  value={formData.type}
                  onChange={e => setFormData({ ...formData, type: e.target.value as '' | 'global' })}
                >
                  <option value="">普通服务</option>
                  <option value="global">全局服务（所有节点）</option>
                </select>
                <p className="form-hint">全局服务会在所有节点上执行</p>
              </div>
              
              <div className="form-group">
                <label className="form-label">描述</label>
                <input
                  type="text"
                  className="input"
                  value={formData.description}
                  onChange={e => setFormData({ ...formData, description: e.target.value })}
                  placeholder="例如: ZooKeeper 详细配置"
                />
              </div>
              
              <div className="form-group">
                <label className="form-label">变量 (JSON)</label>
                <textarea
                  className={`input font-mono text-sm ${errors.vars ? 'border-danger' : ''}`}
                  value={formData.vars}
                  onChange={e => setFormData({ ...formData, vars: e.target.value })}
                  rows={8}
                  placeholder='{"version": "3.7.1", "dataDir": "/data/zookeeper"}'
                />
                {errors.vars && <p className="text-danger text-sm mt-1">{errors.vars}</p>}
                <p className="form-hint">服务的详细配置参数，以 JSON 格式输入</p>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={closeModal}>取消</button>
              <button className="btn btn-primary" onClick={handleSubmit}>
                {editingConfig ? '保存' : '添加'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
