# AGENTS.md - 项目概览与开发指南

## 项目概述

本项目是一个基于 Go Template 的大数据平台离线配置生成器，用于在无 SSH 互通的网络环境下自动化交付 Hadoop 生态组件。项目提供 Web 可视化配置界面和 REST API，读取 YAML 配置文件，渲染生成各服务的配置文件和安装脚本。

### 核心功能
- 配置管理：支持全局配置、节点配置、服务拓扑和服务配置的四层分离结构
- 模板渲染：基于 Go Template 语法，支持条件判断、循环、函数调用等高级功能
- Web 界面：提供可视化配置界面，支持 CRUD 操作和实时预览
- 引用检查：删除前检查 template 引用，防止误删被引用的对象

## 技术栈

### 后端
- Go 1.21
- gopkg.in/yaml.v3 (YAML 解析)
- text/template (模板引擎)
- net/http (HTTP 服务)

### 前端
- Vite + React + TypeScript
- js-yaml (YAML 格式化)
- Tailwind CSS (样式)

### 配置格式
- YAML (主配置文件：config.yaml)
- Go Template (.tmpl 文件)

## 项目结构

```
.
├── main.go                 # Go 后端主入口（单体架构，约1374行）
├── config.yaml             # 主配置文件
├── templates/              # Go Template 模板目录
│   ├── service1/          # 服务1的模板
│   └── service2/          # 服务2的模板
├── web/                   # 前端源码目录
│   ├── src/
│   │   ├── components/   # React 组件
│   │   │   ├── VariablesEditor.tsx      # 变量编辑器
│   │   │   ├── ServicesPage.tsx        # 服务配置页面
│   │   │   ├── GlobalConfigPage.tsx    # 全局配置页面
│   │   │   └── NodesPage.tsx           # 节点配置页面
│   │   └── api/        # API 调用
│   │       └── config.ts # 配置相关 API
│   ├── dist/            # 构建输出
│   └── package.json
├── .coze                 # 项目配置（构建和运行）
└── AGENTS.md             # 本文档
```

## 配置结构（v5）

项目采用四层分离结构，职责清晰：

### 1. Global（全局配置）
```yaml
global:
  user: bigdata
  group: bigdata
  install_base_dir: /data/localization
  data_base_dir: /data
  java_home: /data/jdk
```

### 2. Nodes（节点池）
```yaml
nodes:
  master:
    ip: 192.168.1.100
    hostname: master.hadoop.local
  worker1:
    ip: 192.168.1.101
    hostname: worker1.hadoop.local
```

### 3. ServiceTop（服务拓扑）
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

### 4. ServerConfig（服务配置）
```yaml
serverConfig:
  zookeeper:
    description: ZooKeeper 详细配置
    vars:
      client_port: 2181
      tick_time: 2000
```

## 核心 API

### 配置管理
- `GET /api/config` - 获取完整配置
- `PUT /api/config` - 保存完整配置
- `POST /api/config/reload` - 从文件重新加载配置

### 配置生成
- `POST /api/generate` - 生成配置文件和安装脚本
- `GET /api/output` - 获取生成的文件列表
- `GET /api/output/file?path=xxx` - 获取指定文件内容
- `GET /api/output/download` - 下载生成的文件

### 引用检查
- `POST /api/check-references` - 检查对象是否被 template 引用

请求体：
```json
{
  "type": "global|node|service|vars",
  "key": "对象名称",
  "service": "服务名称（仅type为service或vars时需要）"
}
```

响应体：
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

### 描述管理
- `GET /api/descriptions/global` - 获取全局配置描述
- `POST /api/descriptions/global` - 保存全局配置描述
- `GET /api/descriptions/:serviceName` - 获取服务配置描述
- `POST /api/descriptions/:serviceName` - 保存服务配置描述

## 引用检查机制

### 检查逻辑

系统在删除对象前会检查该对象是否被 template 引用，防止误删导致配置生成失败。

#### 1. Global 变量引用检查
- 检查模式：`.Global.xxx`
- 示例：删除 `global.user` 会检查 template 中是否有 `.Global.user` 引用

#### 2. Node 引用检查
- 检查模式：
  - `.Instance.Node.NodeName === "nodeName"`
  - `serviceNodes "nodeName"`
- 示例：删除节点 `master` 会检查 template 中是否有对该节点的引用

#### 3. Service 引用检查
- 检查模式：
  - `serviceNodes "serviceName"`
  - `serviceEndpointsJoin "serviceName"`
  - 该服务有 template 目录
- 示例：删除服务 `zookeeper` 会检查其他 template 是否引用了该服务

#### 4. Vars 变量引用检查（关键修复）
- 检查模式：`.Instance.Vars.xxx`
- **修复前问题**：当 `serviceName` 为空时（如添加新服务配置时），后端无法正确匹配变量引用
- **修复后逻辑**：
  - 如果指定了 `service`，只检查该服务的 template 文件
  - 如果 `service` 为空，检查所有 template 文件（覆盖所有可能的情况）
- 示例：删除服务变量 `client_port` 会检查 template 中是否有 `.Instance.Vars.client_port` 引用

### 引用检查使用场景

#### 前端删除流程
```typescript
const handleDelete = async (index: number) => {
  const item = items[index];

  // 检查是否被template引用
  let checkResult;
  if (isGlobal) {
    checkResult = await checkReferences({ type: 'global', key: item.key });
  } else if (serviceName) {
    checkResult = await checkReferences({ type: 'vars', key: item.key, service: serviceName });
  } else {
    // serviceName为空，跳过引用检查，直接允许删除
    checkResult = { hasReferences: false };
  }

  if (checkResult && checkResult.hasReferences) {
    alert('无法删除，变量被以下template引用...');
    return;
  }

  // 执行删除操作
};
```

#### 后端引用检查实现（main.go）
```go
func webCheckReferencesHandler(w http.ResponseWriter, r *http.Request) {
    // 解析请求
    var req struct {
        Type    string `json:"type"`    // "global", "node", "service", "vars"
        Key     string `json:"key"`     // 对象名称
        Service string `json:"service"` // 服务名称（仅type为service或vars时需要）
    }
    // ...

    // 遍历所有template文件，检查引用
    switch req.Type {
    case "vars":
        // 关键修复：处理serviceName为空的情况
        if req.Service == "" || filepath.Dir(relPath) == req.Service || filepath.Dir(relPath) == "." {
            pattern := fmt.Sprintf(`\.Instance\.Vars\.%s`, req.Key)
            isReferenced = regexp.MustCompile(pattern).MatchString(contentStr)
            if isReferenced {
                if req.Service != "" {
                    details = append(details, fmt.Sprintf("引用服务变量: .Instance.Vars.%s (在服务 %s 中)", req.Key, req.Service))
                } else {
                    details = append(details, fmt.Sprintf("引用服务变量: .Instance.Vars.%s (在 %s 中)", req.Key, relPath))
                }
            }
        } else if req.Service != "" {
            // 检查其他服务是否间接引用了该服务的变量
            // ...
        }
    }
}
```

## 模板语法示例

### 基础引用
```go
# 全局配置
{{ .Global.user }}

# 节点信息
{{ .Instance.Node.IP }}

# 服务变量
{{ .Instance.Vars.client_port }}
```

### 条件判断
```go
{{ if .Instance.AutoID }}
ID: {{ .Instance.AutoID }}
{{ else }}
ID: {{ .Vars.id }}
{{ end }}
```

### 循环
```go
{{ range serviceNodes "zookeeper" }}
{{ . }}:2181
{{ end }}
```

### 函数调用
```go
{{ serviceEndpointsJoin "hdfs_namenode" "," }}
```

## 开发规范

### 代码规范
1. Go 代码遵循 Go 标准规范，使用 `gofmt` 格式化
2. React/TypeScript 代码使用 ESLint 检查
3. 配置文件使用 YAML 格式，缩进使用 2 空格

### 提交规范
- feat: 新功能
- fix: 修复 bug
- refactor: 重构
- docs: 文档
- test: 测试

### 调试
- 前端日志：浏览器控制台（console.log）
- 后端日志：标准输出（fmt.Printf）
- 服务日志：`/app/work/logs/bypass/app.log`、`/app/work/logs/bypass/console.log`

## 常见问题

### 1. 变量删除未弹出引用检查警告
**原因**：在添加新服务配置时，`serviceName` 为空，导致后端无法正确匹配变量引用。

**解决方案**：
- 后端修复：在 `webCheckReferencesHandler` 中，当 `req.Service == ""` 时，检查所有 template 文件
- 前端修复：在 `VariablesEditor` 中，当 `serviceName` 为空时，跳过引用检查或检查所有 template

**代码位置**：
- 后端：main.go 第 1025-1053 行
- 前端：web/src/components/VariablesEditor.tsx 第 123-162 行

### 2. 项目结构混乱
**问题**：Go 后端代码误放入 web/src 目录。

**解决方案**：
- Go 后端代码应放在项目根目录（main.go）
- 前端代码应放在 web/src 目录
- 使用 `.coze` 文件定义构建和运行方式

### 3. 缺少调试日志
**解决方案**：
- 在关键操作前后添加 console.log（前端）或 fmt.Printf（后端）
- 记录请求参数、响应结果、错误信息

## 构建和运行

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

## 测试

### 手动测试
1. 启动服务：`./server-bin --web`
2. 访问 Web 界面：`http://localhost:5000`
3. 测试删除功能：
   - 全局配置：删除一个被引用的全局变量，应弹出警告
   - 服务配置：删除一个被引用的服务变量，应弹出警告
   - 节点配置：删除一个被引用的节点，应弹出警告
   - 服务：删除一个被引用的服务，应弹出警告

### 引用检查测试用例
1. 测试全局变量引用检查
2. 测试节点引用检查
3. 测试服务引用检查
4. 测试服务变量引用检查（包括 serviceName 为空的情况）

## 版本历史

### v5（当前版本）
- 实现四层分离结构（global + nodes + serviceTop + serverConfig）
- 实现删除前的 template 引用检查
- 修复服务配置页面变量删除的引用检查逻辑
- 添加详细的调试日志

### v2
- 三层定义结构
- 基础 Web 界面

### v1
- 初始版本
- 命令行工具

## 联系方式

- 项目负责人：[待补充]
- 文档维护：[待补充]
- 问题反馈：[待补充]
