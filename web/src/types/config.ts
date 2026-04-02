// 类型定义已移到 api/config.ts
// 此文件保留用于兼容性导出

export type { 
  AppConfig, 
  GlobalConfig, 
  NodeInfo, 
  ServiceTopoItem, 
  ServiceConfigItem 
} from '../api/config';

// 编辑器标签页类型
export type EditorTab = 'overview' | 'global' | 'nodes' | 'services' | 'preview' | 'generate' | 'export';
