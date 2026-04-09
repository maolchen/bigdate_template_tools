# 大数据平台离线配置生成器

一个基于 Go Template 的配置生成器，用于在无 SSH 互通的网络环境下自动化交付 Hadoop 生态组件。

## 功能特性

- ✅ **Web 可视化配置界面** - 友好的图形化配置界面
- ✅ **REST API** - 完整的 API 支持，可集成到其他系统
- ✅ **命令行模式** - 支持直接生成配置文件
- ✅ **模板引擎** - 基于 Go Template，支持复杂逻辑
- ✅ **多节点管理** - 支持节点复用和服务拓扑配置
- ✅ **配置导出** - 支持单个文件下载和批量打包下载
- ✅ **AI 模板工作台** - 支持对话生成模板、流式回复、历史会话、Skill 管理

## 快速开始

### 1. 命令行模式（生成配置）

```bash
./server-bin
```

这会读取 `config.yaml` 并生成配置文件到 `output/` 目录。

### 2. Web 模式（可视化配置）

```bash
# 默认端口 5000
./server-bin --web

# 指定端口
./server-bin --web --port=8080
```

访问 http://localhost:5000/ 打开配置界面。

## Windows 使用

```powershell
# 命令行模式
server.exe

# Web 模式
server.exe --web
```

## 文件结构

```
project/
├── main.go              # 源代码
├── config.yaml          # 配置文件
├── templates/           # 模板目录
│   ├── hadoop3/
│   ├── kafka/
│   └── ...
├── web/
│   └── dist/            # 前端构建产物
├── server-bin           # Linux/Mac 二进制文件
├── server.exe           # Windows 二进制文件
├── build.sh             # 构建脚本
└── output/              # 输出目录（生成后）
```

## API 接口

### 配置管理
- `GET /api/config` - 获取当前配置
- `PUT /api/config` - 保存配置
- `POST /api/config/reload` - 从文件重新加载配置

### 配置生成
- `POST /api/generate` - 生成配置文件

### 输出管理
- `GET /api/output` - 获取生成的文件列表
- `GET /api/output/download` - 下载配置包（ZIP）
- `GET /api/output/file?path=xxx` - 获取单个文件内容

### 模板管理
- `GET /api/templates` - 获取模板列表

### AI 模板工作台
- `GET /api/ai/settings` / `PUT /api/ai/settings` - AI 接口设置
- `POST /api/ai/settings/test` - 测试连接
- `GET /api/ai/template/session` / `POST /api/ai/template/session` - 历史会话与新建会话
- `POST /api/ai/template/session/:id/message?stream=1` - 流式对话生成模板草稿
- `POST /api/ai/template/session/:id/upload` - 上传附件（文本/图片）
- `POST /api/ai/template/session/:id/save` - 审核后落盘到 `templates/`

## 开发

### 构建前端

```bash
pnpm install
pnpm run build
```

### 编译后端

```bash
# Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server-bin main.go

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o server.exe main.go

# macOS
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o server-darwin main.go
```

### 完整构建

```bash
bash build.sh
```

## 清理

```bash
bash clean.sh
```

## 配置文件说明

配置文件 `config.yaml` 包含以下部分：

- `global` - 全局配置（用户、组、路径等）
- `nodes` - 节点定义（IP、主机名）
- `serviceTop` - 服务拓扑（服务与节点映射）
- `serverConfig` - 服务配置（变量、端口等）
- `nodeOverrides` - 节点特化配置（可选）

详细配置说明请参考 [USAGE_GUIDE.md](USAGE_GUIDE.md)。
AI 工作台说明请参考 [docs/AI_WORKBENCH.md](docs/AI_WORKBENCH.md)。

## 模板开发

模板存放在 `templates/` 目录，按服务名组织：

```
templates/
├── hadoop3/
│   ├── install.sh.tmpl
│   ├── core-site.xml.tmpl
│   └── ...
├── kafka/
│   ├── install.sh.tmpl
│   ├── server.properties.tmpl
│   └── ...
```

### 模板函数

模板支持以下内置函数：

**基础函数：**
- `toUpper`, `toLower`, `trim`, `replace`, `default`

**数学函数：**
- `add`, `sub`, `mul`, `div`

**服务节点函数：**
- `serviceNodes(serviceName)` - 获取服务的所有节点实例
- `serviceEndpoints(serviceName, portField)` - 获取服务端点列表
- `serviceIPs(serviceName)` - 获取服务所有 IP
- `serviceHostnames(serviceName)` - 获取服务所有主机名

**配置访问函数：**
- `serviceVars(serviceName)` - 获取服务配置变量
- `nodeInfo(nodeName)` - 获取节点信息

## 常见问题

### 端口被占用

修改启动端口：

```bash
./server-bin --web --port=8080
```

### 前端无法访问

1. 确保 `web/dist/index.html` 存在
2. 运行 `pnpm run build` 构建前端
3. 检查启动日志中的前端目录路径

### 配置格式错误

1. 使用在线 YAML 验证器检查格式
2. 查看启动日志中的错误信息
3. 参考 `config.yaml` 示例

## 环境变量

- `DEPLOY_RUN_PORT` - 指定服务端口（优先级低于 --port 参数）

## 许可证

MIT
