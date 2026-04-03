import { useState, useEffect } from 'react';
import { Plus, Trash2 } from 'lucide-react';

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
          className="input w-full text-base"
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
          className="input w-full text-base"
          value={item.value}
          onChange={(e) => handleValueChange(index, Number(e.target.value))}
        />
      );
    } else if (item.type === 'string') {
      return (
        <input
          type="text"
          className="input w-full text-base"
          value={item.value}
          onChange={(e) => handleValueChange(index, e.target.value)}
        />
      );
    } else if (item.type === 'array') {
      const jsonStr = JSON.stringify(item.value, null, 2);
      const lines = jsonStr.split('\n');
      const autoHeight = Math.max(120, Math.min(lines.length * 20, 400));

      return (
        <textarea
          className="w-full bg-gray-900 text-green-400 font-mono text-sm p-3 rounded-lg focus:outline-none resize-none"
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
      const autoHeight = Math.max(120, Math.min(lines.length * 20, 400));

      return (
        <textarea
          className="w-full bg-gray-900 text-green-400 font-mono text-sm p-3 rounded-lg focus:outline-none resize-none"
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
      {/* 表头 */}
      {!readonly && (
        <div className="grid grid-cols-12 gap-3 px-3 py-2 bg-gray-100 rounded-lg">
          <div className="col-span-3 text-sm font-medium text-gray-700">变量名</div>
          <div className="col-span-2 text-sm font-medium text-gray-700">类型</div>
          <div className="col-span-3 text-sm font-medium text-gray-700">值</div>
          <div className="col-span-3 text-sm font-medium text-gray-700">说明</div>
          <div className="col-span-1 text-sm font-medium text-gray-700">操作</div>
        </div>
      )}

      {/* 变量列表 */}
      {items.map((item, index) => (
        <div key={index} className="grid grid-cols-12 gap-3 items-start px-3 py-3 border border-gray-200 rounded-lg bg-white hover:bg-gray-50 transition-colors">
          {/* 变量名 */}
          <div className="col-span-3">
            <input
              type="text"
              className={`input w-full text-base ${readonly ? 'bg-gray-50' : ''}`}
              value={item.key}
              onChange={(e) => !readonly && handleKeyChange(index, e.target.value)}
              placeholder="变量名"
              disabled={readonly}
            />
          </div>

          {/* 类型 */}
          <div className="col-span-2">
            <select
              className="select w-full text-base"
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

          {/* 值 */}
          <div className="col-span-3">
            {renderValueInput(item, index)}
          </div>

          {/* 说明 */}
          <div className="col-span-3">
            <input
              type="text"
              className="input w-full text-base"
              value={item.description}
              onChange={(e) => !readonly && handleDescriptionChange(index, e.target.value)}
              placeholder="配置项说明"
              disabled={readonly}
            />
          </div>

          {/* 删除按钮 */}
          <div className="col-span-1">
            {!readonly && (
              <button
                className="btn btn-sm btn-danger w-full"
                onClick={() => handleDelete(index)}
                title="删除"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>
      ))}

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
