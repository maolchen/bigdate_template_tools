# 大数据平台脚本生成工具

一个面向公司内部交付场景的离线配置与脚本生成工具。

项目基于 `config.yaml + templates/` 工作，提供 Web 配置界面、AI 模板工作台、模板在线编辑器，以及按用户隔离的配置与输出目录，适用于 Hadoop 生态及其他大数据服务的标准化交付。

## 当前能力

- 用户登录与权限控制
  - 仅支持 `admin` / `user` 两种角色
  - 管理员同一时间只允许一个有效登录会话
- 配置管理
  - 全局配置
  - 节点管理
  - 服务拓扑
  - 服务配置
- 用户空间隔离
  - 每个用户拥有自己的 `config.yaml`
  - 每个用户拥有自己的输出目录
  - 每个用户可维护多份“配置模板”备份
- 主配置增量同步
  - 用户登录后可感知根 `config.yaml` 是否更新
  - 支持同步新增项
  - 支持危险操作：按主配置清理删除项
- 配置模板管理
  - 当前主模板固定表现为 `config.yaml`
  - 备份模板以 `<id>_config.yaml` 保存在用户目录
  - 切换模板采用“主模板/备份模板互换”方式
- 配置生成
  - 基于 `config.yaml + templates/` 渲染输出
  - 输出目录按用户隔离
- AI 模板工作台
  - AI 会话、流式回复、附件上传、草稿管理
  - AI 可生成模板草稿、配置补丁建议、计划动作
  - 仅管理员允许保存模板到 `templates/`
- 模板编辑器
  - 独立页面打开
  - 管理员可编辑 `templates/`，普通用户只读
  - 支持黑白模式、目录树操作、右键菜单

## 当前实现边界

- 面向内部单机场景，不引入数据库
- 认证基于文件存储与 Cookie 会话
- 正式模板目录仍为 `templates/`
- AI 生成的模板仍然落盘到 `templates/`，未做模板版本库
- `config.yaml` 仍然是生成的核心输入，只是在 Web 场景下按用户复制到用户空间使用

## 目录结构

```text
.
├── main.go
├── config.yaml
├── templates/
├── output/
├── data/
│   ├── auth/
│   │   └── sessions.json
│   ├── users/
│   │   ├── users.json
│   │   └── <username>/
│   │       ├── configs/
│   │       │   ├── config.yaml
│   │       │   ├── <id>_config.yaml
│   │       │   └── templates_index.json
│   │       └── meta.json
│   └── ai/
│       ├── settings.json
│       ├── template_rules.md
│       ├── examples_index.json
│       ├── skills/
│       ├── sessions/
│       └── uploads/
├── config/
├── generator/
├── checker/
├── server/
├── utils/
└── web/
```

## 核心架构

### 后端

- `config/`
  - 配置结构定义与 YAML 读写
- `generator/`
  - 构建服务实例
  - Go Template 函数与渲染
  - 输出生成
- `checker/`
  - 删除前模板引用检查
- `server/`
  - 认证与用户管理
  - 用户配置模板管理
  - 配置读写与主配置同步
  - AI 工作台接口
  - 模板编辑器接口
- `utils/`
  - 路径、ID、嵌套变量等通用工具

### 前端

- `App.tsx`
  - 登录态初始化
  - 页面切换
  - 主配置版本轮询
- `web/src/components/`
  - 概览、全局配置、配置管理、AI 模板、生成、预览、导出、模板编辑器等页面组件
- `web/src/api/`
  - 按模块封装接口调用

## 主要页面

- `概览`
- `全局配置`
- `配置管理`
  - 模板管理
  - 节点管理
  - 服务拓扑 / 服务配置
- `AI模板`
- `模板编辑器`
- `生成配置`
- `YAML预览`
- `导出配置`
- `系统设置`
  - 修改密码
  - 用户管理（仅管理员）
  - 退出系统

## 快速开始

### 1. 安装前端依赖

```bash
# node v20.19.0
pnpm install
```

### 2. 启动 Web 服务

```bash
# go1.22.6
go run main.go --web
```

或：

```bash
./server.exe --web
```

默认地址：

- Web: [http://localhost:5000/](http://localhost:5000/)
- API: [http://localhost:5000/api/config](http://localhost:5000/api/config)

### 3. 默认管理员账号

首次启动会自动初始化：

- 用户名：`admin`
- 密码：`Admin@123`

## 命令行模式

直接生成输出：

```bash
go run main.go
```

命令行模式会直接读取根目录 `config.yaml` 与 `templates/`，输出到 `output/`。

## 构建

### 前端构建

```bash
pnpm --dir web build
```

### 后端构建

```bash
go build -o server.exe main.go
```

## 测试

```bash
go test ./...
pnpm --dir web build
```

说明：当前仓库存在部分既有 AI 单测失败时，优先以具体失败日志为准，不代表配置/模板主链路不可用。

## 配置与渲染模型

核心配置结构：

- `global`
- `nodes`
- `serviceTop`
- `serverConfig`
- `nodeOverrides`

生成流程：

1. 读取配置
2. 构建服务实例列表
3. 扫描 `templates/<service>/*.tmpl`
4. 渲染到 `output/users/<username>/<nodeIP>/<service>/...`

## 模板变量模型

运行时模板上下文位于 `config.Context`：

- `.Global`
- `.Nodes`
- `.Instance.ServiceName`
- `.Instance.NodeName`
- `.Instance.Node.IP`
- `.Instance.Node.Hostname`
- `.Instance.Node.NodeName`
- `.Instance.Vars`
- `.Instance.AutoID`
- `.AllInstances`

常用内置函数位于 `generator/template.go`，例如：

- `serviceNodes`
- `serviceEndpoints`
- `serviceIPs`
- `serviceHostnames`
- `serviceVars`
- `serviceVar`
- `nodeInfo`
- `join`
- `default`

## 文档

- [需求设计文档](docs/REQUIREMENTS_DESIGN.md)
- [接口文档](docs/API_REFERENCE.md)
- [调用链图](docs/CALL_CHAIN.md)
- [AI 工作台说明](docs/AI_WORKBENCH.md)
- [AI 模板使用规范](docs/AI_TEMPLATE_USAGE_GUIDE.md)
- [全局变量复用规范](docs/GLOBAL_VARS_GUIDE.md)

## 注意事项

- 所有 `/api/*` 认证接口依赖 Cookie 会话
- 模板在线编辑与 AI 模板保存均要求管理员权限
- 普通用户保存配置时只影响自己的用户空间
- 管理员保存根配置后会同步增量更新到普通用户空间
- 普通用户看到的“主模板”本质上始终是自己空间下的 `config.yaml`
