import { useState } from 'react';
import { Download, FileCode, FileJson, FileArchive, Check } from 'lucide-react';
import type { AppConfig } from '../api/config';
import { downloadOutput } from '../api/config';

interface ExportPageProps {
  config: AppConfig;
  hidePageTitle?: boolean;
}

// 简单的 YAML 导出函数
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
  lines.push('# 服务拓扑');
  lines.push('# ============================================================');
  lines.push('serviceTop:');
  Object.entries(config.serviceTop).forEach(([name, service]) => {
    const nodes = service.nodes || [];
    lines.push(`  ${name}:`);
    lines.push(`    nodes: [${nodes.join(', ')}]`);
    if (service.description) {
      lines.push(`    description: "${service.description}"`);
    }
  });
  
  lines.push('');
  lines.push('# ============================================================');
  lines.push('# 服务配置');
  lines.push('# ============================================================');
  lines.push('serverConfig:');
  Object.entries(config.serverConfig).forEach(([name, cfg]) => {
    lines.push(`  ${name}:`);
    if (cfg.type) lines.push(`    type: "${cfg.type}"`);
    if (cfg.description) lines.push(`    description: "${cfg.description}"`);
  });
  
  return lines.join('\n');
}

export function ExportPage({ config, hidePageTitle = false }: ExportPageProps) {
  const [downloaded, setDownloaded] = useState<string | null>(null);
  
  const handleDownloadYaml = () => {
    const yaml = configToYaml(config);
    const blob = new Blob([yaml], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'config.yaml';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    
    setDownloaded('yaml');
    setTimeout(() => setDownloaded(null), 2000);
  };
  
  const handleDownloadJson = () => {
    const json = JSON.stringify(config, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'config.json';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    
    setDownloaded('json');
    setTimeout(() => setDownloaded(null), 2000);
  };
  
  const handleDownloadOutput = () => {
    downloadOutput();
    setDownloaded('output');
    setTimeout(() => setDownloaded(null), 2000);
  };
  
  const handleDownloadHosts = () => {
    const lines: string[] = ['# 集群节点 hosts 配置', ''];
    Object.entries(config.nodes).forEach(([, node]) => {
      lines.push(`${node.ip} ${node.hostname}`);
    });

    const content = lines.join('\n');
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'hosts';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);

    setDownloaded('hosts');
    setTimeout(() => setDownloaded(null), 2000);
  };

  const handleDownloadNodeTable = () => {
    const lines: string[] = ['# 节点配置表', '', '节点名称,IP地址,主机名'];
    Object.entries(config.nodes).forEach(([name, node]) => {
      lines.push(`${name},${node.ip},${node.hostname}`);
    });

    const content = lines.join('\n');
    const blob = new Blob([content], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'nodes.csv';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);

    setDownloaded('node-table');
    setTimeout(() => setDownloaded(null), 2000);
  };

  const handleDownloadServiceTopo = () => {
    const lines: string[] = ['# 服务拓扑分布表', '', '服务名称,节点列表'];
    Object.entries(config.serviceTop).forEach(([serviceName, service]) => {
      const nodes = service.nodes || [];
      lines.push(`${serviceName},"${nodes.join(', ')}"`);
    });

    const content = lines.join('\n');
    const blob = new Blob([content], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'service-topology.csv';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);

    setDownloaded('service-topo');
    setTimeout(() => setDownloaded(null), 2000);
  };

  const handleDownloadFullReport = () => {
    const lines: string[] = ['# 大数据平台配置报告', '# 生成时间: ' + new Date().toLocaleString('zh-CN'), ''];

    // 1. 全局配置摘要
    lines.push('# ============================================================');
    lines.push('# 全局配置');
    lines.push('# ============================================================');
    Object.entries(config.global).forEach(([key, value]) => {
      lines.push(`${key}: ${typeof value === 'string' ? `"${value}"` : value}`);
    });
    lines.push('');

    // 2. 节点配置
    lines.push('# ============================================================');
    lines.push('# 节点配置');
    lines.push('# ============================================================');
    lines.push('节点名称,IP地址,主机名');
    Object.entries(config.nodes).forEach(([name, node]) => {
      lines.push(`${name},${node.ip},${node.hostname}`);
    });
    lines.push('');

    // 3. 服务拓扑分布表
    lines.push('# ============================================================');
    lines.push('# 服务拓扑分布表');
    lines.push('# ============================================================');
    lines.push('服务名称,节点列表,节点数量');
    Object.entries(config.serviceTop).forEach(([serviceName, service]) => {
      const nodes = service.nodes || [];
      const displayNodes = nodes.length > 0 ? nodes.join(', ') : '未配置';
      lines.push(`${serviceName},"${displayNodes}",${nodes.length}`);
    });
    lines.push('');

    // 4. 节点服务分布（按节点）
    lines.push('# ============================================================');
    lines.push('# 节点服务分布（按节点）');
    lines.push('# ============================================================');
    lines.push('节点名称,IP地址,部署的服务');
    Object.entries(config.nodes).forEach(([nodeName, node]) => {
      const servicesOnNode: string[] = [];
      Object.entries(config.serviceTop).forEach(([serviceName, service]) => {
        const nodes = service.nodes || [];
        if (nodes.includes(nodeName) || nodes.includes('*')) {
          servicesOnNode.push(serviceName);
        }
      });
      lines.push(`${nodeName},${node.ip},"${servicesOnNode.join(', ')}"`);
    });

    const content = lines.join('\n');
    const blob = new Blob([content], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'config-report.csv';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);

    setDownloaded('report');
    setTimeout(() => setDownloaded(null), 2000);
  };
  
  const nodeCount = Object.keys(config.nodes).length;
  const serviceCount = Object.keys(config.serviceTop).length;
  
  return (
    <div>
      {!hidePageTitle && (
        <div className="mb-6">
          <h2 className="text-2xl font-bold text-gray-800">导出配置</h2>
          <p className="text-gray-500 mt-1">将配置导出为不同格式的文件</p>
        </div>
      )}
      
      {/* 导出选项 */}
      <div className="grid grid-cols-2 gap-6 export-grid">
        {/* 生成的部署脚本 */}
        <div className="card export-card" style={{ borderColor: 'var(--success)', borderWidth: '2px' }}>
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg flex items-center justify-center flex-shrink-0" style={{ background: 'rgba(16,185,129,0.1)' }}>
                <FileArchive className="w-6 h-6" style={{ color: 'var(--success)' }} />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">
                  生成的部署脚本
                  <span className="badge badge-green ml-2">推荐</span>
                </h3>
                <p className="text-sm text-gray-500 mt-1">
                  包含所有生成的 Shell 安装脚本和配置文件，可直接部署使用；需要先在「生成配置」页面生成脚本
                </p>
              </div>
            </div>
            <button 
              className="btn w-full mt-4"
              onClick={handleDownloadOutput}
              style={{ background: 'var(--success)', color: 'white' }}
            >
              {downloaded === 'output' ? (
                <>
                  <Check className="w-4 h-4" />
                  已下载
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  下载部署脚本 (output.zip)
                </>
              )}
            </button>
          </div>
        </div>
        
        {/* YAML 配置 */}
        <div className="card export-card" style={{ borderColor: 'var(--primary)', borderWidth: '2px' }}>
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-blue-50 flex items-center justify-center flex-shrink-0">
                <FileCode className="w-6 h-6 text-primary" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">config.yaml</h3>
                <p className="text-sm text-gray-500 mt-1">
                  配置源文件，可直接编辑或版本控制
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>{nodeCount} 个节点</span>
                  <span>{serviceCount} 个服务</span>
                </div>
              </div>
            </div>
            <button 
              className="btn btn-primary w-full mt-4"
              onClick={handleDownloadYaml}
            >
              {downloaded === 'yaml' ? (
                <>
                  <Check className="w-4 h-4" />
                  已下载
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  下载 YAML
                </>
              )}
            </button>
          </div>
        </div>
        
        {/* JSON 配置 */}
        <div className="card export-card" style={{ borderColor: '#9333ea', borderWidth: '2px' }}>
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-purple-50 flex items-center justify-center flex-shrink-0">
                <FileJson className="w-6 h-6" style={{ color: '#9333ea' }} />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">config.json</h3>
                <p className="text-sm text-gray-500 mt-1">
                  JSON 格式配置，便于程序解析
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>结构化数据</span>
                  <span>易于解析</span>
                </div>
              </div>
            </div>
            <button 
              className="btn w-full mt-4"
              onClick={handleDownloadJson}
              style={{ background: 'rgba(147,51,234,0.1)', color: '#9333ea', border: 'none' }}
            >
              {downloaded === 'json' ? (
                <>
                  <Check className="w-4 h-4" />
                  已下载
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  下载 JSON
                </>
              )}
            </button>
          </div>
        </div>
        
        {/* Hosts 文件 */}
        <div className="card export-card" style={{ borderColor: '#f97316', borderWidth: '2px' }}>
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-orange-50 flex items-center justify-center flex-shrink-0">
                <FileCode className="w-6 h-6" style={{ color: '#f97316' }} />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">hosts</h3>
                <p className="text-sm text-gray-500 mt-1">
                  /etc/hosts 格式，用于 DNS 解析
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>{nodeCount} 条记录</span>
                </div>
              </div>
            </div>
            <button 
              className="btn w-full mt-4"
              onClick={handleDownloadHosts}
              style={{ background: 'rgba(249,115,22,0.1)', color: '#f97316', border: 'none' }}
            >
              {downloaded === 'hosts' ? (
                <>
                  <Check className="w-4 h-4" />
                  已下载
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  下载 Hosts 文件
                </>
              )}
            </button>
          </div>
        </div>

        {/* 节点配置表 */}
        <div className="card export-card" style={{ borderColor: '#22c55e', borderWidth: '2px' }}>
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-green-50 flex items-center justify-center flex-shrink-0">
                <FileCode className="w-6 h-6" style={{ color: '#22c55e' }} />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">节点配置表</h3>
                <p className="text-sm text-gray-500 mt-1">
                  CSV 格式的节点列表，包含名称、IP、主机名
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>{nodeCount} 个节点</span>
                  <span>CSV 格式</span>
                </div>
              </div>
            </div>
            <button
              className="btn w-full mt-4"
              onClick={handleDownloadNodeTable}
              style={{ background: 'rgba(34,197,94,0.1)', color: '#22c55e', border: 'none' }}
            >
              {downloaded === 'node-table' ? (
                <>
                  <Check className="w-4 h-4" />
                  已下载
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  下载节点表
                </>
              )}
            </button>
          </div>
        </div>

        {/* 服务拓扑分布表 */}
        <div className="card export-card" style={{ borderColor: '#06b6d4', borderWidth: '2px' }}>
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-cyan-50 flex items-center justify-center flex-shrink-0">
                <FileCode className="w-6 h-6" style={{ color: '#06b6d4' }} />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">服务拓扑分布表</h3>
                <p className="text-sm text-gray-500 mt-1">
                  CSV 格式，显示每个服务部署在哪些节点
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>{serviceCount} 个服务</span>
                  <span>CSV 格式</span>
                </div>
              </div>
            </div>
            <button
              className="btn w-full mt-4"
              onClick={handleDownloadServiceTopo}
              style={{ background: 'rgba(6,182,212,0.1)', color: '#06b6d4', border: 'none' }}
            >
              {downloaded === 'service-topo' ? (
                <>
                  <Check className="w-4 h-4" />
                  已下载
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  下载服务拓扑表
                </>
              )}
            </button>
          </div>
        </div>

        {/* 完整配置报告 */}
        <div className="card export-card" style={{ borderColor: 'var(--primary)', borderWidth: '2px' }}>
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-indigo-50 flex items-center justify-center flex-shrink-0">
                <FileArchive className="w-6 h-6" style={{ color: '#6366f1' }} />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">
                  完整配置报告
                  <span className="badge badge-primary ml-2">推荐</span>
                </h3>
                <p className="text-sm text-gray-500 mt-1">
                  包含全局配置、节点列表、服务拓扑、节点服务分布的完整报告
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>CSV 格式</span>
                  <span>完整信息</span>
                </div>
              </div>
            </div>
            <button
              className="btn w-full mt-4"
              onClick={handleDownloadFullReport}
              style={{ background: 'var(--primary)', color: 'white' }}
            >
              {downloaded === 'report' ? (
                <>
                  <Check className="w-4 h-4" />
                  已下载
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  下载配置报告
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
