import { useEffect, useMemo, useState } from 'react';
import Editor from '@monaco-editor/react';
import { FileCode, Copy, Check, AlertCircle, AlertTriangle } from 'lucide-react';
import type { AppConfig } from '../api/config';

interface PreviewPageProps {
  config: AppConfig;
  hidePageTitle?: boolean;
}

function yamlValueToString(value: any, indent: number = 0): string {
  const prefix = '  '.repeat(indent);

  if (value === null || value === undefined) {
    return 'null';
  }
  if (typeof value === 'string') {
    if (/[:{}\[\],\n\r\t]/.test(value)) {
      return `"${value.replace(/"/g, '\\"')}"`;
    }
    return value;
  }
  if (typeof value === 'boolean' || typeof value === 'number') {
    return String(value);
  }
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return '[]';
    }
    if (value.every((item) => typeof item === 'string' || typeof item === 'number' || typeof item === 'boolean')) {
      return `[${value.map((item) => (typeof item === 'string' ? `"${item}"` : String(item))).join(', ')}]`;
    }
    const items = value.map((item) => `${prefix}  - ${yamlValueToString(item, indent + 1)}`);
    return `\n${items.join('\n')}`;
  }
  if (typeof value === 'object') {
    const keys = Object.keys(value);
    if (keys.length === 0) {
      return '{}';
    }
    const entries = keys.map((key) => {
      const valStr = yamlValueToString(value[key], indent + 1);
      if (valStr.includes('\n')) {
        return `${prefix}  ${key}:\n${valStr}`;
      }
      return `${prefix}  ${key}: ${valStr}`;
    });
    return `\n${entries.join('\n')}`;
  }

  return String(value);
}

function configToYaml(config: AppConfig): string {
  const lines: string[] = [];

  lines.push('# ============================================================');
  lines.push('# 全局配置');
  lines.push('# ============================================================');
  lines.push('global:');
  Object.entries(config.global).forEach(([key, value]) => {
    lines.push(`  ${key}: ${typeof value === 'string' ? `"${value}"` : value}`);
  });

  lines.push('');
  lines.push('# ============================================================');
  lines.push('# 节点配置');
  lines.push('# ============================================================');
  lines.push('nodes:');
  Object.entries(config.nodes).forEach(([name, node]) => {
    lines.push(`  ${name}:`);
    lines.push(`    ip: "${node.ip}"`);
    lines.push(`    hostname: "${node.hostname}"`);
  });

  lines.push('');
  lines.push('# ============================================================');
  lines.push('# 服务拓扑（部署位置）');
  lines.push('# ============================================================');
  lines.push('serviceTop:');
  Object.entries(config.serviceTop).forEach(([name, service]) => {
    lines.push(`  ${name}:`);
    lines.push(`    nodes: [${(service.nodes || []).join(', ')}]`);
    if (service.description) {
      lines.push(`    description: "${service.description}"`);
    }
    if (service.id_auto_derive) {
      lines.push('    id_auto_derive: true');
    }
  });

  lines.push('');
  lines.push('# ============================================================');
  lines.push('# 服务配置');
  lines.push('# ============================================================');
  lines.push('serverConfig:');
  Object.entries(config.serverConfig).forEach(([name, cfg]) => {
    lines.push(`  ${name}:`);
    if (cfg.type) {
      lines.push(`    type: "${cfg.type}"`);
    }
    if (cfg.description) {
      lines.push(`    description: "${cfg.description}"`);
    }
    if (cfg.vars && Object.keys(cfg.vars).length > 0) {
      lines.push('    vars:');
      Object.entries(cfg.vars).forEach(([k, v]) => {
        const valStr = yamlValueToString(v, 3);
        if (valStr.includes('\n')) {
          lines.push(`      ${k}:${valStr}`);
        } else {
          lines.push(`      ${k}: ${valStr}`);
        }
      });
    }
  });

  return lines.join('\n');
}

export function PreviewPage({ config, hidePageTitle = false }: PreviewPageProps) {
  const [yamlContent, setYamlContent] = useState('');
  const [copied, setCopied] = useState(false);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const [validationWarnings, setValidationWarnings] = useState<string[]>([]);

  useEffect(() => {
    const yaml = configToYaml(config);
    setYamlContent(yaml);

    const errors: string[] = [];
    const warnings: string[] = [];

    if (!config.global.user) errors.push('全局配置：运行用户不能为空');
    if (!config.global.install_base_dir) errors.push('全局配置：安装基础目录不能为空');
    if (!config.global.java_home) errors.push('全局配置：JAVA_HOME 不能为空');

    if (Object.keys(config.nodes).length === 0) {
      errors.push('节点配置：至少需要一个节点');
    }

    Object.entries(config.nodes).forEach(([name, node]) => {
      if (!node.ip) errors.push(`节点 ${name}：IP 地址不能为空`);
      if (!node.hostname) errors.push(`节点 ${name}：主机名不能为空`);
    });

    const allNodeNames = Object.keys(config.nodes);
    Object.entries(config.serviceTop).forEach(([serviceName, service]) => {
      (service.nodes || []).forEach((node) => {
        if (node !== '*' && !allNodeNames.includes(node)) {
          errors.push(`服务拓扑 ${serviceName}：引用了不存在的节点 "${node}"`);
        }
      });
    });

    if (Object.keys(config.serviceTop).length === 0) {
      warnings.push('服务拓扑：没有配置任何服务');
    }
    if (Object.keys(config.serverConfig).length === 0) {
      warnings.push('服务配置：没有配置任何服务参数');
    }

    setValidationErrors(errors);
    setValidationWarnings(warnings);
  }, [config]);

  const editorHeight = useMemo(() => {
    const lineCount = Math.max(1, yamlContent.split('\n').length);
    return Math.max(320, lineCount * 22 + 36);
  }, [yamlContent]);

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
        {!hidePageTitle && (
          <div>
            <h2 className="text-2xl font-bold text-gray-800">YAML 预览</h2>
            <p className="text-gray-500 mt-1">预览生成的 config.yaml 配置文件内容</p>
          </div>
        )}
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
        <div className="editor-container preview-editor-auto" style={{ height: `${editorHeight}px` }}>
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
              theme: 'vs',
              automaticLayout: true,
              scrollbar: {
                alwaysConsumeMouseWheel: false,
              },
            }}
          />
        </div>
      </div>
    </div>
  );
}
