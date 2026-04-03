import { useState, useEffect } from 'react';
import { Plus, Trash2, ChevronDown, ChevronRight } from 'lucide-react';

interface VariableItem {
  key: string;
  value: any;
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
        <div className="flex-1 bg-gray-50 px-3 py-2 rounded border border-gray-200">
          <code className="text-sm">{JSON.stringify(item.value, null, 2)}</code>
        </div>
      );
    }

    if (item.type === 'boolean') {
      return (
        <select
          className="input flex-1"
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
          className="input flex-1"
          value={item.value}
          onChange={(e) => handleValueChange(index, Number(e.target.value))}
        />
      );
    } else if (item.type === 'string') {
      return (
        <input
          type="text"
          className="input flex-1"
          value={item.value}
          onChange={(e) => handleValueChange(index, e.target.value)}
        />
      );
    } else if (item.type === 'array') {
      const isExpanded = expandedKeys.has(item.key);
      return (
        <div className="flex-1 border border-gray-200 rounded overflow-hidden">
          <div
            className="flex items-center justify-between px-3 py-2 bg-gray-50 cursor-pointer hover:bg-gray-100"
            onClick={() => toggleExpand(item.key)}
          >
            <span className="text-sm text-gray-600">数组 [{Array.isArray(item.value) ? item.value.length : 0}]</span>
            {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          </div>
          {isExpanded && (
            <div className="p-3 bg-white">
              <textarea
                className="w-full h-32 p-2 border border-gray-200 rounded font-mono text-sm"
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
        <div className="flex-1 border border-gray-200 rounded overflow-hidden">
          <div
            className="flex items-center justify-between px-3 py-2 bg-gray-50 cursor-pointer hover:bg-gray-100"
            onClick={() => toggleExpand(item.key)}
          >
            <span className="text-sm text-gray-600">对象 [{keyCount}]</span>
            {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          </div>
          {isExpanded && (
            <div className="p-3 bg-white">
              <textarea
                className="w-full h-32 p-2 border border-gray-200 rounded font-mono text-sm"
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
    <div className="space-y-3">
      {items.map((item, index) => (
        <div key={index} className="flex items-center gap-2">
          <input
            type="text"
            className={`input w-1/3 ${readonly ? 'bg-gray-50' : ''}`}
            value={item.key}
            onChange={(e) => !readonly && handleKeyChange(index, e.target.value)}
            placeholder="变量名"
            disabled={readonly}
          />
          {readonly ? (
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-500 px-2 py-1 bg-gray-100 rounded">
                {item.type}
              </span>
              <div className="flex-1">{renderValueInput(item, index)}</div>
            </div>
          ) : (
            <>
              <select
                className="input w-28"
                value={item.type}
                onChange={(e) => handleTypeChange(index, e.target.value as VariableItem['type'])}
              >
                <option value="string">字符串</option>
                <option value="number">数字</option>
                <option value="boolean">布尔</option>
                <option value="object">对象</option>
                <option value="array">数组</option>
              </select>
              {renderValueInput(item, index)}
              <button className="btn btn-sm btn-danger" onClick={() => handleDelete(index)}>
                <Trash2 className="w-3 h-3" />
              </button>
            </>
          )}
        </div>
      ))}
      {!readonly && (
        <button className="btn btn-secondary w-full" onClick={handleAdd}>
          <Plus className="w-4 h-4" />
          添加变量
        </button>
      )}
      {items.length === 0 && (
        <div className="text-center text-gray-400 py-4">暂无变量配置</div>
      )}
    </div>
  );
}
