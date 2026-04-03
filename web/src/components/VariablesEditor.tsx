import { useState, useEffect } from 'react';
import { Plus, Trash2, GripVertical } from 'lucide-react';

interface VariableItem {
  key: string;
  value: any;
  description: string;
  type: 'string' | 'number' | 'boolean' | 'object' | 'array';
}

interface VariablesEditorProps {
  vars: Record<string, any>;
  onChange: (vars: Record<string, any>) => void;
  readonly?: boolean;
}

type ColumnType = 'key' | 'type' | 'value' | 'description' | 'action';

export function VariablesEditor({ vars, onChange, readonly = false }: VariablesEditorProps) {
  const [items, setItems] = useState<VariableItem[]>([]);
  const [columnWidths, setColumnWidths] = useState<Record<ColumnType, number>>({
    key: 150,
    type: 120,
    value: 300,
    description: 200,
    action: 80
  });

  useEffect(() => {
    const newItems: VariableItem[] = Object.entries(vars || {}).map(([key, value]) => {
      // 检查是否有对应的描述字段
      const descriptionKey = key + '_description';
      const description = descriptionKey in vars ? vars[descriptionKey] : '';

      // 如果是描述字段本身，则跳过
      if (key.endsWith('_description')) {
        return null;
      }

      return {
        key,
        value,
        description: description || '',
        type: inferType(value)
      };
    }).filter((item): item is VariableItem => item !== null);

    setItems(newItems);
  }, [vars]);

  const inferType = (value: any): 'string' | 'number' | 'boolean' | 'object' | 'array' => {
    if (Array.isArray(value)) return 'array';
    if (typeof value === 'object' && value !== null) return 'object';
    if (typeof value === 'number') return 'number';
    if (typeof value === 'boolean') return 'boolean';
    return 'string';
  };

  const handleAdd = () => {
    const newItem: VariableItem = {
      key: `var_${items.length + 1}`,
      value: '',
      description: '',
      type: 'string'
    };
    const newItems = [...items, newItem];
    setItems(newItems);
    saveChanges(newItems);
  };

  const handleDelete = (index: number) => {
    const newItems = items.filter((_, i) => i !== index);
    setItems(newItems);
    saveChanges(newItems);
  };

  const handleKeyChange = (index: number, newKey: string) => {
    const newItems = [...items];
    newItems[index].key = newKey;
    setItems(newItems);
    saveChanges(newItems);
  };

  const handleDescriptionChange = (index: number, newDescription: string) => {
    const newItems = [...items];
    newItems[index].description = newDescription;
    setItems(newItems);
    saveChanges(newItems);
  };

  const handleTypeChange = (index: number, newType: VariableItem['type']) => {
    const newItems = [...items];
    newItems[index].type = newType;

    // 根据类型设置默认值
    if (newType === 'number') {
      newItems[index].value = 0;
    } else if (newType === 'boolean') {
      newItems[index].value = false;
    } else if (newType === 'object') {
      newItems[index].value = {};
    } else if (newType === 'array') {
      newItems[index].value = [];
    } else {
      newItems[index].value = '';
    }

    setItems(newItems);
    saveChanges(newItems);
  };

  const handleValueChange = (index: number, newValue: any) => {
    const newItems = [...items];
    newItems[index].value = newValue;
    setItems(newItems);
    saveChanges(newItems);
  };

  const saveChanges = (newItems: VariableItem[]) => {
    const newVars: Record<string, any> = {};
    newItems.forEach((item) => {
      if (item.key && item.key.trim() !== '') {
        newVars[item.key] = item.value;
        if (item.description) {
          newVars[item.key + '_description'] = item.description;
        }
      }
    });
    onChange(newVars);
  };

  const handleMouseDown = (column: ColumnType, e: React.MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = columnWidths[column];

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const diff = moveEvent.clientX - startX;
      const newWidth = Math.max(80, startWidth + diff);
      setColumnWidths(prev => ({
        ...prev,
        [column]: newWidth
      }));
    };

    const handleMouseUp = () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  };

  const renderValueInput = (item: VariableItem, index: number) => {
    if (readonly) {
      return (
        <div className="bg-gray-900 text-green-400 p-2 rounded font-mono text-sm overflow-x-auto whitespace-pre-wrap max-h-40 overflow-y-auto">
          {JSON.stringify(item.value, null, 2)}
        </div>
      );
    }

    if (item.type === 'boolean') {
      return (
        <select
          className="input w-full text-sm"
          value={String(item.value)}
          onChange={(e) => handleValueChange(index, e.target.value === 'true')}
        >
          <option value="true">true</option>
          <option value="false">false</option>
        </select>
      );
    } else if (item.type === 'number') {
      return (
        <input
          type="number"
          className="input w-full text-sm"
          value={item.value}
          onChange={(e) => handleValueChange(index, Number(e.target.value))}
        />
      );
    } else if (item.type === 'string') {
      return (
        <input
          type="text"
          className="input w-full text-sm"
          value={item.value}
          onChange={(e) => handleValueChange(index, e.target.value)}
        />
      );
    } else if (item.type === 'array') {
      const jsonStr = JSON.stringify(item.value, null, 2);
      const lines = jsonStr.split('\n');
      const autoHeight = Math.max(100, Math.min(lines.length * 20, 300));

      return (
        <textarea
          className="w-full bg-gray-900 text-green-400 font-mono text-sm p-2 rounded focus:outline-none resize-none"
          value={jsonStr}
          onChange={(e) => {
            try {
              const parsed = JSON.parse(e.target.value);
              handleValueChange(index, parsed);
            } catch (err) {
              // Ignore JSON parse errors during typing
            }
          }}
          placeholder='输入 JSON 数组，例如: ["a", "b", "c"]'
          style={{ height: `${autoHeight}px` }}
        />
      );
    } else if (item.type === 'object') {
      const jsonStr = JSON.stringify(item.value, null, 2);
      const lines = jsonStr.split('\n');
      const autoHeight = Math.max(100, Math.min(lines.length * 20, 300));

      return (
        <textarea
          className="w-full bg-gray-900 text-green-400 font-mono text-sm p-2 rounded focus:outline-none resize-none"
          value={jsonStr}
          onChange={(e) => {
            try {
              const parsed = JSON.parse(e.target.value);
              handleValueChange(index, parsed);
            } catch (err) {
              // Ignore JSON parse errors during typing
            }
          }}
          placeholder='输入 JSON 对象，例如: {"key": "value"}'
          style={{ height: `${autoHeight}px` }}
        />
      );
    }
  };

  return (
    <div className="space-y-3">
      <div className="card" style={{ overflow: 'auto' }}>
        <table className="table variable-table" style={{ minWidth: '100%' }}>
          <thead>
            <tr>
              <th style={{ width: columnWidths.key, minWidth: columnWidths.key }} className="resizable-th">
                变量名
                <div
                  className="resize-handle"
                  onMouseDown={(e) => handleMouseDown('key', e)}
                >
                  <GripVertical className="w-4 h-4 text-gray-400" />
                </div>
              </th>
              <th style={{ width: columnWidths.type, minWidth: columnWidths.type }} className="resizable-th">
                类型
                <div
                  className="resize-handle"
                  onMouseDown={(e) => handleMouseDown('type', e)}
                >
                  <GripVertical className="w-4 h-4 text-gray-400" />
                </div>
              </th>
              <th style={{ width: columnWidths.value, minWidth: columnWidths.value }} className="resizable-th">
                值
                <div
                  className="resize-handle"
                  onMouseDown={(e) => handleMouseDown('value', e)}
                >
                  <GripVertical className="w-4 h-4 text-gray-400" />
                </div>
              </th>
              <th style={{ width: columnWidths.description, minWidth: columnWidths.description }} className="resizable-th">
                说明
                <div
                  className="resize-handle"
                  onMouseDown={(e) => handleMouseDown('description', e)}
                >
                  <GripVertical className="w-4 h-4 text-gray-400" />
                </div>
              </th>
              <th style={{ width: columnWidths.action, minWidth: columnWidths.action }}>
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            {items.map((item, index) => (
              <tr key={index}>
                <td style={{ width: columnWidths.key, minWidth: columnWidths.key }}>
                  <input
                    type="text"
                    className={`input w-full text-sm ${readonly ? 'bg-gray-50' : ''}`}
                    value={item.key}
                    onChange={(e) => !readonly && handleKeyChange(index, e.target.value)}
                    placeholder="变量名"
                    disabled={readonly}
                  />
                </td>
                <td style={{ width: columnWidths.type, minWidth: columnWidths.type }}>
                  <select
                    className="select w-full text-sm"
                    value={item.type}
                    onChange={(e) => !readonly && handleTypeChange(index, e.target.value as VariableItem['type'])}
                    disabled={readonly}
                  >
                    <option value="string">字符串</option>
                    <option value="number">数字</option>
                    <option value="boolean">布尔</option>
                    <option value="object">对象</option>
                    <option value="array">数组</option>
                  </select>
                </td>
                <td style={{ width: columnWidths.value, minWidth: columnWidths.value }}>
                  {renderValueInput(item, index)}
                </td>
                <td style={{ width: columnWidths.description, minWidth: columnWidths.description }}>
                  <input
                    type="text"
                    className="input w-full text-sm"
                    value={item.description}
                    onChange={(e) => !readonly && handleDescriptionChange(index, e.target.value)}
                    placeholder="配置项说明"
                    disabled={readonly}
                  />
                </td>
                <td style={{ width: columnWidths.action, minWidth: columnWidths.action }}>
                  {!readonly && (
                    <button
                      className="btn btn-sm btn-danger w-full"
                      onClick={() => handleDelete(index)}
                      title="删除"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {items.length === 0 && (
          <div className="empty-state">
            <p>暂无配置项</p>
          </div>
        )}
      </div>

      {/* 添加按钮 */}
      {!readonly && (
        <button className="btn btn-secondary w-full text-base py-2.5" onClick={handleAdd}>
          <Plus className="w-4 h-4 mr-2" />
          添加配置项
        </button>
      )}
    </div>
  );
}
