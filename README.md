# 大数据平台离线配置生成器 v5.1 - 通用版（零硬编码）

## 一、核心特性

### 🎯 完全零硬编码

**代码中不包含任何服务名、端口或连接串的硬编码**，所有配置都通过模板函数动态获取：

| 对比项 | 传统方式 | 本工具 |
|--------|----------|--------|
| 服务名 | 代码中硬编码 | 配置文件定义 |
| 端口号 | 代码中硬编码 | 配置文件定义 |
| 连接串 | 代码中拼接 | 模板函数动态生成 |
| ID推导 | 代码中写死规则 | 配置格式化模板 |

### 🆕 新增特性

| 特性 | 说明 |
|------|------|
| **全局服务** | 支持 `nodes: ["*"]` 表示所有节点执行，适用于服务器初始化、JDK安装等 |
| **节点复用** | 一个IP可对应多个节点别名，适用于混合部署场景 |
| **ID格式配置化** | 通过 `id_format` 配置ID格式，如 `"index+1"` 或 `"nn{index+1}"` |
| **服务类型标识** | 全局服务通过 `type: "global"` 标识，无需ID推导 |

### 配置结构

```yaml
config.yaml
├── global:        全局参数 - JDK、目录、用户等
├── nodes:         节点池 - IP/hostname映射（支持复用）
├── serviceTop:    服务拓扑 - 部署位置（支持 "*" 全局服务）
└── serverConfig:  服务配置 - 参数、ID字段、ID格式
```

---

## 二、配置文件详解

### 2.1 global - 全局配置

```yaml
global:
  # 基础目录
  install_base_dir: "/data/localization"
  data_base_dir: "/data/bigdata"
  
  # JDK配置
  jdk_version: "1.8.0_65"
  java_home: "/data/jdk/jdk1.8.0_65"
  
  # 系统用户
  user: "bigdata"
  timezone: "Asia/Shanghai"
```

### 2.2 nodes - 节点池（支持节点复用）

```yaml
nodes:
  # Master节点
  dw-master1:
    ip: "192.168.10.10"
    hostname: "dw-master1"
  
  # Worker节点
  dw-worker1:
    ip: "192.168.10.13"
    hostname: "dw-worker1"
  
  # ⭐ 节点复用示例：以下两个节点别名指向同一个IP
  # realtime-kafka1 和 dw-worker1 共用一台物理机
  realtime-kafka1:
    ip: "192.168.10.13"  # 与dw-worker1共用IP
    hostname: "realtime-kafka1"
```

**节点复用说明**：
- 一个IP可以有多个节点别名
- 适用于混合部署（如Kafka和Hadoop DataNode共用机器）
- 每个别名可以有独立的配置

### 2.3 serviceTop - 服务拓扑（支持全局服务）

```yaml
serviceTop:
  # ⭐ 全局服务：使用 "*" 表示所有节点
  server_init:
    nodes: ["*"]  # 所有节点执行服务器初始化
  
  install_jdk:
    nodes: ["*"]  # 所有节点安装JDK
  
  # 中间件服务
  zookeeper:
    nodes: [dw-master1, dw-master2, dw-master3]
    id_auto_derive: true  # 自动推导myid
  
  kafka:
    nodes: [realtime-kafka1, realtime-kafka2, realtime-kafka3]
    id_auto_derive: true  # 自动推导broker_id
```

**全局服务说明**：
- `nodes: ["*"]` 或 `nodes: ["all"]` 表示所有节点
- 适用于服务器初始化、安装JDK、安装Docker等全局操作
- 全局服务不需要ID推导

### 2.4 serverConfig - 服务配置（ID格式配置化）

```yaml
serverConfig:
  # 全局服务
  server_init:
    type: "global"  # 标识为全局服务
    description: "服务器初始化（所有节点）"
  
  install_jdk:
    type: "global"
    description: "安装JDK（所有节点）"
  
  # 中间件服务
  zookeeper:
    id_field: "myid"        # ID字段名
    id_format: "index+1"    # 格式：1, 2, 3...
    vars:
      version: "3.8.3"
      client_port: 2181
  
  kafka:
    id_field: "broker_id"
    id_format: "index+1"    # 格式：1, 2, 3...
    vars:
      version: "3.5.1"
      broker_port: 9092
  
  hadoop_namenode:
    id_field: "namenode_id"
    id_format: "nn{index+1}"  # ⭐ 格式化：nn1, nn2...
    vars:
      version: "3.3.6"
      namenode_port: 9820
```

**ID格式说明**：
- `index+1`：索引+1，如 1, 2, 3
- `index`：原始索引，如 0, 1, 2
- `nn{index+1}`：格式化模板，如 nn1, nn2
- `broker-{index+1}`：带前缀，如 broker-1, broker-2

---

## 三、常见场景示例

### 3.1 添加新服务（无需修改代码）

**步骤1：添加服务拓扑**

```yaml
serviceTop:
  doris_fe:
    nodes: [dw-master1, dw-master2]
    id_auto_derive: true
```

**步骤2：添加服务配置**

```yaml
serverConfig:
  doris_fe:
    id_field: "fe_id"
    id_format: "index+1"
    vars:
      version: "2.0.3"
      http_port: 8030
```

**步骤3：创建模板文件**

```
templates/doris_fe/fe.conf.tmpl
templates/doris_fe/install.sh.tmpl
```

**模板示例**：

```gotemplate
# templates/doris_fe/fe.conf.tmpl
# Doris FE 配置文件
# 节点: {{.Instance.NodeName}}
# IP: {{.Instance.Node.IP}}
# fe_id: {{.Instance.AutoID}}

meta_dir = {{.Global.data_base_dir}}/doris/fe/doris-meta
http_port = {{.Instance.Vars.http_port}}
```

### 3.2 节点复用配置

**场景**：Kafka和Hadoop DataNode共用3台机器

```yaml
nodes:
  # Worker节点（Hadoop）
  dw-worker1:
    ip: "192.168.10.13"
  dw-worker2:
    ip: "192.168.10.14"
  dw-worker3:
    ip: "192.168.10.15"
  
  # Kafka节点（复用Worker机器）
  realtime-kafka1:
    ip: "192.168.10.13"  # 与dw-worker1共用
  realtime-kafka2:
    ip: "192.168.10.14"  # 与dw-worker2共用
  realtime-kafka3:
    ip: "192.168.10.15"  # 与dw-worker3共用

serviceTop:
  hadoop_datanode:
    nodes: [dw-worker1, dw-worker2, dw-worker3]
  
  kafka:
    nodes: [realtime-kafka1, realtime-kafka2, realtime-kafka3]
    id_auto_derive: true
```

**生成结果**：
- `dw-worker1` 和 `realtime-kafka1` 都会生成配置文件
- 它们使用不同的节点别名，但共用同一IP
- Kafka的broker.id会自动推导为1, 2, 3

### 3.3 交付流程配置

**标准交付流程**：服务器初始化 → 安装全局工具 → 中间件 → 应用

```yaml
serviceTop:
  # 第一步：服务器初始化（所有节点）
  server_init:
    nodes: ["*"]
  
  # 第二步：安装全局工具（所有节点）
  install_jdk:
    nodes: ["*"]
  
  # install_docker:
  #   nodes: ["*"]
  
  # 第三步：基础服务
  zookeeper:
    nodes: [dw-master1, dw-master2, dw-master3]
    id_auto_derive: true
  
  kafka:
    nodes: [realtime-kafka1, realtime-kafka2, realtime-kafka3]
    id_auto_derive: true
  
  # 第四步：计算引擎
  hadoop_namenode:
    nodes: [dw-master1, dw-master2]
    id_auto_derive: true
  
  spark_master:
    nodes: [dw-master1]
  
  # 第五步：业务应用
  myapp:
    nodes: [dw-master2, dw-worker1]
```

---

## 四、模板开发指南

### 4.1 模板变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `.Global` | 全局配置 | `.Global.data_base_dir` |
| `.Instance.NodeName` | 节点别名 | `dw-master1` |
| `.Instance.Node.IP` | 节点IP | `192.168.10.10` |
| `.Instance.Node.Hostname` | 主机名 | `dw-master1` |
| `.Instance.Vars` | 服务参数 | `.Instance.Vars.client_port` |
| `.Instance.AutoID` | 自动推导的ID | `1` 或 `nn1` |
| `.AllInstances` | 所有服务实例 | 用于模板函数 |

### 4.2 通用模板函数（零硬编码）

**核心设计**：通过模板函数动态获取任意服务的端点，而非硬编码在代码中。

#### 端点生成函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `serviceEndpoints "服务名" "端口字段"` | 获取服务的端点列表 `[]string` | `["192.168.1.10:2181", "192.168.1.11:2181"]` |
| `serviceEndpointsJoin "服务名" "端口字段"` | 获取端点连接串（逗号分隔） | `"192.168.1.10:2181,192.168.1.11:2181"` |
| `serviceNodes "服务名"` | 获取服务的所有实例 | 用于遍历生成配置 |

#### 使用示例

**获取ZooKeeper连接串**：
```gotemplate
{{/* 不需要硬编码zookeeper，通过函数动态获取 */}}
zookeeper.connect={{serviceEndpointsJoin "zookeeper" "client_port"}}
```

**获取Kafka Brokers**：
```gotemplate
{{/* 不需要硬编码kafka，通过函数动态获取 */}}
bootstrap.servers={{serviceEndpointsJoin "kafka" "broker_port"}}
```

**遍历服务节点生成配置**：
```gotemplate
{{range $idx, $node := serviceNodes "zookeeper"}}
server.{{$node.AutoID}}={{$node.Node.IP}}:2888:3888
{{end}}
```

### 4.3 模板示例

**Kafka配置模板（使用模板函数获取ZK连接）**：

```gotemplate
# templates/kafka/server.properties.tmpl
broker.id={{.Instance.AutoID}}
listeners=PLAINTEXT://{{.Instance.Node.IP}}:{{.Instance.Vars.broker_port}}

# ⭐ ZK连接串通过模板函数动态获取，无需硬编码
zookeeper.connect={{serviceEndpointsJoin "zookeeper" "client_port"}}

log.dirs={{.Global.data_base_dir}}/kafka/logs
num.partitions={{.Instance.Vars.num_partitions}}
```

**全局服务模板**：

```gotemplate
# templates/server_init/install.sh.tmpl
#!/bin/bash
# 服务器初始化: {{.Instance.NodeName}}
# IP: {{.Instance.Node.IP}}

# 设置时区
ln -sf /usr/share/zoneinfo/{{.Global.timezone}} /etc/localtime

# 创建用户
groupadd -f {{.Global.group}}
id -u {{.Global.user}} &>/dev/null || useradd -g {{.Global.group}} {{.Global.user}}

# 创建目录
mkdir -p {{.Global.install_base_dir}}
mkdir -p {{.Global.data_base_dir}}
```

---

## 五、运行与输出

### 5.1 运行生成器

```bash
go run main.go
```

### 5.2 输出示例

```
============================================
配置加载成功
============================================
节点数: 12
服务数: 14

============================================
服务实例列表
============================================

[server_init] - 服务器初始化（所有节点）
  类型: 全局服务（所有节点）
  节点:
    - dw-master1 (192.168.10.10)
    - dw-master2 (192.168.10.11)
    ...

[zookeeper]
  ID字段: myid (格式: index+1)
  节点:
    - dw-master1 (192.168.10.10) [myid=1]
    - dw-master2 (192.168.10.11) [myid=2]
    - dw-master3 (192.168.10.12) [myid=3]

============================================
节点复用统计
============================================
192.168.10.13: [dw-worker1 realtime-kafka1] (复用)
192.168.10.14: [dw-worker2 realtime-kafka2] (复用)
192.168.10.15: [dw-worker3 realtime-kafka3] (复用)

============================================
生成完成！
============================================
```

### 5.3 输出目录结构

```
output/
├── server_init/
│   └── install.sh  # 每个节点一个
├── install_jdk/
│   └── install.sh
├── zookeeper/
│   ├── dw-master1_myid         # 内容: 1
│   ├── dw-master1_zoo.cfg
│   ├── dw-master1_install.sh
│   └── ...
└── kafka/
    ├── realtime-kafka1_server.properties  # broker.id=1
    ├── realtime-kafka1_install.sh
    └── ...
```

---

## 六、版本对比

| 特性 | v4版本 | v5.1版本 |
|------|--------|----------|
| 服务扩展 | 需修改代码 | **无需修改代码** |
| 全局服务 | 不支持 | **支持 `nodes: ["*"]`** |
| 节点复用 | 支持 | **支持（完整示例）** |
| ID格式 | 固定规则 | **配置化 `id_format`** |
| 衍生变量 | 代码硬编码 | **模板函数动态生成** |
| 代码复杂度 | 中（有硬编码） | **低（零硬编码）** |

### 关键改进（v5.1）

**移除硬编码的衍生变量计算**：
- ❌ 旧版：代码中硬编码 `zk_connect`、`kafka_brokers`
- ✅ 新版：通过 `serviceEndpointsJoin` 模板函数动态获取任意服务的端点

---

## 七、最佳实践

### 7.1 节点命名规范

```yaml
# 按业务模块+序号命名
nodes:
  dw-master1:      # 数仓Master节点
  dw-worker1:      # 数仓Worker节点
  realtime-kafka1: # 实时Kafka节点
  realtime-es1:    # 实时ES节点
```

### 7.2 服务拓扑组织

```yaml
serviceTop:
  # 按交付顺序组织
  # 1. 全局服务
  server_init:
    nodes: ["*"]
  install_jdk:
    nodes: ["*"]
  
  # 2. 基础服务
  zookeeper: ...
  kafka: ...
  
  # 3. 计算引擎
  hadoop_namenode: ...
  spark_master: ...
  
  # 4. 数据存储
  elasticsearch: ...
  
  # 5. 业务应用
  myapp: ...
```

---

## 八、注意事项

1. **模板目录名**：必须与服务名一致（如 `zookeeper` → `templates/zookeeper/`）
2. **模板文件后缀**：必须以 `.tmpl` 结尾
3. **全局服务标识**：需要在 `serverConfig` 中配置 `type: "global"`

---

**版本**: v5.1 通用版  
**核心特性**: 零硬编码 + 模板函数动态生成端点 + 全局服务 + 节点复用  
**维护效率**: 添加新服务无需修改代码，只需配置+模板
