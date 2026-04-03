import { useState, useEffect } from 'react';
import { Plus, Trash2, Edit2, Globe, AlertCircle } from 'lucide-react';
import type { AppConfig } from '../api/config';
import {
  getFieldDescription,
  getCategories
} from '../lib/global-fields';

interface GlobalPageProps {
  config: AppConfig;
  onChange: (config: AppConfig) => void;
}

interface ConfigItem {
  key: string;
  value: any;
  description: string;
  category: string;
  type: 'string' | 'number' | 'boolean';
}

export function GlobalPage({ config, onChange }: GlobalPageProps) {
  const [configItems, setConfigItems] = useState<ConfigItem[]>([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<ConfigItem | null>(null);
  const [formData, setFormData] = useState<ConfigItem>({
    key: '',
    value: '',
    description: '',
    category: '其他',
    type: 'string'
  });
  const [errors, setErrors] = useState<Partial<ConfigItem>>({});

  // 初始化配置项列表
  useEffect(() => {
    const items: ConfigItem[] = [];
    Object.entries(config.global).forEach(([key, value]) => {
      const fieldDef = getFieldDescription(key);
      items.push({
        key,
        value,
        description: fieldDef?.description || '',
        category: fieldDef?.category || '其他',
        type: fieldDef?.type || 'string'
      });
    });
    setConfigItems(items);
  }, [config]);

  const handleAdd = () => {
    setEditingItem(null);
    setFormData({
      key: '',
      value: '',
      description: '',
      category: '其他',
      type: 'string'
    });
    setErrors({});
    setIsModalOpen(true);
  };

  const handleEdit = (item: ConfigItem) => {
    setEditingItem(item);
    setFormData({ ...item });
    setErrors({});
    setIsModalOpen(true);
  };

  const handleDelete = (key: string) => {
    if (!confirm(`确定要删除配置项 "${key}" 吗？`)) {
      return;
    }

    const newGlobal = { ...config.global };
    delete newGlobal[key];

    onChange({
      ...config,
      global: newGlobal
    });
  };

  const handleSubmit = () => {
    const newErrors: Partial<ConfigItem> = {};

    if (!formData.key.trim()) {
      newErrors.key = '配置项名称不能为空';
    } else if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(formData.key)) {
      newErrors.key = '配置项名称只能包含字母、数字和下划线，且必须以字母或下划线开头';
    } else if (!editingItem && config.global[formData.key]) {
      newErrors.key = '配置项名称已存在';
    }

    if (formData.value === '' || formData.value === null || formData.value === undefined) {
      newErrors.value = '配置值不能为空';
    }

    setErrors(newErrors);
    if (Object.keys(newErrors).length > 0) return;

    const newGlobal = { ...config.global };

    if (formData.type === 'number') {
      newGlobal[formData.key] = Number(formData.value);
    } else if (formData.type === 'boolean') {
      newGlobal[formData.key] = formData.value === 'true';
    } else {
      newGlobal[formData.key] = formData.value;
    }

    onChange({
      ...config,
      global: newGlobal
    });

    setIsModalOpen(false);
  };

  const handleClose = () => {
    setIsModalOpen(false);
    setEditingItem(null);
  };

  // 按分类分组
  const categories = getCategories();
  const groupedItems = categories.reduce((acc, category) => {
    acc[category] = configItems.filter(item => item.category === category);
    return acc;
  }, {} as Record<string, ConfigItem[]>);

  // 其他分类
  const otherItems = configItems.filter(item => !categories.includes(item.category));
  if (otherItems.length > 0) {
    groupedItems['其他'] = otherItems;
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">全局配置管理</h2>
          <p className="text-gray-500 mt-1">管理全局配置项，包括目录路径、用户、JDK 等</p>
        </div>
        <button className="btn btn-primary" onClick={handleAdd}>
          <Plus className="w-4 h-4" />
          添加配置项
        </button>
      </div>

      {/* 配置项列表 */}
      <div className="space-y-6">
        {Object.entries(groupedItems).map(([category, items]) => (
          <div key={category} className="card">
            <div className="card-header">
              <div className="flex items-center gap-2">
                <Globe className="w-5 h-5 text-primary" />
                <span className="font-semibold text-gray-800">{category}</span>
              </div>
              <span className="badge badge-gray">{items.length} 项</span>
            </div>
            <div className="card-body">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {items.map((item) => {
                  const fieldDef = getFieldDescription(item.key);
                  const isRequired = fieldDef?.required || false;
                  const displayName = fieldDef?.name || item.key;

                  return (
                    <div key={item.key} className="space-y-1.5">
                      <label className="flex items-center gap-1 text-sm font-medium text-gray-700">
                        <span>{displayName} ({item.key})</span>
                        {isRequired && <span className="text-danger">*</span>}
                      </label>
                      <div className="flex items-center gap-2">
                        <input
                          type="text"
                          className="input flex-1"
                          value={typeof item.value === 'string' ? item.value : String(item.value)}
                          readOnly
                          placeholder={item.description || '未配置'}
                        />
                        <button
                          className="btn btn-sm btn-secondary"
                          onClick={() => handleEdit(item)}
                          title="编辑"
                        >
                          <Edit2 className="w-4 h-4" />
                        </button>
                        <button
                          className="btn btn-sm btn-danger"
                          onClick={() => handleDelete(item.key)}
                          disabled={isRequired}
                          title={isRequired ? '必填项，不能删除' : '删除'}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                      {item.description && (
                        <p className="text-xs text-gray-500">{item.description}</p>
                      )}
                    </div>
                  );
                })}
              </div>
              {items.length === 0 && (
                <div className="text-center text-gray-400 py-8">暂无配置项</div>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* 提示信息 */}
      <div className="card mt-6">
        <div className="card-body">
          <div className="flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-primary mt-0.5" />
            <div>
              <h4 className="font-medium text-gray-800">配置说明</h4>
              <ul className="text-sm text-gray-600 mt-2 space-y-1 list-disc list-inside">
                <li>带 * 号的配置项为必填项，不能删除</li>
                <li>可以添加自定义配置项，建议使用有意义的名称</li>
                <li>配置值类型会自动转换（字符串、数字、布尔值）</li>
                <li>鼠标悬停在图标上可查看配置项说明</li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      {/* 添加/编辑弹窗 */}
      {isModalOpen && (
        <div className="modal-overlay" onClick={handleClose}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="font-semibold text-gray-800">
                {editingItem ? '编辑配置项' : '添加配置项'}
              </h3>
              <button className="text-gray-400 hover:text-gray-600" onClick={handleClose}>
                ×
              </button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">
                  配置项名称 <span className="text-danger">*</span>
                </label>
                <input
                  type="text"
                  className={`input ${errors.key ? 'border-danger' : ''}`}
                  value={formData.key}
                  onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                  placeholder="例如: custom_dir"
                  disabled={!!editingItem}
                />
                {errors.key && <p className="text-danger text-sm mt-1">{errors.key}</p>}
              </div>

              <div className="form-group">
                <label className="form-label">配置值 <span className="text-danger">*</span></label>
                {formData.type === 'boolean' ? (
                  <select
                    className="input"
                    value={formData.value}
                    onChange={(e) => setFormData({ ...formData, value: e.target.value })}
                  >
                    <option value="true">true</option>
                    <option value="false">false</option>
                  </select>
                ) : (
                  <input
                    type={formData.type === 'number' ? 'number' : 'text'}
                    className={`input ${errors.value ? 'border-danger' : ''}`}
                    value={formData.value}
                    onChange={(e) => setFormData({ ...formData, value: e.target.value })}
                    placeholder={formData.type === 'number' ? '123' : '配置值'}
                  />
                )}
                {errors.value && <p className="text-danger text-sm mt-1">{errors.value}</p>}
              </div>

              <div className="form-group">
                <label className="form-label">数据类型</label>
                <select
                  className="input"
                  value={formData.type}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      type: e.target.value as 'string' | 'number' | 'boolean',
                      value: ''
                    })
                  }
                >
                  <option value="string">字符串</option>
                  <option value="number">数字</option>
                  <option value="boolean">布尔值</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">说明</label>
                <input
                  type="text"
                  className="input"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="配置项说明（可选）"
                />
              </div>

              <div className="form-group">
                <label className="form-label">分类</label>
                <select
                  className="input"
                  value={formData.category}
                  onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                >
                  {categories.map((cat) => (
                    <option key={cat} value={cat}>
                      {cat}
                    </option>
                  ))}
                  <option value="其他">其他</option>
                </select>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={handleClose}>
                取消
              </button>
              <button className="btn btn-primary" onClick={handleSubmit}>
                {editingItem ? '保存' : '添加'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
