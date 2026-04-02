import { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { FileCode, Copy, Check, AlertCircle, AlertTriangle } from 'lucide-react';
import type { AppConfig } from '../types/config';
import { exportToYaml } from '../data/defaultConfig';

interface PreviewPageProps {
  config: AppConfig;
}

export function PreviewPage({ config }: PreviewPageProps) {
  const [yamlContent, setYamlContent] = useState('');
  const [copied, setCopied] = useState(false);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const [validationWarnings, setValidationWarnings] = useState<string[]>([]);
  
  useEffect(() => {
    const yaml = exportToYaml(config);
    setYamlContent(yaml);
    
    // 验证配置
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // 检查全局配置
    if (!config.global.user) errors.push('全局配置：运行用户不能为空');
    if (!config.global.install_base_dir) errors.push('全局配置：安装基础目录不能为空');
    if (!config.global.java_home) errors.push('全局配置：JAVA_HOME 不能为空');
    
    // 检查节点
    if (Object.keys(config.nodes).length === 0) {
      errors.push('节点配置：至少需要一个节点');
    }
    
    Object.entries(config.nodes).forEach(([name, node]) => {
      if (!node.ip) errors.push(`节点 ${name}：IP 地址不能为空`);
      if (!node.hostname) errors.push(`节点 ${name}：主机名不能为空`);
    });
    
    // 检查服务拓扑中的节点引用
    const allNodeNames = Object.keys(config.nodes);
    Object.entries(config.serviceTop).forEach(([serviceName, service]) => {
      service.nodes.forEach(node => {
        if (node !== '*' && !allNodeNames.includes(node)) {
          errors.push(`服务拓扑 ${serviceName}：引用了不存在的节点 "${node}"`);
        }
      });
    });
    
    // 警告
    if (Object.keys(config.serviceTop).length === 0) {
      warnings.push('服务拓扑：没有配置任何服务');
    }
    
    if (Object.keys(config.serverConfig).length === 0) {
      warnings.push('服务配置：没有配置任何服务参数');
    }
    
    // 检查是否有服务拓扑但没有对应的服务配置
    Object.keys(config.serviceTop).forEach(serviceName => {
      if (!config.serverConfig[serviceName]) {
        warnings.push(`服务 ${serviceName}：有拓扑配置但没有对应的服务配置`);
      }
    });
    
    setValidationErrors(errors);
    setValidationWarnings(warnings);
  }, [config]);
  
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(yamlContent);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };
  
  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">YAML 预览</h2>
          <p className="text-gray-500 mt-1">预览生成的 config.yaml 配置文件内容</p>
        </div>
        <button className="btn btn-primary" onClick={handleCopy}>
          {copied ? (
            <>
              <Check className="w-4 h-4" />
              已复制
            </>
          ) : (
            <>
              <Copy className="w-4 h-4" />
              复制到剪贴板
            </>
          )}
        </button>
      </div>
      
      {/* 验证结果 */}
      {(validationErrors.length > 0 || validationWarnings.length > 0) && (
        <div className="mb-6 space-y-3">
          {validationErrors.length > 0 && (
            <div className="card border-danger" style={{ borderColor: 'var(--danger)', borderWidth: '1px' }}>
              <div className="card-header" style={{ background: 'rgba(239,68,68,0.05)' }}>
                <div className="flex items-center gap-2 text-danger">
                  <AlertCircle className="w-5 h-5" />
                  <span className="font-semibold">验证错误 ({validationErrors.length})</span>
                </div>
              </div>
              <div className="card-body">
                <ul className="space-y-2">
                  {validationErrors.map((error, idx) => (
                    <li key={idx} className="flex items-start gap-2 text-danger text-sm">
                      <span className="mt-1">•</span>
                      <span>{error}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}
          
          {validationWarnings.length > 0 && (
            <div className="card" style={{ borderColor: 'var(--warning)', borderWidth: '1px' }}>
              <div className="card-header" style={{ background: 'rgba(245,158,11,0.05)' }}>
                <div className="flex items-center gap-2" style={{ color: 'var(--warning)' }}>
                  <AlertTriangle className="w-5 h-5" />
                  <span className="font-semibold">警告 ({validationWarnings.length})</span>
                </div>
              </div>
              <div className="card-body">
                <ul className="space-y-2">
                  {validationWarnings.map((warning, idx) => (
                    <li key={idx} className="flex items-start gap-2 text-sm" style={{ color: 'var(--warning)' }}>
                      <span className="mt-1">•</span>
                      <span>{warning}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}
        </div>
      )}
      
      {validationErrors.length === 0 && validationWarnings.length === 0 && (
        <div className="card mb-6" style={{ borderColor: 'var(--success)', borderWidth: '1px' }}>
          <div className="card-body">
            <div className="flex items-center gap-2 text-success">
              <Check className="w-5 h-5" />
              <span className="font-medium">配置验证通过，没有错误或警告</span>
            </div>
          </div>
        </div>
      )}
      
      {/* 编辑器 */}
      <div className="card">
        <div className="card-header">
          <div className="flex items-center gap-2">
            <FileCode className="w-5 h-5 text-primary" />
            <span className="font-semibold text-gray-800">config.yaml</span>
          </div>
          <span className="text-sm text-gray-500">
            {yamlContent.split('\n').length} 行 · {yamlContent.length} 字符
          </span>
        </div>
        <div className="editor-container" style={{ height: '600px' }}>
          <Editor
            height="100%"
            defaultLanguage="yaml"
            value={yamlContent}
            options={{
              readOnly: true,
              minimap: { enabled: false },
              fontSize: 14,
              lineNumbers: 'on',
              scrollBeyondLastLine: false,
              wordWrap: 'on',
              theme: 'vs'
            }}
          />
        </div>
      </div>
    </div>
  );
}
