# AGENTS.md - 项目开发规范与指南

## 📋 项目概述

本项目是一个基于 Go Template 的大数据平台离线配置生成器，用于在无 SSH 互通的网络环境下自动化交付 Hadoop 生态组件。项目提供 Web 可视化配置界面和 REST API，读取 YAML 配置文件，渲染生成各服务的配置文件和安装脚本。

### 核心功能
- **配置管理**：支持全局配置、节点配置、服务拓扑和服务配置的四层分离结构
- **模板渲染**：基于 Go Template 语法，支持条件判断、循环、函数调用等高级功能
- **Web 界面**：提供可视化配置界面，支持 CRUD 操作和实时预览
- **引用检查**：删除前检查 template 引用，防止误删被引用的对象

### 代码架构（v6+）
本项目采用标准 Go 分层架构，将单体代码拆分为多个独立包，提升可维护性和可测试性：

- **config 包**：配置数据结构和加载/保存逻辑
  - `types.go`：定义所有配置相关的结构体（Global、Node、ServiceConfig 等）
  - `config.go`：配置文件的加载和保存

- **generator 包**：配置生成核心逻辑
  - `instance.go`：构建服务实例列表，处理节点特化配置
  - `template.go`：Go Template 渲染引擎，支持丰富的模板函数
  - `output.go`：生成配置文件和安装脚本

- **checker 包**：引用检查逻辑
  - `checker.go`：检查配置对象是否被 template 引用

- **server 包**：HTTP 服务器和 API
  - `server.go`：服务器设置、中间件和启动逻辑
  - `handlers.go`：所有 API 处理函数

- **utils 包**：通用工具函数
  - `utils.go`：ID 格式化、嵌套值获取、工作目录获取等

- **main.go**：程序入口（约 100 行）
  - 解析命令行参数
  - 调用 CLI 模式或 Web 模式

### 架构优势
1. **职责清晰**：每个包专注于单一职责，便于理解和维护
2. **易于测试**：模块化设计便于单元测试和集成测试
3. **可扩展性**：新增功能只需在对应包中添加代码，不影响其他模块
4. **代码复用**：工具函数和配置逻辑可在不同场景下复用

## 🏗️ 项目结构

```
.
├── main.go                      # 主入口文件（约 100 行）
├── go.mod                       # Go 模块依赖
├── go.sum                       # Go 依赖锁定文件
├── config.yaml                  # 主配置文件
├── templates/                   # Go Template 模板目录
│   ├── 服务名称/               # 每个服务一个目录
│   │   ├── install.sh.tmpl     # 安装脚本模板
│   │   ├── start.sh.tmpl       # 启动脚本模板
│   │   └── ...                  # 其他配置文件模板
├── web/                         # 前端源码目录
│   ├── src/
│   │   ├── api/               # API 调用封装
│   │   │   └── config.ts      # 配置相关 API 和类型定义
│   │   ├── components/        # React 组件
│   │   │   ├── ExportPage.tsx      # 导出页面组件
│   │   │   ├── GeneratePage.tsx    # 生成配置页面组件
│   │   │   ├── GlobalConfigPage.tsx # 全局配置页面组件
│   │   │   ├── NodesPage.tsx       # 节点配置页面组件
│   │   │   ├── OverviewPage.tsx    # 概览页面组件
│   │   │   ├── PreviewPage.tsx     # 预览页面组件
│   │   │   ├── ServicesPage.tsx    # 服务配置页面组件
│   │   │   ├── Sidebar.tsx          # 侧边栏导航组件
│   │   │   └── VariablesEditor.tsx  # 变量编辑器组件
│   │   ├── lib/               # 工具库
│   │   │   └── global-fields.ts # 全局字段定义
│   │   ├── types/             # TypeScript 类型定义
│   │   │   └── config.ts      # 配置类型导出
│   │   ├── App.tsx            # 主应用组件
│   │   └── main.tsx           # 应用入口
│   ├── dist/                  # 构建输出（生产环境使用）
│   └── package.json
├── config/                      # 配置管理包
│   ├── types.go               # 配置结构体定义
│   └── config.go              # 配置加载和保存
├── generator/                   # 模板生成包
│   ├── instance.go            # 服务实例构建
│   ├── template.go            # 模板渲染
│   └── output.go              # 输出生成
├── checker/                     # 引用检查包
│   └── checker.go             # Template 引用检查逻辑
├── server/                      # HTTP 服务器包
│   ├── server.go              # 服务器设置和启动
│   └── handlers.go            # API 处理函数
├── utils/                       # 工具函数包
│   └── utils.go               # 通用工具函数
├── docs/                        # 项目文档
│   ├── GLOBAL_VARS_GUIDE.md    # 全局变量复用规范
│   ├── TEMPLATE_MODIFICATIONS.md # 模板修改记录
│   └── REFERENCE_CHECK_VALIDATION.md # 引用检查验证报告
├── .coze                        # 项目配置（构建和运行）
└── AGENTS.md                    # 本文档
```

## 🔧 技术栈

### 后端
- **语言**：Go 1.21
- **依赖**：
  - `gopkg.in/yaml.v3` - YAML 解析
  - `text/template` - Go 模板引擎
  - `net/http` - HTTP 服务

### 前端
- **框架**：React 19 + TypeScript 5
- **构建工具**：Vite 8
- **样式**：Tailwind CSS 4
- **依赖**：
  - `js-yaml` - YAML 格式化和解析
  - `lucide-react` - 图标库
  - `@monaco-editor/react` - 代码编辑器

## 📐 配置结构

### 四层分离架构

#### 1. Global（全局配置）
```yaml
global:
  user: bigdata
  group: bigdata
  install_base_dir: /data/localization
  data_base_dir: /data
  java_home: /data/jdk
```

#### 2. Nodes（节点池）
```yaml
nodes:
  master:
    ip: 192.168.1.100
    hostname: master.hadoop.local
  worker1:
    ip: 192.168.1.101
    hostname: worker1.hadoop.local
```

#### 3. ServiceTop（服务拓扑）
```yaml
serviceTop:
  zookeeper:
    nodes: ["master"]
    description: ZooKeeper 集群
    id_auto_derive: false
  hadoop/hdfs_namenode:
    nodes: ["master"]
    description: HDFS NameNode
    id_auto_derive: true
```

#### 4. ServerConfig（服务配置）
```yaml
serverConfig:
  zookeeper:
    description: ZooKeeper 详细配置
    vars:
      client_port: 2181
      tick_time: 2000
```

## 🎯 核心编码规范

### 1. 职责分离原则（CRITICAL）

#### 后端职责
- **唯一责任**：提供 REST API 接口，处理业务逻辑
- **数据验证**：在 API Handler 中验证请求数据
- **引用检查**：所有引用检查逻辑必须在后端完成
- **模板渲染**：负责 Go Template 的解析和渲染

#### 前端职责
- **UI 交互**：提供用户界面和交互逻辑
- **状态管理**：管理组件状态和表单数据
- **API 调用**：封装 API 调用，处理响应和错误
- **用户反馈**：显示错误提示和成功消息

#### ❌ 禁止模式
```typescript
// ❌ 错误示例：前端做业务判断
const handleDelete = async (key: string) => {
  if (serviceName === '') {
    // 前端判断 serviceName 为空，跳过检查
    return;
  }
  // 调用后端接口
  await checkReferences({ type: 'vars', key, service: serviceName });
}
```

#### ✅ 正确模式
```typescript
// ✅ 正确示例：前端只负责调用接口，不做业务判断
const handleDelete = async (key: string) => {
  // 直接调用后端接口，后端负责所有业务逻辑
  const result = await checkReferences({ type: 'vars', key, service: serviceName || '' });

  if (result.hasReferences) {
    alert('无法删除，变量被引用...');
    return;
  }

  // 执行删除
  await deleteItem(key);
}
```

### 2. API 调用规范

#### 统一错误处理
```typescript
// ✅ 正确示例：统一的错误处理
export async function checkReferences(params: CheckParams): Promise<CheckResult> {
  try {
    const response = await fetch('/api/check-references', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
    });

    if (!response.ok) {
      throw new Error(`检查引用失败: ${response.statusText}`);
    }

    return response.json();
  } catch (error) {
    console.error('[API] checkReferences error:', error);
    throw error; // 重新抛出，让调用者处理
  }
}
```

### 3. 组件设计规范

#### Props 定义
```typescript
// ✅ 正确示例：清晰的 Props 定义
interface VariablesEditorProps {
  vars: Record<string, any>;
  onChange: (vars: Record<string, any>) => void;
  readonly?: boolean;           // 可选参数，使用默认值
  serviceName?: string;          // 可选参数
  isGlobal?: boolean;           // 可选参数
}

export function VariablesEditor({
  vars,
  onChange,
  readonly = false,
  serviceName,
  isGlobal = false,
}: VariablesEditorProps) {
  // ...
}
```

### 4. 日志规范

#### 前端日志
```typescript
// ✅ 正确示例：结构化日志
console.log('[ComponentName] 操作描述', params);
console.warn('[ComponentName] 警告信息', params);
console.error('[ComponentName] 错误信息', error);

// 示例：
console.log('[VariablesEditor] 尝试删除变量:', item.key, 'serviceName:', serviceName);
console.error('[VariablesEditor] 检查引用失败:', err);
```

#### 后端日志
```go
// ✅ 正确示例：结构化日志
fmt.Printf("[HandlerName] 操作描述: 参数1=%s, 参数2=%s\n", val1, val2)
fmt.Printf("[HandlerName] 错误信息: %v\n", err)

// 示例：
fmt.Printf("[CheckReferences] 请求参数: type=%s, key=%s, service=%s\n", req.Type, req.Key, req.Service)
```

## 🔌 核心 API

### 配置管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/config` | 获取完整配置 |
| PUT | `/api/config` | 保存完整配置 |
| POST | `/api/config/reload` | 从文件重新加载配置 |

### 配置生成 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/generate` | 生成配置文件和安装脚本 |
| GET | `/api/output` | 获取生成的文件列表 |
| GET | `/api/output/file?path=xxx` | 获取指定文件内容 |
| GET | `/api/output/download` | 下载生成的文件 |

### 引用检查 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/check-references` | 检查对象是否被 template 引用 |

**请求体**：
```json
{
  "type": "global|node|service|vars",
  "key": "对象名称",
  "service": "服务名称（仅 type 为 service 或 vars 时需要）"
}
```

**响应体**：
```json
{
  "hasReferences": true,
  "references": [
    {
      "path": "template文件路径",
      "service": "服务名称",
      "details": ["引用详情"]
    }
  ]
}
```

### 描述管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/descriptions/global` | 获取全局配置描述 |
| POST | `/api/descriptions/global` | 保存全局配置描述 |
| GET | `/api/descriptions/:serviceName` | 获取服务配置描述 |
| POST | `/api/descriptions/:serviceName` | 保存服务配置描述 |

## 📝 模板编写规范

### 1. 全局变量使用规范

#### 目录配置规范
```yaml
# config.yaml 中（服务配置）
vars:
  data_subdir: "service_name/data"
  install_subdir: "service_name"

# 模板中
DATA_DIR="{{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}"
INSTALL_DIR="{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}"
```

#### 用户/组配置规范
```yaml
# ❌ 错误：服务级别定义
vars:
  run_user: "bigdata"
  run_group: "bigdata"

# ✅ 正确：使用全局变量
# 模板中
RUN_USER="{{ .Global.user }}"
RUN_GROUP="{{ .Global.group }}"
```

#### JAVA_HOME 配置规范
```yaml
# ❌ 错误：服务级别定义
vars:
  java_home: "/data/jdk"

# ✅ 正确：使用全局变量
# 模板中
JAVA_HOME="{{ .Global.java_home }}"
```

### 2. 脚本编写规范

#### 用户权限判断
```bash
# ✅ 正确：添加用户权限判断函数
run_as_root() {
  if [ "$(whoami)" != "root" ]; then
    sudo "$@"
  else
    "$@"
  fi
}

# 需要root权限的操作
run_as_root mkdir -p "{{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}"
```

#### 环境变量配置
```bash
# ✅ 正确：根据用户类型选择配置文件
if [ "{{ .Global.user }}" = "root" ]; then
  echo "export XXX=yyy" >> /etc/profile.d/xxx.sh
else
  echo "export XXX=yyy" >> /home/{{ .Global.user }}/.bash_profile
  echo "export XXX=yyy" >> /etc/profile.d/xxx.sh
fi
```

#### 服务启动规范
```bash
# ✅ 正确：使用全局用户启动
# 非 systemd 服务
su - {{ .Global.user }} -c "xxx start"

# systemd 服务
[Unit]
Description=XXX Service
After=network.target

[Service]
Type=forking
User={{ .Global.user }}
Group={{ .Global.group }}
ExecStart=xxx
ExecStop=xxx
Restart=always

[Install]
WantedBy=multi-user.target
```

### 3. 幂等性要求（CRITICAL）

所有安装脚本必须具备幂等性，即多次执行结果一致：

```bash
# ✅ 正确示例：幂等性操作
# 1. 创建目录（已存在不会报错）
run_as_root mkdir -p "{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}"

# 2. 创建软链接（已存在先删除再创建）
run_as_root rm -f /usr/local/xxx
run_as_root ln -s "{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}" /usr/local/xxx

# 3. 配置文件（先备份再覆盖）
if [ -f /etc/xxx.conf ]; then
  cp /etc/xxx.conf /etc/xxx.conf.bak.$(date +%Y%m%d%H%M%S)
fi
cat > /etc/xxx.conf << 'EOF'
{{ content }}
EOF
```

### 4. 模板文件命名规范

```
服务名称/
├── install.sh.tmpl      # 安装脚本
├── start.sh.tmpl        # 启动脚本
├── stop.sh.tmpl         # 停止脚本
├── check.sh.tmpl        # 健康检查脚本
├── install_binary.sh.tmpl  # 安装二进制包
├── setup_dirs.sh.tmpl   # 创建目录结构
└── config.conf.tmpl     # 配置文件模板
```

## 🚫 常见问题与规范

### 1. 编码混乱问题

#### 问题：前后端职责不清
```typescript
// ❌ 错误：前端做业务判断
if (serviceName === '') {
  return; // 跳过检查
}

// ✅ 正确：后端统一处理
const result = await checkReferences({ type: 'vars', key, service: serviceName || '' });
```

#### 解决方案
- **前端**：只负责 UI 交互和 API 调用
- **后端**：负责所有业务逻辑、数据验证和引用检查

### 2. 类型安全问题

#### 问题：any 类型滥用
```typescript
// ❌ 错误：滥用 any
const vars: Record<string, any> = {};

// ✅ 正确：定义具体类型
interface ServiceVars {
  [key: string]: string | number | boolean | object | null;
}
const vars: Record<string, ServiceVars> = {};
```

### 3. 错误处理不规范

#### 问题：错误被吞掉
```typescript
// ❌ 错误：错误被吞掉
try {
  await apiCall();
} catch (err) {
  console.log(err); // 只打印，不处理
}

// ✅ 正确：错误向上抛出
try {
  await apiCall();
} catch (err) {
  console.error('[API] apiCall error:', err);
  throw err; // 重新抛出
}
```

## 🧪 测试规范

### 1. 功能测试
- 在浏览器中手动测试所有功能
- 使用开发者工具检查网络请求和控制台日志
- 验证错误场景的处理

### 2. 引用检查测试
1. 测试全局变量引用检查
2. 测试节点引用检查
3. 测试服务引用检查
4. 测试服务变量引用检查（包括 serviceName 为空的情况）

### 3. 模板测试
1. 测试全局变量正确使用
2. 测试路径拼接正确性
3. 测试用户/组配置
4. 测试脚本幂等性

## 🔧 Go 开发环境规范

### Go 版本要求
本项目需要 **Go 1.21** 或更高版本。推荐使用 Go 1.21.13。

### 环境变量配置
```bash
# 设置 Go 国内镜像（加速依赖下载）
export GOPROXY=https://goproxy.cn,direct

# 设置 Go 工作目录
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

### 本地安装 Go（Linux/macOS）
```bash
# 下载 Go 1.21.13
wget https://go.dev/dl/go1.21.13.linux-amd64.tar.gz

# 解压到 /usr/local（需要 root 权限）
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz

# 配置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPROXY=https://goproxy.cn,direct' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

### 依赖管理
```bash
# 进入项目目录
cd /workspace/projects

# 下载依赖
go mod download

# 验证依赖完整性
go mod verify

# 整理依赖（移除未使用的依赖）
go mod tidy
```

### 构建二进制文件
```bash
# 构建生产版本
go build -o server-bin main.go

# 交叉编译（例如编译 Linux amd64）
GOOS=linux GOARCH=amd64 go build -o server-bin main.go
```

### 开发调试
```bash
# 直接运行（带调试信息）
go run main.go --web

# 运行测试
go test ./...

# 代码格式化
go fmt ./...

# 代码检查
go vet ./...
```

## 📚 构建和部署

### 开发环境
```bash
# 启动前端开发服务器（热更新）
cd web
pnpm dev

# 启动后端服务（如果已编译）
./server-bin --web
```

### 部署环境
```bash
# 构建前端
cd web
pnpm build

# 启动服务（使用预编译的二进制文件）
./server-bin --web
```

### Docker 部署
```dockerfile
FROM node:24-alpine
WORKDIR /app
COPY . .
RUN cd web && pnpm install && pnpm build
CMD ["./server-bin", "--web"]
```

## 📖 参考文档

- [全局变量复用规范](docs/GLOBAL_VARS_GUIDE.md)
- [模板修改记录](docs/TEMPLATE_MODIFICATIONS.md)
- [使用指南](USAGE_GUIDE.md)
- [本地部署指南](LOCAL_DEPLOYMENT.md)
- [Windows 部署指南](WINDOWS_DEPLOYMENT.md)

## 🔄 版本历史

### v6（当前版本）
- **重构**：将单体 main.go（1381 行）拆分为标准分层架构
  - config 包：配置结构定义和加载/保存
  - generator 包：服务实例构建、模板渲染、输出生成
  - checker 包：Template 引用检查逻辑
  - server 包：HTTP 服务器和 API 处理
  - utils 包：通用工具函数
  - main.go：仅保留主入口（约 100 行）
- **优化**：模块化设计，提升代码可维护性和可测试性
- **验证**：所有功能测试通过，API 接口正常工作

### v5.1
- **新增**：Go 开发环境规范，明确版本要求和依赖管理
- **修复**：main.go 语法错误（缺失闭合大括号）
- **验证**：所有删除按钮的引用检查逻辑符合规范
- **验证**：嵌套变量引用检查正常工作（如 `mysql.vars.port`）
- **完善**：文档结构，补充完整的组件列表

### v5
- 实现四层分离结构（global + nodes + serviceTop + serverConfig）
- 实现删除前的 template 引用检查
- 修复服务配置页面变量删除的引用检查逻辑
- 添加详细的调试日志
- **新增**：前后端职责分离规范
- **新增**：模板编写规范
- **新增**：编码规范约束

### v2
- 三层定义结构
- 基础 Web 界面

### v1
- 初始版本
- 命令行工具
