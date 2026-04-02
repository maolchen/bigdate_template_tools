import type { AppConfig } from '../types/config';

export const defaultConfig: AppConfig = {
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
  },
  nodes: {
    'node1': {
      ip: '192.168.1.101',
      hostname: 'bigdata-node1'
    },
    'node2': {
      ip: '192.168.1.102',
      hostname: 'bigdata-node2'
    },
    'node3': {
      ip: '192.168.1.103',
      hostname: 'bigdata-node3'
    }
  },
  serviceTop: {
    'jdk': {
      nodes: ['*'],
      description: 'JDK安装（所有节点）',
      vars: {
        version: '1.8.0_202',
        pkg_name: 'jdk-8u202-linux-x64.tar.gz'
      }
    },
    'zookeeper': {
      nodes: ['node1', 'node2', 'node3'],
      description: 'ZooKeeper集群',
      vars: {
        version: '3.7.1',
        client_port: 2181,
        data_subdir: 'zookeeper/data'
      }
    },
    'hadoop/hdfs_namenode': {
      nodes: ['node1', 'node2'],
      description: 'HDFS NameNode（HA）',
      vars: {
        rpc_port: 8020,
        http_port: 9870
      }
    },
    'hadoop/hdfs_datanode': {
      nodes: ['node1', 'node2', 'node3'],
      description: 'HDFS DataNode',
      vars: {
        data_subdir: 'hadoop/hdfs/data'
      }
    },
    'doris/fe': {
      nodes: ['node1'],
      description: 'Doris FE（前端）',
      vars: {
        version: '2.0.3',
        http_port: 8030,
        query_port: 9030,
        install_subdir: 'apache-doris/fe',
        data_subdir: 'apache-doris/fe/doris-meta',
        priority_networks: '192.168.1.0/24'
      }
    },
    'doris/be': {
      nodes: ['node1', 'node2', 'node3'],
      description: 'Doris BE（后端）',
      vars: {
        version: '2.0.3',
        be_port: 9060,
        webserver_port: 8040,
        heartbeat_service_port: 9050,
        install_subdir: 'apache-doris/be',
        data_subdir: 'apache-doris/be/storage'
      }
    }
  },
  serverConfig: {
    'jdk': {
      type: 'global',
      description: 'JDK环境配置',
      vars: {
        version: '1.8.0_202',
        install_dir: '/usr/local/jdk'
      }
    },
    'zookeeper': {
      vars: {
        version: '3.7.1',
        tickTime: 2000,
        initLimit: 10,
        syncLimit: 5,
        dataDir: '{{.Global.data_base_dir}}/zookeeper/data',
        clientPort: 2181
      }
    },
    'doris/fe': {
      vars: {
        version: '2.0.3',
        http_port: 8030,
        rpc_port: 9020,
        query_port: 9030,
        edit_log_port: 9010,
        install_subdir: 'apache-doris/fe',
        data_subdir: 'apache-doris/fe/doris-meta',
        priority_networks: '192.168.1.0/24'
      }
    },
    'doris/be': {
      vars: {
        version: '2.0.3',
        be_port: 9060,
        webserver_port: 8040,
        heartbeat_service_port: 9050,
        brpc_port: 8060,
        install_subdir: 'apache-doris/be',
        data_subdir: 'apache-doris/be/storage'
      }
    }
  }
};

// 从 localStorage 加载配置或返回默认配置
export function loadConfig(): AppConfig {
  try {
    const saved = localStorage.getItem('bigdata-config');
    if (saved) {
      return JSON.parse(saved);
    }
  } catch (e) {
    console.error('Failed to load config from localStorage:', e);
  }
  return JSON.parse(JSON.stringify(defaultConfig));
}

// 保存配置到 localStorage
export function saveConfig(config: AppConfig): void {
  try {
    localStorage.setItem('bigdata-config', JSON.stringify(config));
  } catch (e) {
    console.error('Failed to save config to localStorage:', e);
  }
}

// 导出为 YAML 格式
export function exportToYaml(config: AppConfig): string {
  const lines: string[] = [];
  
  // global
  lines.push('# ============================================================');
  lines.push('# 全局配置');
  lines.push('# ============================================================');
  lines.push('global:');
  Object.entries(config.global).forEach(([key, value]) => {
    lines.push(`  ${key}: ${typeof value === 'string' ? `"${value}"` : value}`);
  });
  
  // nodes
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
  
  // serviceTop
  lines.push('');
  lines.push('# ============================================================');
  lines.push('# 服务拓扑（部署位置）');
  lines.push('# ============================================================');
  lines.push('serviceTop:');
  Object.entries(config.serviceTop).forEach(([name, service]) => {
    lines.push(`  ${name}:`);
    lines.push(`    nodes: [${service.nodes.join(', ')}]`);
    if (service.description) {
      lines.push(`    description: "${service.description}"`);
    }
    if (service.vars && Object.keys(service.vars).length > 0) {
      lines.push('    vars:');
      Object.entries(service.vars).forEach(([k, v]) => {
        if (typeof v === 'string') {
          lines.push(`      ${k}: "${v}"`);
        } else if (Array.isArray(v)) {
          lines.push(`      ${k}: [${v.join(', ')}]`);
        } else {
          lines.push(`      ${k}: ${v}`);
        }
      });
    }
  });
  
  // serverConfig
  lines.push('');
  lines.push('# ============================================================');
  lines.push('# 服务配置（完全配置化，无硬编码）');
  lines.push('# ============================================================');
  lines.push('serverConfig:');
  Object.entries(config.serverConfig).forEach(([name, config]) => {
    lines.push(`  ${name}:`);
    if (config.type) {
      lines.push(`    type: "${config.type}"`);
    }
    if (config.description) {
      lines.push(`    description: "${config.description}"`);
    }
    if (config.vars && Object.keys(config.vars).length > 0) {
      lines.push('    vars:');
      Object.entries(config.vars).forEach(([k, v]) => {
        if (typeof v === 'string') {
          lines.push(`      ${k}: "${v}"`);
        } else if (Array.isArray(v)) {
          lines.push(`      ${k}: [${v.map((i: any) => typeof i === 'string' ? `"${i}"` : i).join(', ')}]`);
        } else {
          lines.push(`      ${k}: ${v}`);
        }
      });
    }
  });
  
  return lines.join('\n');
}
