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

- `web/src/App.tsx`
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
  - 服务拓扑
  - 服务配置
- `AI模板`
- `模板编辑器`
- `生成配置`
- `YAML预览`
- `导出配置`
- `系统设置`
  - 修改密码
  - 用户管理
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
#node20.19.0
pnpm --dir web build
```

### 后端构建

```bash
#go1.22.6
go build -o server_bin main.go
```

## 测试

```bash
go test ./...
pnpm --dir web build
```

## 运行
```bash
export CONFIG_GENERATOR_AI_MASTER_KEY="AES_KEY"  #AES_KEY 为AI API的加解密key，保存api key是密文保存的
./server_bin --web --port 5000    #默认5000端口
```

说明：当前仓库存在部分既有 AI 单测失败时，优先以具体失败日志为准，不代表配置/模板主链路不可用。

## 配置文件详解

当前核心配置结构见 [config/types.go](config/types.go)：

- `global`
- `nodes`
- `serviceTop`
- `serverConfig`
- `nodeOverrides`

### 1. `global`

放全局复用变量，供所有服务模板共享。

示例：

```yaml
global:
  user: bigdata
  group: bigdata
  install_base_dir: /data/localization
  data_base_dir: /data/bigdata
  log_base_dir: /data/logs
  software_dir: /data/software
  java_home: /data/jdk
```

适合放：

- 用户与用户组
- 安装基目录
- 数据基目录
- 日志基目录
- 软件包目录
- JDK 路径

模板使用示例：

```gotemplate
{{ .Global.user }}
{{ .Global.install_base_dir }}
{{ .Global.software_dir }}
```

### 2. `nodes`

节点池，键名为节点别名。

示例：

```yaml
nodes:
  master1:
    ip: 192.168.10.11
    hostname: master1.hadoop.local
  master2:
    ip: 192.168.10.12
    hostname: master2.hadoop.local
  worker1:
    ip: 192.168.10.21
    hostname: worker1.hadoop.local
```

说明：

- 节点别名用于 `serviceTop.nodes`
- 运行时模板中推荐使用 `.Instance.Node.IP`、`.Instance.Node.Hostname`
- 同一个 IP 理论上可以对应多个节点别名

### 3. `serviceTop`

定义服务拓扑，决定服务部署到哪些节点。

示例：

```yaml
serviceTop:
  zookeeper:
    nodes: [master1, master2, master3]
    id_auto_derive: true
  server_init:
    nodes: ["*"]
    id_auto_derive: false
```

说明：

- `nodes` 是节点别名列表
- `nodes: ["*"]` 或 `["all"]` 表示所有节点
- `id_auto_derive: true` 时，生成实例会自动填充 `.Instance.AutoID`
- 是否真的生成 `AutoID` 还取决于对应 `serverConfig.<service>` 中是否配置了 `id_field`

### 4. `serverConfig`

定义服务级配置与变量。

示例：

```yaml
serverConfig:
  zookeeper:
    type: ""
    description: ZooKeeper 集群
    id_field: myid
    id_format: index+1
    vars:
      version: 3.8.4
      install_subdir: zookeeper
      data_subdir: zookeeper/data
      log_subdir: zookeeper/logs
      client_port: 2181
```

字段说明：

- `type`
  - 当前用于区分特殊服务类型，常见值为空字符串或 `global`
- `description`
  - 服务描述，主要供界面与说明使用
- `id_field`
  - 自动 ID 对应的逻辑字段名，例如 `myid`
- `id_format`
  - 自动 ID 的格式规则
- `vars`
  - 服务实际渲染变量来源，对应模板中的 `.Instance.Vars`

### 5. `nodeOverrides`

节点级覆盖变量，用于某个服务在不同节点上的局部差异。

示例：

```yaml
nodeOverrides:
  worker1:
    elasticsearch:
      heap_size: 8g
      data_subdir: elasticsearch/data01
```

说明：

- 键路径是 `nodeOverrides.<nodeName>.<service>.<varKey>`
- 会覆盖对应节点上该服务的 `serverConfig.vars`
- 适合处理单节点磁盘路径、堆大小、端口差异

## 渲染模型说明

当前渲染逻辑见：

- [generator/instance.go](generator/instance.go)
- [generator/template.go](generator/template.go)
- [generator/output.go](generator/output.go)

### 生成流程

1. 读取当前配置文件
2. 遍历 `serviceTop`
3. 构建 `ServiceInstance`
4. 合并 `serverConfig.vars` 与 `nodeOverrides`
5. 扫描 `templates/<service>/*.tmpl`
6. 对每个节点、每个服务实例渲染输出
7. 输出到用户目录或命令行默认目录

### 运行时模板上下文

模板上下文位于 `config.Context`：

- `.Global`
- `.Nodes`
- `.Instance`
- `.AllInstances`

其中 `.Instance` 结构包含：

- `.Instance.ServiceName`
- `.Instance.NodeName`
- `.Instance.Node.IP`
- `.Instance.Node.Hostname`
- `.Instance.Node.NodeName`
- `.Instance.Vars`
- `.Instance.AutoID`

示例：

```gotemplate
SERVICE_NAME={{ .Instance.ServiceName }}
NODE_NAME={{ .Instance.NodeName }}
NODE_IP={{ .Instance.Node.IP }}
NODE_HOSTNAME={{ .Instance.Node.Hostname }}
INSTALL_DIR={{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}
```

## 内置模板函数说明

当前函数定义以 [generator/template.go](generator/template.go) 为准。

### 基础函数

- `join`
- `split`
- `default`
- `toUpper`
- `toLower`
- `trim`
- `replace`

### 服务查询函数

#### `serviceNodes(serviceName)`

返回某个服务的全部实例，类型为 `[]config.ServiceInstance`。

示例：

```gotemplate
{{ range serviceNodes "zookeeper" }}
server.{{ .AutoID }}={{ .Node.IP }}:2888:3888
{{ end }}
```

#### `serviceIPs(serviceName)`

返回某个服务的全部实例 IP 列表，类型为 `[]string`。

示例：

```gotemplate
{{ join "," (serviceIPs "zookeeper") }}
```

#### `serviceHostnames(serviceName)`

返回某个服务的全部实例主机名列表，类型为 `[]string`。

示例：

```gotemplate
{{ join "," (serviceHostnames "zookeeper") }}
```

#### `serviceEndpoints(serviceName, portField)`

根据服务名和端口字段，返回 `ip:port` 列表，类型为 `[]string`。

示例：

```gotemplate
{{ join "," (serviceEndpoints "zookeeper" "client_port") }}
{{ join "," (serviceEndpoints "elasticsearch" "transport_port") }}
```

#### `serviceEndpointsJoin(serviceName, portField)`

与 `serviceEndpoints` 类似，但直接返回逗号拼接后的字符串。

示例：

```gotemplate
{{ serviceEndpointsJoin "zookeeper" "client_port" }}
```

#### `getServiceNodes(serviceName)`

返回某个服务涉及到的节点别名列表，去重后输出 `[]string`。

示例：

```gotemplate
{{ join "," (getServiceNodes "hdfs_namenode") }}
```

#### `serviceVars(serviceName)`

返回某个服务在 `serverConfig.<service>.vars` 下的全部变量。

示例：

```gotemplate
{{ index (serviceVars "zookeeper") "client_port" }}
```

#### `serviceVar(serviceName, varName)`

读取某个服务单个变量。

示例：

```gotemplate
{{ serviceVar "zookeeper" "client_port" }}
{{ default "2181" (serviceVar "zookeeper" "client_port") }}
```

注意：

- 当前实现只有两个参数
- 不支持 `{{ serviceVar "zookeeper" "client_port" "2181" }}`
- 需要默认值时，应配合 `default`

#### `serviceConfig(serviceName)`

返回完整服务配置对象 `serverConfig.<service>`。

示例：

```gotemplate
{{ (serviceConfig "zookeeper").Description }}
```

#### `nodeInfo(nodeName)`

按节点别名返回 `nodes.<nodeName>`。

示例：

```gotemplate
{{ with nodeInfo "master1" }}
{{ .IP }} {{ .Hostname }}
{{ end }}
```

## 模板编写说明

### 1. 模板目录规范

模板路径必须位于：

```text
templates/<service>/*.tmpl
```

例如：

```text
templates/elasticsearch/install.sh.tmpl
templates/elasticsearch/elasticsearch.yml.tmpl
templates/elasticsearch/check.sh.tmpl
```

### 2. 输出目录规范

当前输出目录由代码自动决定，默认结构为：

```text
output/users/<username>/<nodeIP>/<service>/
```

命令行模式通常输出到：

```text
output/<nodeIP>/<service>/
```

因此模板内容应默认理解为：

- 安装脚本与配置文件会出现在 `<nodeIP>/<service>/`
- 不要自行假设额外存在 `config/` 子目录，除非你的模板文件本身路径就设计成子目录输出

### 3. 变量使用规范

优先使用：

```gotemplate
{{ .Global.user }}
{{ .Global.group }}
{{ .Global.install_base_dir }}
{{ .Global.data_base_dir }}
{{ .Instance.Node.IP }}
{{ .Instance.Node.Hostname }}
{{ .Instance.Node.NodeName }}
{{ .Instance.Vars.install_subdir }}
{{ .Instance.Vars.data_subdir }}
```

避免使用不存在或历史错误变量：

```gotemplate
{{ .Instance.NodeAlias }}
{{ .Instance.Node.HostName }}
{{ .Instance.Hostname }}
{{ .Node.Hostname }}
```

### 4. 路径拼接建议

推荐方式：

```gotemplate
INSTALL_DIR={{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}
DATA_DIR={{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}
LOG_DIR={{ .Global.log_base_dir }}/{{ .Instance.Vars.log_subdir }}
```

不建议在模板里写死：

```gotemplate
/data/elasticsearch
/opt/elasticsearch
```

除非该服务明确要求固定路径，且你已经在说明里写清楚。

### 5. 脚本编写建议

安装脚本建议满足：

- 幂等
- 路径可配置
- 用户和组取自全局变量
- 配置文件生成后可以被安装脚本复制到服务真实目录
- 尽量避免写死端口、路径和账号

推荐模式：

```bash
run_as_root mkdir -p "{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}"
run_as_root mkdir -p "{{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}"
run_as_root chown -R {{ .Global.user }}:{{ .Global.group }} "{{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}"
```

### 6. 集群模板写法建议

集群成员相关配置优先使用全局服务函数，而不是写死节点。

例如：

```gotemplate
zookeeper.connect={{ join "," (serviceEndpoints "zookeeper" "client_port") }}
cluster.initial_master_nodes={{ join "," (serviceHostnames "elasticsearch") }}
discovery.seed_hosts={{ join "," (serviceEndpoints "elasticsearch" "transport_port") }}
```

### 7. 配置文件与安装脚本配合

如果模板生成配置文件，通常还需要在安装脚本中：

- 备份原始配置
- 把生成后的配置复制到服务目录
- 修改权限
- 重载 systemd 或重启服务

示例：

```bash
run_as_root cp ./elasticsearch.yml "{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}/config/elasticsearch.yml"
run_as_root chown {{ .Global.user }}:{{ .Global.group }} "{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}/config/elasticsearch.yml"
```

## AI 模板编写建议

如果通过 AI 生成模板，建议在提问时写清楚：

- 服务名
- 服务版本
- 安装包路径
- 安装目录
- 数据目录
- 日志目录
- 运行用户和用户组
- 配置文件是否需要替换到实际目录
- 最终输出目录结构要求
- 需要补哪些 `serviceTop` 与 `serverConfig.vars`

推荐搭配：

- `core/template_contract`
- `core/variable_contract`
- `task/install_script`
- `task/config_file`
- `task/systemd_service`

更完整的 AI 使用规范见：

- [AI 模板使用规范](docs/AI_TEMPLATE_USAGE_GUIDE.md)

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
- 模板函数说明必须以当前代码实现为准，不要沿用历史旧变量名或历史函数签名
