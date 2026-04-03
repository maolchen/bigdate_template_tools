// 全局配置字段说明
export interface GlobalFieldDescription {
  key: string;
  name: string;
  label: string;
  description: string;
  type: 'string' | 'number' | 'boolean';
  category: string;
  required: boolean;
}

// 全局配置字段定义
export const GLOBAL_FIELD_DEFINITIONS: GlobalFieldDescription[] = [
  // 基础目录
  {
    key: 'install_base_dir',
    name: '安装基础目录',
    label: '安装基础目录',
    description: '所有软件安装的基础根目录',
    type: 'string',
    category: '基础目录',
    required: true
  },
  {
    key: 'data_base_dir',
    name: '数据基础目录',
    label: '数据基础目录',
    description: '所有数据存储的基础根目录',
    type: 'string',
    category: '基础目录',
    required: true
  },
  {
    key: 'temp_dir',
    name: '临时目录',
    label: '临时目录',
    description: '安装过程中的临时文件目录',
    type: 'string',
    category: '基础目录',
    required: false
  },
  {
    key: 'software_dir',
    name: '软件包目录',
    label: '软件包目录',
    description: '软件包（tar.gz 等）的存放目录',
    type: 'string',
    category: '基础目录',
    required: false
  },
  {
    key: 'pkg_pro_dir',
    name: '项目包目录',
    label: '项目包目录',
    description: '项目特定的软件包目录',
    type: 'string',
    category: '基础目录',
    required: false
  },

  // JDK 配置
  {
    key: 'jdk_version',
    name: 'JDK 版本',
    label: 'JDK 版本',
    description: '安装的 JDK 版本号',
    type: 'string',
    category: 'JDK 配置',
    required: false
  },
  {
    key: 'jdk_path',
    name: 'JDK 路径',
    label: 'JDK 路径',
    description: 'JDK 安装路径',
    type: 'string',
    category: 'JDK 配置',
    required: false
  },
  {
    key: 'java_home',
    name: 'JAVA_HOME',
    label: 'JAVA_HOME',
    description: 'JAVA_HOME 环境变量路径',
    type: 'string',
    category: 'JDK 配置',
    required: true
  },

  // 系统用户
  {
    key: 'user',
    name: '运行用户',
    label: '运行用户',
    description: '运行大数据服务的系统用户',
    type: 'string',
    category: '系统配置',
    required: true
  },
  {
    key: 'group',
    name: '运行用户组',
    label: '运行用户组',
    description: '运行大数据服务的系统用户组',
    type: 'string',
    category: '系统配置',
    required: true
  },
  {
    key: 'timezone',
    name: '时区',
    label: '时区',
    description: '服务器时区设置',
    type: 'string',
    category: '系统配置',
    required: false
  },
  {
    key: 'pkg_base_dir',
    name: '软件包基础目录',
    label: '软件包基础目录',
    description: '软件包下载和存放的基础目录',
    type: 'string',
    category: '基础目录',
    required: false
  },
  {
    key: 'log_base_dir',
    name: '日志基础目录',
    label: '日志基础目录',
    description: '日志文件的基础目录',
    type: 'string',
    category: '基础目录',
    required: false
  }
];

// 根据配置项获取说明
export function getFieldDescription(key: string): GlobalFieldDescription | undefined {
  return GLOBAL_FIELD_DEFINITIONS.find(field => field.key === key);
}

// 获取所有分类
export function getCategories(): string[] {
  const categories = new Set(GLOBAL_FIELD_DEFINITIONS.map(f => f.category));
  return Array.from(categories);
}

// 按分类获取字段定义
export function getFieldsByCategory(category: string): GlobalFieldDescription[] {
  return GLOBAL_FIELD_DEFINITIONS.filter(f => f.category === category);
}
