import { useState } from 'react';
import { Download, FileCode, FileJson, FileArchive, Check, AlertCircle } from 'lucide-react';
import type { AppConfig } from '../types/config';
import { exportToYaml } from '../data/defaultConfig';

interface ExportPageProps {
  config: AppConfig;
}

export function ExportPage({ config }: ExportPageProps) {
  const [downloaded, setDownloaded] = useState<string | null>(null);
  
  const handleDownloadYaml = () => {
    const yaml = exportToYaml(config);
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
  
  const handleDownloadEnv = () => {
    // 生成环境变量文件
    const lines: string[] = [
      '# 大数据平台环境变量配置',
      '# 由配置编辑器自动生成',
      '',
      '# 全局配置',
      `export BIGDATA_USER="${config.global.user}"`,
      `export BIGDATA_GROUP="${config.global.group}"`,
      `export BIGDATA_VERSION="${config.global.version}"`,
      '',
      '# 目录配置',
      `export PKG_BASE_DIR="${config.global.pkg_base_dir}"`,
      `export INSTALL_BASE_DIR="${config.global.install_base_dir}"`,
      `export DATA_BASE_DIR="${config.global.data_base_dir}"`,
      `export SOFTWARE_DIR="${config.global.software_dir}"`,
      `export LOG_BASE_DIR="${config.global.log_base_dir}"`,
      '',
      '# Java 配置',
      `export JAVA_HOME="${config.global.java_home}"`,
      'export PATH="${JAVA_HOME}/bin:${PATH}"',
      '',
      '# 节点列表',
      `export NODES="${Object.keys(config.nodes).join(' ')}"`,
    ];
    
    // 添加每个节点的信息
    Object.entries(config.nodes).forEach(([name, node]) => {
      lines.push(`export NODE_${name.toUpperCase()}_IP="${node.ip}"`);
      lines.push(`export NODE_${name.toUpperCase()}_HOSTNAME="${node.hostname}"`);
    });
    
    lines.push('');
    lines.push('# 服务列表');
    lines.push(`export SERVICES="${Object.keys(config.serviceTop).join(' ')}"`);
    
    const content = lines.join('\n');
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'bigdata-env.sh';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    
    setDownloaded('env');
    setTimeout(() => setDownloaded(null), 2000);
  };
  
  const handleDownloadHosts = () => {
    // 生成 hosts 文件
    const lines: string[] = [
      '# 大数据平台 hosts 配置',
      '# 由配置编辑器自动生成',
      '',
      '# 集群节点',
    ];
    
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
  
  const nodeCount = Object.keys(config.nodes).length;
  const serviceCount = Object.keys(config.serviceTop).length;
  
  return (
    <div>
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-800">导出配置</h2>
        <p className="text-gray-500 mt-1">将配置导出为不同格式的文件，用于部署和共享</p>
      </div>
      
      {/* 导出选项 */}
      <div className="grid grid-cols-2 gap-6">
        {/* YAML 配置 */}
        <div className="card">
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-blue-50 flex items-center justify-center flex-shrink-0">
                <FileCode className="w-6 h-6 text-primary" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">config.yaml</h3>
                <p className="text-sm text-gray-500 mt-1">
                  主配置文件，包含全局配置、节点、服务拓扑和服务配置
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
        <div className="card">
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-green-50 flex items-center justify-center flex-shrink-0">
                <FileJson className="w-6 h-6 text-success" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">config.json</h3>
                <p className="text-sm text-gray-500 mt-1">
                  JSON 格式的配置文件，便于程序解析和处理
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>结构化数据</span>
                  <span>易于解析</span>
                </div>
              </div>
            </div>
            <button 
              className="btn btn-success w-full mt-4"
              onClick={handleDownloadJson}
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
        
        {/* 环境变量 */}
        <div className="card">
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-purple-50 flex items-center justify-center flex-shrink-0">
                <FileArchive className="w-6 h-6" style={{ color: '#9333ea' }} />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">bigdata-env.sh</h3>
                <p className="text-sm text-gray-500 mt-1">
                  Shell 环境变量脚本，可在部署时 source 使用
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>环境变量</span>
                  <span>Shell 脚本</span>
                </div>
              </div>
            </div>
            <button 
              className="btn btn-secondary w-full mt-4"
              onClick={handleDownloadEnv}
              style={{ background: 'rgba(147,51,234,0.1)', color: '#9333ea', border: 'none' }}
            >
              {downloaded === 'env' ? (
                <>
                  <Check className="w-4 h-4" />
                  已下载
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  下载环境变量脚本
                </>
              )}
            </button>
          </div>
        </div>
        
        {/* Hosts 文件 */}
        <div className="card">
          <div className="card-body">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-orange-50 flex items-center justify-center flex-shrink-0">
                <FileCode className="w-6 h-6" style={{ color: '#f97316' }} />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800">hosts</h3>
                <p className="text-sm text-gray-500 mt-1">
                  /etc/hosts 格式的 hosts 文件，用于 DNS 解析
                </p>
                <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                  <span>{nodeCount} 条记录</span>
                  <span>/etc/hosts 格式</span>
                </div>
              </div>
            </div>
            <button 
              className="btn btn-secondary w-full mt-4"
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
      </div>
      
      {/* 使用说明 */}
      <div className="card mt-6">
        <div className="card-header">
          <div className="flex items-center gap-2">
            <AlertCircle className="w-5 h-5 text-primary" />
            <span className="font-semibold text-gray-800">使用说明</span>
          </div>
        </div>
        <div className="card-body">
          <div className="space-y-4 text-sm text-gray-600">
            <div>
              <h4 className="font-medium text-gray-800 mb-1">config.yaml</h4>
              <p>主配置文件，需要放置在配置生成器的根目录，供 Go 程序读取并渲染模板。</p>
            </div>
            <div>
              <h4 className="font-medium text-gray-800 mb-1">config.json</h4>
              <p>JSON 格式的配置，可用于其他程序解析配置信息，或作为备份。</p>
            </div>
            <div>
              <h4 className="font-medium text-gray-800 mb-1">bigdata-env.sh</h4>
              <p>在部署节点上执行 <code>source bigdata-env.sh</code> 可设置环境变量，方便脚本使用。</p>
            </div>
            <div>
              <h4 className="font-medium text-gray-800 mb-1">hosts</h4>
              <p>将内容追加到 <code>/etc/hosts</code> 文件，实现集群节点间的 DNS 解析。</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
