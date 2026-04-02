import { useState } from 'react';
import { Plus, Edit2, Trash2, Server, AlertCircle } from 'lucide-react';
import type { AppConfig, NodeInfo } from '../api/config';

interface NodesPageProps {
  config: AppConfig;
  onChange: (config: AppConfig) => void;
}

interface NodeFormData {
  name: string;
  ip: string;
  hostname: string;
}

export function NodesPage({ config, onChange }: NodesPageProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingNode, setEditingNode] = useState<string | null>(null);
  const [formData, setFormData] = useState<NodeFormData>({ name: '', ip: '', hostname: '' });
  const [errors, setErrors] = useState<Partial<NodeFormData>>({});
  
  const validate = (): boolean => {
    const newErrors: Partial<NodeFormData> = {};
    
    if (!formData.name.trim()) {
      newErrors.name = '节点名称不能为空';
    } else if (!/^[a-zA-Z][a-zA-Z0-9_-]*$/.test(formData.name)) {
      newErrors.name = '节点名称必须以字母开头，只能包含字母、数字、下划线和连字符';
    } else if (editingNode !== formData.name && config.nodes[formData.name]) {
      newErrors.name = '节点名称已存在';
    }
    
    if (!formData.ip.trim()) {
      newErrors.ip = 'IP 地址不能为空';
    } else if (!/^(\d{1,3}\.){3}\d{1,3}$/.test(formData.ip)) {
      newErrors.ip = 'IP 地址格式不正确';
    }
    
    if (!formData.hostname.trim()) {
      newErrors.hostname = '主机名不能为空';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };
  
  const handleSubmit = () => {
    if (!validate()) return;
    
    const newNodes = { ...config.nodes };
    
    // 如果是编辑且名称改变了，删除旧的
    if (editingNode && editingNode !== formData.name) {
      delete newNodes[editingNode];
      
      // 更新服务拓扑中的节点引用
      const newServiceTop = { ...config.serviceTop };
      Object.keys(newServiceTop).forEach(serviceName => {
        const service = newServiceTop[serviceName];
        const nodeIndex = service.nodes.indexOf(editingNode);
        if (nodeIndex !== -1) {
          service.nodes[nodeIndex] = formData.name;
        }
      });
      
      onChange({
        ...config,
        nodes: {
          ...newNodes,
          [formData.name]: { ip: formData.ip, hostname: formData.hostname }
        },
        serviceTop: newServiceTop
      });
    } else {
      onChange({
        ...config,
        nodes: {
          ...newNodes,
          [formData.name]: { ip: formData.ip, hostname: formData.hostname }
        }
      });
    }
    
    closeModal();
  };
  
  const handleDelete = (name: string) => {
    if (!confirm(`确定要删除节点 "${name}" 吗？\n注意：这将同时从所有服务拓扑中移除该节点。`)) {
      return;
    }
    
    const newNodes = { ...config.nodes };
    delete newNodes[name];
    
    // 从服务拓扑中移除该节点
    const newServiceTop = { ...config.serviceTop };
    Object.keys(newServiceTop).forEach(serviceName => {
      const service = newServiceTop[serviceName];
      service.nodes = service.nodes.filter(n => n !== name);
    });
    
    onChange({
      ...config,
      nodes: newNodes,
      serviceTop: newServiceTop
    });
  };
  
  const openAddModal = () => {
    setEditingNode(null);
    setFormData({ name: '', ip: '', hostname: '' });
    setErrors({});
    setIsModalOpen(true);
  };
  
  const openEditModal = (name: string, node: NodeInfo) => {
    setEditingNode(name);
    setFormData({ name, ip: node.ip, hostname: node.hostname });
    setErrors({});
    setIsModalOpen(true);
  };
  
  const closeModal = () => {
    setIsModalOpen(false);
    setEditingNode(null);
    setFormData({ name: '', ip: '', hostname: '' });
    setErrors({});
  };
  
  // 检查节点是否被服务引用
  const getNodeReferences = (nodeName: string): string[] => {
    const refs: string[] = [];
    Object.entries(config.serviceTop).forEach(([serviceName, service]) => {
      if (service.nodes.includes(nodeName)) {
        refs.push(serviceName);
      }
    });
    return refs;
  };
  
  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">节点管理</h2>
          <p className="text-gray-500 mt-1">管理集群中的所有节点（IP、主机名）</p>
        </div>
        <button className="btn btn-primary" onClick={openAddModal}>
          <Plus className="w-4 h-4" />
          添加节点
        </button>
      </div>
      
      {/* 节点列表 */}
      <div className="card">
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>节点名称</th>
                <th>IP 地址</th>
                <th>主机名</th>
                <th>被引用服务</th>
                <th style={{ width: '120px' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(config.nodes).map(([name, node]) => {
                const refs = getNodeReferences(name);
                return (
                  <tr key={name}>
                    <td>
                      <div className="flex items-center gap-2">
                        <Server className="w-4 h-4 text-gray-400" />
                        <span className="font-medium">{name}</span>
                      </div>
                    </td>
                    <td>
                      <code className="bg-gray-100 px-2 py-1 rounded text-sm">{node.ip}</code>
                    </td>
                    <td>{node.hostname}</td>
                    <td>
                      {refs.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {refs.slice(0, 3).map(ref => (
                            <span key={ref} className="badge badge-gray text-xs">{ref}</span>
                          ))}
                          {refs.length > 3 && (
                            <span className="badge badge-gray text-xs">+{refs.length - 3}</span>
                          )}
                        </div>
                      ) : (
                        <span className="text-gray-400 text-sm">未使用</span>
                      )}
                    </td>
                    <td>
                      <div className="flex gap-1">
                        <button 
                          className="btn btn-sm btn-secondary"
                          onClick={() => openEditModal(name, node)}
                        >
                          <Edit2 className="w-3 h-3" />
                        </button>
                        <button 
                          className="btn btn-sm btn-danger"
                          onClick={() => handleDelete(name)}
                          disabled={refs.length > 0}
                          title={refs.length > 0 ? '该节点正在被服务引用，无法删除' : ''}
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        
        {Object.keys(config.nodes).length === 0 && (
          <div className="empty-state">
            <Server className="empty-state-icon" />
            <p>暂无节点配置</p>
            <button className="btn btn-primary mt-4" onClick={openAddModal}>
              <Plus className="w-4 h-4 mr-2" />
              添加第一个节点
            </button>
          </div>
        )}
      </div>
      
      {/* 提示信息 */}
      <div className="card mt-6">
        <div className="card-body">
          <div className="flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-primary mt-0.5" />
            <div>
              <h4 className="font-medium text-gray-800">节点使用说明</h4>
              <ul className="text-sm text-gray-600 mt-2 space-y-1 list-disc list-inside">
                <li>节点名称在配置中作为唯一标识，建议使用有意义的名称（如 node1, master1, worker1）</li>
                <li>IP 地址用于节点间通信和 SSH 连接</li>
                <li>主机名应与实际系统主机名一致</li>
                <li>被服务引用的节点无法直接删除，需要先从服务拓扑中移除</li>
                <li>使用 <code>*</code> 表示全局服务，将部署到所有节点</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
      
      {/* 添加/编辑弹窗 */}
      {isModalOpen && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="font-semibold text-gray-800">
                {editingNode ? '编辑节点' : '添加节点'}
              </h3>
              <button className="text-gray-400 hover:text-gray-600" onClick={closeModal}>×</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">
                  节点名称 <span className="text-danger">*</span>
                </label>
                <input
                  type="text"
                  className={`input ${errors.name ? 'border-danger' : ''}`}
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  placeholder="例如: node1"
                  disabled={editingNode !== null}
                />
                {errors.name && <p className="text-danger text-sm mt-1">{errors.name}</p>}
                <p className="form-hint">必须以字母开头，只能包含字母、数字、下划线和连字符</p>
              </div>
              
              <div className="form-group">
                <label className="form-label">
                  IP 地址 <span className="text-danger">*</span>
                </label>
                <input
                  type="text"
                  className={`input ${errors.ip ? 'border-danger' : ''}`}
                  value={formData.ip}
                  onChange={e => setFormData({ ...formData, ip: e.target.value })}
                  placeholder="例如: 192.168.1.101"
                />
                {errors.ip && <p className="text-danger text-sm mt-1">{errors.ip}</p>}
              </div>
              
              <div className="form-group">
                <label className="form-label">
                  主机名 <span className="text-danger">*</span>
                </label>
                <input
                  type="text"
                  className={`input ${errors.hostname ? 'border-danger' : ''}`}
                  value={formData.hostname}
                  onChange={e => setFormData({ ...formData, hostname: e.target.value })}
                  placeholder="例如: bigdata-node1"
                />
                {errors.hostname && <p className="text-danger text-sm mt-1">{errors.hostname}</p>}
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={closeModal}>取消</button>
              <button className="btn btn-primary" onClick={handleSubmit}>
                {editingNode ? '保存' : '添加'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
