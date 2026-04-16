# AI 模板使用规范

本文档用于规范 `AI模板` 页面生成、修改、审核和保存 `templates/` 模板的方式，目标是减少变量乱用、路径错误、配置补丁不完整、保存后不可渲染等问题。

## 1. 使用边界

`AI模板` 适合做：

- 新服务模板初稿生成
- 已有模板批量改写
- 根据安装包、官方文档、历史脚本生成安装/启动/停止/检查脚本
- 根据模板中实际使用到的变量，补充 `serverConfig.vars` 建议
- 根据服务部署方式，补充 `serviceTop` 建议

`AI模板` 不适合做：

- 不经人工审核直接发布生产模板
- 直接修改普通用户项目配置
- 让 AI 猜测未知安装目录、端口、账号密码
- 一次性生成过多文件，导致输出截断或审核困难

## 2. 权限规则

- 管理员可以生成草稿、编辑草稿、保存模板到 `templates/`
- 普通用户可以生成草稿和查看内容，但不能保存到正式 `templates/`
- `保存全部` / `保存当前` 只负责把草稿写入模板目录
- `保存并同步配置` 会尝试根据 AI 返回的配置补丁同步 `config.yaml`，但仍需要人工确认 `serviceTop` 和 `serverConfig` 是否合理

## 3. 推荐工作流

### 3.1 第一次生成某个服务模板

1. 准备基础信息：
   - 服务名，例如 `elasticsearch`、`spark3`
   - 安装包路径，例如 `/data/software/elasticsearch-8.12.2-linux-x86_64.tar.gz`
   - 安装目录
   - 数据目录、日志目录、配置文件目录
   - 部署角色和节点分布方式
   - 需要生成哪些模板文件
2. 在提问框中明确说明：
   - 服务如何部署
   - 模板最终输出目录是什么
   - 安装脚本是否需要替换默认配置文件
   - 需要补哪些 `serviceTop` / `serverConfig.vars`
3. 人工审核 AI 返回的每个草稿文件
4. 必要时进入编辑模式手工修订
5. 管理员执行 `保存全部` 或 `保存当前`
6. 若需要补配置，再执行 `保存并同步配置`
7. 到 `配置管理` 页面检查：
   - `serviceTop.<service>` 是否存在
   - `serverConfig.<service>.vars` 是否完整
   - 节点和变量值是否符合当前项目
8. 到 `生成配置` 页面试生成一次，确认输出路径和内容正确

### 3.2 批量修改已有模板

建议明确告诉 AI：

- 修改范围：例如 `templates/elasticsearch/` 下所有模板
- 旧写法和新写法：例如把 `{{ .Instance.NodeAlias }}` 改成 `{{ .Instance.Node.Hostname }}`
- 返回所有受影响文件的完整草稿，不要只返回解释

## 4. 变量与上下文约束

运行时模板上下文来自 `config.Context`，对应实现见 [generator/template.go](../generator/template.go) 和 [config/types.go](../config/types.go)。

### 4.1 可动态扩展的变量

#### `.Global.<key>`

来源：`config.yaml > global`

特点：

- 可以新增或删除 key
- 新增后需要同步到配置文件
- 适合放全局复用变量

常见示例：

- `{{ .Global.user }}`
- `{{ .Global.group }}`
- `{{ .Global.install_base_dir }}`
- `{{ .Global.data_base_dir }}`
- `{{ .Global.java_home }}`
- `{{ .Global.software_dir }}`

#### `.Instance.Vars.<key>`

来源：`config.yaml > serverConfig.<service>.vars`

特点：

- 可以按服务新增或删除 key
- 新增后需要同步到 `serverConfig.<service>.vars`
- 适合放服务自身参数

常见示例：

- `{{ .Instance.Vars.version }}`
- `{{ .Instance.Vars.install_subdir }}`
- `{{ .Instance.Vars.data_subdir }}`
- `{{ .Instance.Vars.http_port }}`
- `{{ .Instance.Vars.cluster_name }}`

### 4.2 固定变量

下面这些字段由代码结构固定，不允许自定义字段名。

允许使用：

```gotemplate
{{ .Instance.ServiceName }}
{{ .Instance.NodeName }}
{{ .Instance.Node.IP }}
{{ .Instance.Node.Hostname }}
{{ .Instance.Node.NodeName }}
{{ .Instance.AutoID }}
```

禁止使用：

```gotemplate
{{ .Instance.NodeAlias }}
{{ .Instance.Node.HostName }}
{{ .Instance.Hostname }}
{{ .Node.Hostname }}
```

### 4.3 全量上下文对象

模板内还可以使用：

- `.Global`
- `.Nodes`
- `.Instance`
- `.AllInstances`

其中 `.AllInstances` 常用于生成集群连接串、主从节点列表、服务发现配置。

## 5. 全局服务函数说明与示例

这些函数由后端模板函数映射提供，实际实现以 [generator/template.go](../generator/template.go) 为准。

### 5.1 `serviceNodes`

用途：

- 获取某个服务的全部实例
- 返回值类型是 `[]config.ServiceInstance`
- 常用于 `range` 遍历生成集群配置

示例：

```gotemplate
{{ range serviceNodes "zookeeper" }}
server.{{ .AutoID }}={{ .Node.IP }}:2888:3888
{{ end }}
```

适用场景：

- ZooKeeper `server.N`
- Hadoop HA 节点列表
- Kafka broker 列表的复杂拼装

### 5.2 `serviceIPs`

用途：

- 获取某个服务全部实例的 IP 列表
- 返回值类型是 `[]string`

示例：

```gotemplate
{{ serviceIPs "zookeeper" }}
{{ join "," (serviceIPs "zookeeper") }}
```

典型输出：

```text
192.168.10.11,192.168.10.12,192.168.10.13
```

适用场景：

- 白名单
- 集群成员 IP 列表
- 防火墙或 hosts 汇总配置

### 5.3 `serviceHostnames`

用途：

- 获取某个服务全部实例的主机名列表
- 返回值类型是 `[]string`

示例：

```gotemplate
{{ join "," (serviceHostnames "zookeeper") }}
```

适用场景：

- 生成 hostname 列表
- 生成 quorum / seed hosts / peers 配置

### 5.4 `serviceEndpoints`

用途：

- 根据服务名和端口字段，生成 `ip:port` 列表
- 返回值类型是 `[]string`
- 端口字段从目标服务的 `vars` 中读取

示例：

```gotemplate
{{ serviceEndpoints "zookeeper" "client_port" }}
{{ join "," (serviceEndpoints "zookeeper" "client_port") }}
```

典型输出：

```text
192.168.10.11:2181,192.168.10.12:2181,192.168.10.13:2181
```

适用场景：

- ZooKeeper 连接串
- Kafka bootstrap servers
- Redis Sentinel / ES seed nodes 等端点聚合

### 5.5 `serviceVar`

用途：

- 读取某个服务在 `serverConfig.<service>.vars` 下的单个变量
- 当前代码签名是 `serviceVar(serviceName, varName)`
- 找不到时返回 `nil`

示例：

```gotemplate
{{ serviceVar "zookeeper" "client_port" }}
{{ default "2181" (serviceVar "zookeeper" "client_port") }}
```

注意：

- 当前实现不支持三参数写法
- 下面这种写法是错误示例：

```gotemplate
{{ serviceVar "zookeeper" "client_port" "2181" }}
```

如果需要默认值，请组合 `default`：

```gotemplate
{{ default "2181" (serviceVar "zookeeper" "client_port") }}
```

适用场景：

- 读取其他服务共享端口
- 读取集群公共变量
- 跨服务引用某个固定参数

### 5.6 `join`

用途：

- 把数组拼成字符串
- 支持 `join "," list` 和 `join list ","` 两种写法

示例：

```gotemplate
{{ join "," (serviceHostnames "zookeeper") }}
{{ join ":" (serviceIPs "elasticsearch") }}
```

### 5.7 `default`

用途：

- 当值为空或 `nil` 时，使用默认值

示例：

```gotemplate
{{ default "value" .Instance.Vars.some_key }}
{{ default "/data/logs" .Instance.Vars.log_dir }}
{{ default "2181" (serviceVar "zookeeper" "client_port") }}
```

适用场景：

- 某些服务参数允许不显式配置
- 模板需要兜底值
- 兼容历史配置缺项

## 6. 配置结构说明

建议结合 [README.md](../README.md) 一起看。

### 6.1 `global`

用于放全局复用变量。

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

### 6.2 `nodes`

定义节点池，键名是节点别名。

示例：

```yaml
nodes:
  master1:
    ip: 192.168.10.11
    hostname: master1.hadoop.local
  worker1:
    ip: 192.168.10.21
    hostname: worker1.hadoop.local
```

注意：

- 一个 IP 可以被多个节点别名复用
- 模板中应优先使用 `.Instance.Node.IP` 和 `.Instance.Node.Hostname`

### 6.3 `serviceTop`

定义服务部署在哪些节点上。

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

- `nodes: ["*"]` 或 `["all"]` 表示所有节点执行
- `id_auto_derive: true` 时，生成实例会自动计算 `.Instance.AutoID`

### 6.4 `serverConfig`

定义服务元数据和 `vars`。

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
      client_port: 2181
```

说明：

- `type: "global"` 常用于全局服务
- `id_field` / `id_format` 控制自动 ID 推导
- `vars` 是模板中 `.Instance.Vars` 的主要来源

### 6.5 `nodeOverrides`

定义节点级覆盖变量。

示例：

```yaml
nodeOverrides:
  worker1:
    elasticsearch:
      heap_size: 8g
      data_subdir: elasticsearch/data01
```

说明：

- 会覆盖对应节点上该服务的默认 `vars`
- 适用于同一服务在不同节点需要局部差异时使用

## 7. 模板编写规范

### 7.1 模板目录规范

正式模板路径必须在：

```text
templates/<service>/*.tmpl
```

例如：

```text
templates/elasticsearch/install.sh.tmpl
templates/elasticsearch/elasticsearch.yml.tmpl
templates/elasticsearch/check.sh.tmpl
```

### 7.2 输出目录规范

当前生成逻辑见 [generator/output.go](../generator/output.go)。

默认输出目录结构：

```text
output/users/<username>/<nodeIP>/<service>/
```

例如：

```text
output/users/admin/192.168.10.11/elasticsearch/install.sh
output/users/admin/192.168.10.11/elasticsearch/elasticsearch.yml
```

除非明确要求，否则不要让 AI 假设还会额外生成：

```text
output/<ip>/<service>/config/xxx.conf
```

### 7.3 变量使用规范

建议优先：

```gotemplate
{{ .Global.user }}
{{ .Global.group }}
{{ .Global.install_base_dir }}
{{ .Global.data_base_dir }}
{{ .Instance.Node.IP }}
{{ .Instance.Node.Hostname }}
{{ .Instance.Vars.install_subdir }}
{{ .Instance.Vars.data_subdir }}
```

避免：

```gotemplate
{{ .Instance.NodeAlias }}
{{ .Instance.Hostname }}
{{ .Node.IP }}
```

### 7.4 脚本规范

安装脚本建议满足：

- 幂等
- 路径可配置
- 用户和用户组取自全局变量
- 配置文件生成后可被安装脚本复制到实际目录
- 尽量避免写死绝对目录和端口

推荐模式：

```bash
run_as_root mkdir -p "{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}"
run_as_root mkdir -p "{{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}"
```

### 7.5 集群配置规范

需要拼集群成员时，优先用全局服务函数，而不是写死节点：

```gotemplate
cluster.initial_master_nodes={{ join "," (serviceHostnames "elasticsearch") }}
discovery.seed_hosts={{ join "," (serviceEndpoints "elasticsearch" "transport_port") }}
zookeeper.connect={{ join "," (serviceEndpoints "zookeeper" "client_port") }}
```

## 8. 推荐提示词模板

```text
#skill:core/template_contract
#skill:core/variable_contract
#skill:task/install_script
#skill:task/config_file

请为 <服务名> 生成一套可审核、可保存、可渲染的 Go Template 模板草稿。

服务信息：
- 服务名：<service>
- 版本：<version>
- 安装包路径：<安装包路径>
- 安装目录：使用 {{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}
- 数据目录：使用 {{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}
- 日志目录：使用 {{ .Global.log_base_dir }}/{{ .Instance.Vars.log_subdir }}
- 节点主机名必须使用 {{ .Instance.Node.Hostname }}
- 节点 IP 必须使用 {{ .Instance.Node.IP }}

部署要求：
- install.sh.tmpl 需要具备幂等性
- 如果服务有配置文件，安装脚本需要把生成后的配置文件复制或覆盖到服务真实配置目录
- 输出目录按 output/<ip>/<service>/ 理解，不要额外生成 config 子目录，除非我明确要求
- 请同时给出 serviceTop 和 serverConfig.vars 的补丁建议

请输出：
- 每个模板文件的 path、content、reason
- serviceTop 建议
- serverConfig.vars 建议
- 不确定项写入 warnings，不要自行杜撰
```

## 9. 人工审核清单

保存前至少检查：

- 模板路径是否都在 `templates/<service>/` 下
- 文件名是否以 `.tmpl` 结尾
- 是否出现禁用变量，如 `.Instance.NodeAlias`
- 是否错误创建了多余目录，例如 `templates/<service>/config/...`
- shell 脚本是否具备幂等性
- `.Instance.Vars.<key>` 是否都能在 `serverConfig.<service>.vars` 找到
- `serviceTop` 是否包含正确节点
- 安装脚本是否会把配置文件放到服务真实目录
- 输出目录理解是否符合 `output/<ip>/<service>/`

## 10. 常见问题

### AI 使用了错误节点变量

错误示例：

```gotemplate
{{ .Instance.NodeAlias }}
```

正确写法：

```gotemplate
{{ .Instance.Node.Hostname }}
```

### 保存并同步配置提示缺少 `serviceTop`

原因通常是：

- AI 只生成了模板，没有给出 `serviceTop` 补丁
- 当前会话中的配置草案还没补齐拓扑

处理方式：

- 到 `配置管理 > 服务拓扑` 手工补齐
- 或继续让 AI 生成 `serviceTop` 建议

### AI 返回纯文本，不能保存草稿

原因通常是：

- 当前模型未按系统 JSON 协议返回
- prompt 没有带 `core/output_json`

### 页面显示已修改但文件未更新

排查顺序：

1. 确认当前登录用户是否为管理员
2. 确认点击的是 `保存当前` 或 `保存全部`
3. 到模板编辑器或磁盘下直接检查 `templates/<service>/`
4. 查看后端日志中的 `[AI] save write path=...`
