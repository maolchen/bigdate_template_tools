import { 
  Globe, 
  Server, 
  Layers, 
  Settings,
  CheckCircle2,
  AlertCircle,
  Play,
  RefreshCw
} from 'lucide-react';
import type { AppConfig } from '../api/config';

interface OverviewPageProps {
  config: AppConfig;
  onReload?: () => void;
  hidePageTitle?: boolean;
}

export function OverviewPage({ config, onReload, hidePageTitle = false }: OverviewPageProps) {
  const nodeCount = Object.keys(config.nodes).length;
  const serviceCount = Object.keys(config.serviceTop).length;
  const configCount = Object.keys(config.serverConfig).length;
  
  // 检查是否有全局服务
  const globalServices = Object.entries(config.serverConfig)
    .filter(([, cfg]) => cfg.type === 'global')
    .map(([name]) => name);
  
  // 检查节点引用是否有效
  const allNodeNames = Object.keys(config.nodes);
  const invalidNodeRefs: { service: string; node: string }[] = [];
  
  Object.entries(config.serviceTop).forEach(([serviceName, service]) => {
    const nodes = service.nodes || [];
    nodes.forEach(node => {
      if (node !== '*' && !allNodeNames.includes(node)) {
        invalidNodeRefs.push({ service: serviceName, node });
      }
    });
  });
  
  const cards = [
    {
      title: '全局配置',
      icon: Globe,
      count: Object.keys(config.global).length,
      desc: '全局变量设置',
      color: 'blue'
    },
    {
      title: '节点数量',
      icon: Server,
      count: nodeCount,
      desc: '已配置的节点',
      color: 'green'
    },
    {
      title: '服务拓扑',
      icon: Layers,
      count: serviceCount,
      desc: '服务部署配置',
      color: 'blue'
    },
    {
      title: '服务配置',
      icon: Settings,
      count: configCount,
      desc: '详细参数配置',
      color: 'gray'
    }
  ];
  
  return (
    <div className="overview-page">
      {!hidePageTitle && <div className="mb-6 overview-intro">
        <h2 className="text-2xl font-bold text-gray-800">配置概览</h2>
        <p className="text-gray-500 mt-1">查看当前配置的整体状态和统计信息</p>
      </div>}
      
      {/* 快速操作 */}
      <div className="card card-hero mb-6">
        <div className="card-body text-white overview-hero-body">
          <div className="flex items-center justify-between overview-hero-inner">
            <div className="overview-hero-copy">
              <h3 className="text-lg font-semibold">准备生成配置？</h3>
              <p className="text-sm opacity-90 mt-1">点击「生成配置」按钮，根据当前配置生成部署脚本</p>
            </div>
            <div className="flex gap-2 overview-hero-actions">
              {onReload && (
                <button 
                  className="btn btn-glass-light"
                  onClick={onReload}
                >
                  <RefreshCw className="w-4 h-4" />
                  重新加载
                </button>
              )}
              <button 
                className="btn btn-hero-primary"
                onClick={() => window.dispatchEvent(new CustomEvent('navigate', { detail: 'generate' }))}
              >
                <Play className="w-4 h-4" />
                生成配置
              </button>
            </div>
          </div>
        </div>
      </div>
      
      {/* 统计卡片 */}
      <div className="grid grid-cols-4 mb-6 overview-stats">
        {cards.map((card) => {
          const Icon = card.icon;
          return (
            <div key={card.title} className="card overview-stat-card">
              <div className="card-body overview-stat-body">
                <div className="flex items-center justify-between mb-4 overview-stat-top">
                  <div className={`w-12 h-12 rounded-lg flex items-center justify-center ${
                    card.color === 'blue' ? 'bg-blue-50' : 
                    card.color === 'green' ? 'bg-green-50' : 'bg-gray-100'
                  }`}>
                    <Icon className={`w-6 h-6 ${
                      card.color === 'blue' ? 'text-blue-500' : 
                      card.color === 'green' ? 'text-green-500' : 'text-gray-500'
                    }`} />
                  </div>
                  <span className="text-2xl font-bold text-gray-800 overview-stat-count">{card.count}</span>
                </div>
                <div className="overview-stat-copy">
                  <h3 className="font-medium text-gray-800">{card.title}</h3>
                  <p className="text-sm text-gray-500">{card.desc}</p>
                </div>
              </div>
            </div>
          );
        })}
      </div>
      
      {/* 全局配置摘要 */}
      <div className="card mb-6">
        <div className="card-header">
          <h3 className="font-semibold text-gray-800">全局配置摘要</h3>
        </div>
        <div className="card-body">
          <div className="grid grid-cols-3 gap-4">
            <div>
              <span className="text-sm text-gray-500">运行用户</span>
              <p className="font-medium text-gray-800">{config.global.user}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">安装目录</span>
              <p className="font-medium text-gray-800">{config.global.install_base_dir}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">数据目录</span>
              <p className="font-medium text-gray-800">{config.global.data_base_dir}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">软件包目录</span>
              <p className="font-medium text-gray-800">{config.global.pkg_base_dir}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">日志目录</span>
              <p className="font-medium text-gray-800">{config.global.log_base_dir}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">JAVA_HOME</span>
              <p className="font-medium text-gray-800">{config.global.java_home}</p>
            </div>
          </div>
        </div>
      </div>
      
      {/* 全局服务 */}
      {globalServices.length > 0 && (
        <div className="card mb-6">
          <div className="card-header">
            <h3 className="font-semibold text-gray-800">全局服务</h3>
            <span className="badge badge-green">{globalServices.length} 个</span>
          </div>
          <div className="card-body">
            <div className="flex flex-wrap gap-2">
              {globalServices.map(name => (
                <span key={name} className="badge badge-blue">{name}</span>
              ))}
            </div>
          </div>
        </div>
      )}
      
      {/* 验证状态 */}
      <div className="card">
        <div className="card-header">
          <h3 className="font-semibold text-gray-800">配置验证</h3>
          {invalidNodeRefs.length === 0 ? (
            <span className="badge badge-green flex items-center gap-1">
              <CheckCircle2 className="w-3 h-3" />
              通过
            </span>
          ) : (
            <span className="badge badge-danger flex items-center gap-1">
              <AlertCircle className="w-3 h-3" />
              {invalidNodeRefs.length} 个问题
            </span>
          )}
        </div>
        <div className="card-body">
          {invalidNodeRefs.length === 0 ? (
            <div className="flex items-center gap-2 text-success">
              <CheckCircle2 className="w-5 h-5" />
              <span>所有服务拓扑中的节点引用都有效</span>
            </div>
          ) : (
            <div className="space-y-2">
              <p className="text-danger mb-3">以下服务引用了不存在的节点：</p>
              {invalidNodeRefs.map((ref, idx) => (
                <div key={idx} className="flex items-center gap-2 text-sm text-danger">
                  <AlertCircle className="w-4 h-4" />
                  <span>服务 <code>{ref.service}</code> 引用了未定义的节点 <code>{ref.node}</code></span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
