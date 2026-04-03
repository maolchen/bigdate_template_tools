import { useState, useEffect } from 'react';
import { Plus, Trash2, ChevronDown, ChevronRight, Info } from 'lucide-react';

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

export function VariablesEditor({ vars, onChange, readonly = false }: VariablesEditorProps) {
  const [items, setItems] = useState<VariableItem[]>([]);
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());

  useEffect(() => {
    const newItems: VariableItem[] = Object.entries(vars || {}).map(([key, value]) => ({
      key,
      value,
      description: '',
      type: inferType(value)
    }));
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
      // 对象默认展开
      setExpandedKeys(new Set([...expandedKeys, `temp_${index}`]));
    } else if (newType === 'array') {
      newItems[index].value = [];
      // 数组默认展开
      setExpandedKeys(new Set([...expandedKeys, `temp_${index}`]));
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
      }
    });
    onChange(newVars);
  };

  const toggleExpand = (key: string) => {
    const newExpanded = new Set(expandedKeys);
    if (newExpanded.has(key)) {
      newExpanded.delete(key);
    } else {
      newExpanded.add(key);
    }
    setExpandedKeys(newExpanded);
  };

  const renderValueInput = (item: VariableItem, index: number) => {
    if (readonly) {
      return (
        <div className="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm overflow-x-auto">
          <pre className="whitespace-pre-wrap break-all">{JSON.stringify(item.value, null, 2)}</pre>
        </div>
      );
    }

    if (item.type === 'boolean') {
      return (
        <select
          className="input w-full"
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
          className="input w-full"
          value={item.value}
          onChange={(e) => handleValueChange(index, Number(e.target.value))}
        />
      );
    } else if (item.type === 'string') {
      return (
        <input
          type="text"
          className="input w-full"
          value={item.value}
          onChange={(e) => handleValueChange(index, e.target.value)}
        />
      );
    } else if (item.type === 'array') {
      const isExpanded = expandedKeys.has(item.key);
      const arrayLength = Array.isArray(item.value) ? item.value.length : 0;
      return (
        <div className="border border-gray-300 rounded-lg overflow-hidden">
          <div
            className="flex items-center justify-between px-4 py-3 bg-gray-100 cursor-pointer hover:bg-gray-200 transition-colors"
            onClick={() => toggleExpand(item.key)}
          >
            <div className="flex items-center gap-2">
              <Info className="w-4 h-4 text-blue-600" />
              <span className="text-sm font-medium text-gray-700">数组</span>
              <span className="text-xs text-gray-500">({arrayLength} 项)</span>
            </div>
            {isExpanded ? <ChevronDown className="w-4 h-4 text-gray-500" /> : <ChevronRight className="w-4 h-4 text-gray-500" />}
          </div>
          {isExpanded && (
            <div className="p-4 bg-gray-900 text-green-400">
              <textarea
                className="w-full h-40 bg-transparent text-green-400 font-mono text-sm resize-none focus:outline-none"
                value={JSON.stringify(item.value, null, 2)}
                onChange={(e) => {
                  try {
                    const parsed = JSON.parse(e.target.value);
                    handleValueChange(index, parsed);
                  } catch (err) {
                    // Ignore JSON parse errors during typing
                  }
                }}
                placeholder='输入 JSON 数组，例如: ["a", "b", "c"]'
              />
            </div>
          )}
        </div>
      );
    } else if (item.type === 'object') {
      const isExpanded = expandedKeys.has(item.key);
      const keyCount = typeof item.value === 'object' && item.value !== null ? Object.keys(item.value).length : 0;
      return (
        <div className="border border-gray-300 rounded-lg overflow-hidden">
          <div
            className="flex items-center justify-between px-4 py-3 bg-gray-100 cursor-pointer hover:bg-gray-200 transition-colors"
            onClick={() => toggleExpand(item.key)}
          >
            <div className="flex items-center gap-2">
              <Info className="w-4 h-4 text-blue-600" />
              <span className="text-sm font-medium text-gray-700">对象</span>
              <span className="text-xs text-gray-500">({keyCount} 个字段)</span>
            </div>
            {isExpanded ? <ChevronDown className="w-4 h-4 text-gray-500" /> : <ChevronRight className="w-4 h-4 text-gray-500" />}
          </div>
          {isExpanded && (
            <div className="p-4 bg-gray-900 text-green-400">
              <textarea
                className="w-full h-40 bg-transparent text-green-400 font-mono text-sm resize-none focus:outline-none"
                value={JSON.stringify(item.value, null, 2)}
                onChange={(e) => {
                  try {
                    const parsed = JSON.parse(e.target.value);
                    handleValueChange(index, parsed);
                  } catch (err) {
                    // Ignore JSON parse errors during typing
                  }
                }}
                placeholder='输入 JSON 对象，例如: {"key": "value"}'
              />
            </div>
          )}
        </div>
      );
    }
  };

  return (
    <div className="space-y-4">
      {items.map((item, index) => (
        <div key={index} className="border border-gray-200 rounded-lg p-4 space-y-3">
          {/* 第一行：变量名和操作 */}
          <div className="flex items-start gap-3">
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">变量名</label>
              <input
                type="text"
                className={`input w-full ${readonly ? 'bg-gray-50' : ''}`}
                value={item.key}
                onChange={(e) => !readonly && handleKeyChange(index, e.target.value)}
                placeholder="变量名"
                disabled={readonly}
              />
            </div>
            {!readonly && (
              <div className="flex items-end">
                <button className="btn btn-sm btn-danger mt-6" onClick={() => handleDelete(index)}>
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            )}
          </div>

          {/* 第二行：类型和说明 */}
          <div className="flex gap-3">
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">类型</label>
              <select
                className="select w-full"
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
            </div>
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">说明</label>
              <input
                type="text"
                className="input w-full"
                value={item.description}
                onChange={(e) => !readonly && handleDescriptionChange(index, e.target.value)}
                placeholder="配置项说明"
                disabled={readonly}
              />
            </div>
          </div>

          {/* 第三行：值 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">值</label>
            {renderValueInput(item, index)}
          </div>
        </div>
      ))}

      {!readonly && (
        <button className="btn btn-secondary w-full" onClick={handleAdd}>
          <Plus className="w-4 h-4 mr-2" />
          添加配置项
        </button>
      )}
    </div>
  );
}
