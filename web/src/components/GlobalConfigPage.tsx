import { RotateCcw } from 'lucide-react';
import type { AppConfig, GlobalConfig } from '../types/config';

interface GlobalConfigPageProps {
  config: AppConfig;
  onChange: (config: AppConfig) => void;
}

export function GlobalConfigPage({ config, onChange }: GlobalConfigPageProps) {
  const handleChange = (key: keyof GlobalConfig, value: string) => {
    onChange({
      ...config,
      global: {
        ...config.global,
        [key]: value
      }
    });
  };
  
  const handleReset = () => {
    if (confirm('确定要重置全局配置为默认值吗？')) {
      onChange({
        ...config,
        global: {
          user: 'hadoop',
          group: 'hadoop',
          version: '1.0.0',
          pkg_base_dir: '/data/packages',
          install_base_dir: '/data',
          data_base_dir: '/data',
          software_dir: '/data/software',
          log_base_dir: '/data/logs',
          java_home: '/usr/local/jdk'
        }
      });
    }
  };
  
  const fields: { key: keyof GlobalConfig; label: string; placeholder: string; required?: boolean }[] = [
    { key: 'user', label: '运行用户', placeholder: '例如: hadoop', required: true },
    { key: 'group', label: '运行用户组', placeholder: '例如: hadoop', required: true },
    { key: 'version', label: '配置版本', placeholder: '例如: 1.0.0', required: true },
    { key: 'java_home', label: 'Java Home 路径', placeholder: '例如: /usr/local/jdk', required: true },
    { key: 'pkg_base_dir', label: '软件包基础目录', placeholder: '例如: /data/packages', required: true },
    { key: 'install_base_dir', label: '安装基础目录', placeholder: '例如: /data', required: true },
    { key: 'data_base_dir', label: '数据基础目录', placeholder: '例如: /data', required: true },
    { key: 'software_dir', label: '软件目录', placeholder: '例如: /data/software', required: true },
    { key: 'log_base_dir', label: '日志基础目录', placeholder: '例如: /data/logs', required: true },
  ];
  
  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">全局配置</h2>
          <p className="text-gray-500 mt-1">配置全局变量，这些变量将在模板渲染时被替换</p>
        </div>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={handleReset}>
            <RotateCcw className="w-4 h-4" />
            重置
          </button>
        </div>
      </div>
      
      <div className="card">
        <div className="card-body">
          <div className="grid grid-cols-2 gap-6">
            {fields.map((field) => (
              <div key={field.key} className="form-group">
                <label className="form-label">
                  {field.label}
                  {field.required && <span className="text-danger ml-1">*</span>}
                </label>
                <input
                  type="text"
                  className="input"
                  value={config.global[field.key]}
                  onChange={(e) => handleChange(field.key, e.target.value)}
                  placeholder={field.placeholder}
                />
              </div>
            ))}
          </div>
          
          <div className="mt-6 pt-6 border-t">
            <h3 className="font-medium text-gray-800 mb-3">变量使用说明</h3>
            <div className="bg-gray-50 rounded-lg p-4 text-sm text-gray-600 space-y-2">
              <p>在模板文件中，可以通过以下方式引用全局变量：</p>
              <ul className="list-disc list-inside space-y-1 ml-2">
                <li><code>{'{{.Global.user}}'}</code> - 运行用户</li>
                <li><code>{'{{.Global.install_base_dir}}'}</code> - 安装基础目录</li>
                <li><code>{'{{.Global.java_home}}'}</code> - Java Home 路径</li>
              </ul>
              <p className="mt-2 text-gray-500">这些变量会在渲染模板时自动替换为实际值</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
