# 大数据平台离线配置生成器 v2.0

## 一、项目简介

本工具是一个**完全动态、零硬编码**的大数据平台配置生成器，专为无SSH环境的离线交付场景设计。

### 核心特性

- ✅ **Ansible Inventory风格配置** - 简洁、直观、易维护
- ✅ **完全动态解析** - 新增服务无需修改代码
- ✅ **自动推导变量** - 自动计算ZK连接串、Kafka Brokers等跨节点配置
- ✅ **分层配置** - 全局变量 → 服务变量 → 主机变量
- ✅ **模板化输出** - 基于Go Template，灵活可扩展

## 二、配置文件结构

### 2.1 整体结构

```yaml
# 全局变量（所有服务共享）
all:
  vars:
    jdk_version: "1.8.0_65"
    install_base_dir: "/data/localization"
    data_base_dir: "/data/bigdata"
    # ... 其他全局变量

# 服务定义（服务名可任意，不硬编码）
zookeeper:
  vars:       # 服务级变量
    client_port: 2181
    server_port: 2888
  hosts:      # 该服务部署的主机
    192.168.10.10:
      hostname: node1
      myid: 1
    # ...

kafka:
  vars:
    broker_port: 9092
  hosts:
    192.168.10.13:
      hostname: node4
      broker_id: 1
```

### 2.2 三层变量优先级

1. **all.vars** - 全局变量（最低优先级）
2. **<service>.vars** - 服务级变量（中等优先级）
3. **<service>.hosts.<ip>** - 主机级变量（最高优先级）

### 2.3 如何添加新服务

只需在`config.yaml`中添加服务定义，无需修改代码：

```yaml
# 例如：添加Doris FE
doris_fe:
  vars:
    version: "2.0.3"
    http_port: 8030
    query_port: 9030
  hosts:
    192.168.10.10:
      hostname: node1
      fe_id: 1

# 然后在templates/目录下创建模板
# templates/doris_fe/fe.conf.tmpl
# templates/doris_fe/install.sh.tmpl
```

## 三、使用方法

### 3.1 准备工作

1. 编辑`config.yaml`，定义集群拓扑
2. 在`templates/<service>/`目录下创建模板文件（.tmpl后缀）

### 3.2 运行生成器

```bash
# 编译
go build -o generator main.go

# 运行
./generator

# 或者直接运行
go run main.go
```

### 3.3 输出目录结构

```
output/
├── README.md                    # 部署总览
├── 192.168.10.10/              # 每台主机一个目录
│   ├── zookeeper/
│   │   ├── install.sh
│   │   ├── start.sh
│   │   └── zoo.cfg
│   └── kafka/
│       └── server.properties
├── 192.168.10.11/
│   └── ...
└── ...
```

### 3.4 交付步骤

1. 将`output/<ip>`目录上传到对应主机
2. 在目标主机执行：`bash <service>/install.sh`
3. 安装完成后启动服务

## 四、模板编写指南

### 4.1 可用变量

模板中可访问以下变量：

```go
.Vars          // 全局变量 (map)
.ServiceVars   // 当前服务的变量 (map)
.HostVars      // 当前主机的变量 (map)
.IP            // 当前主机IP (string)
.Hostname      // 当前主机名 (string)
.Derived       // 衍生变量 (map)
.AllServices   // 所有服务配置 (map)
```

### 4.2 自定义函数

模板中可使用以下函数：

```go
// 字符串操作
{{ .Vars.some_var | upper }}          // 转大写
{{ .Vars.some_var | lower }}          // 转小写
{{ .Vars.some_var | replace "old" "new" }}  // 替换
{{ join .List "," }}                  // 列表连接

// 数学运算
{{ add 1 2 }}                         // 加法
{{ sub 5 3 }}                         // 减法

// 变量访问
{{ globalVar "jdk_version" }}         // 获取全局变量
{{ serviceNodes "zookeeper" }}        // 获取服务节点列表
{{ serviceVar "kafka" "broker_port" }} // 获取服务变量

// 默认值
{{ .Vars.port | default 8080 }}       // 默认值
```

### 4.3 模板示例

**Zoo.cfg示例**：

```properties
# 数据目录
dataDir={{ globalVar "data_base_dir" }}/zookeeper/data

# 客户端端口
clientPort={{ .ServiceVars.client_port }}

# 服务器列表（自动生成）
{{- $servers := serviceNodes "zookeeper" }}
{{- range $idx, $ip := $servers }}
server.{{ add $idx 1 }}={{ $ip }}:{{ $.ServiceVars.server_port }}:{{ $.ServiceVars.election_port }}
{{- end }}
```

**Kafka server.properties示例**：

```properties
# Broker ID
broker.id={{ .HostVars.broker_id }}

# 监听地址
listeners=PLAINTEXT://{{ .IP }}:{{ .ServiceVars.broker_port }}

# ZK连接串（自动生成）
zookeeper.connect={{ .Derived.zk_connect_string }}
```

## 五、衍生变量说明

生成器会自动计算以下衍生变量：

| 变量名 | 说明 | 示例值 |
|--------|------|--------|
| `zk_connect_string` | ZK连接串 | `192.168.10.10:2181,192.168.10.11:2181` |
| `kafka_brokers` | Kafka Broker列表 | `192.168.10.13:9092,192.168.10.14:9092` |
| `spark_master_url` | Spark Master URL | `spark://192.168.10.10:7077` |
| `namenode_rpc_addresses` | HDFS NameNode RPC地址 | `192.168.10.10:9820,192.168.10.11:9820` |
| `es_seed_hosts` | ES种子节点 | `192.168.10.16:9300,192.168.10.17:9300` |

## 六、与v1版本对比

### 旧版配置（v1）

```yaml
# 需要在Global.Services中定义节点列表
Global:
  Services:
    Zookeeper:
      Nodes: ["192.168.10.10", "192.168.10.11", "192.168.10.12"]
      
# 需要在Groups中定义组
Groups:
  zookeeper_group:
    Services: [Zookeeper]
    
# 需要在Hosts中引用组
Hosts:
  "192.168.10.10":
    Groups: ["zookeeper_group"]
```

### 新版配置（v2）

```yaml
# 节点列表直接在hosts中定义，无需重复
zookeeper:
  vars:
    client_port: 2181
  hosts:
    192.168.10.10:
      hostname: node1
      myid: 1
    192.168.10.11:
      hostname: node2
      myid: 2
```

**优势**：
- 减少70%的配置代码
- 节点信息只定义一次
- 无需维护Groups映射关系

## 七、最佳实践

### 7.1 配置文件组织

```yaml
# 1. 按服务类型分组
all:
  vars: ...

# 基础服务
zookeeper: ...
kafka: ...

# Hadoop生态
hadoop_namenode: ...
hadoop_datanode: ...
spark_master: ...

# 数据存储
elasticsearch: ...
doris_fe: ...

# 调度系统
dolphinscheduler_master: ...
```

### 7.2 变量命名规范

```yaml
# 全局变量：使用完整路径名
install_base_dir: "/data/localization"
data_base_dir: "/data/bigdata"

# 服务变量：使用短名
client_port: 2181
server_port: 2888

# 主机变量：使用特定标识
myid: 1
broker_id: 1
namenode_id: nn1
```

### 7.3 模板文件命名

```
templates/
├── zookeeper/
│   ├── zoo.cfg.tmpl          # 配置文件
│   ├── myid.tmpl             # ID文件
│   ├── install.sh.tmpl       # 安装脚本
│   └── start.sh.tmpl         # 启动脚本
├── kafka/
│   ├── server.properties.tmpl
│   ├── install.sh.tmpl
│   └── start.sh.tmpl
```

## 八、扩展性

### 8.1 添加新的衍生变量

在`main.go`的`computeDerivedVars`函数中添加：

```go
// 例如：添加HBase Master地址
if hbaseNodes, ok := derived["hbase_master_nodes"].([]string); ok {
    var hbaseMasters []string
    for _, ip := range hbaseNodes {
        hbaseMasters = append(hbaseMasters, fmt.Sprintf("%s:16000", ip))
    }
    derived["hbase_master_addresses"] = strings.Join(hbaseMasters, ",")
}
```

### 8.2 添加新的模板函数

在`renderTemplate`函数的`funcMap`中添加：

```go
"trim": strings.TrimSpace,
"split": strings.Split,
// ...
```

## 九、注意事项

1. **服务名必须与模板目录名一致**
   - 服务：`zookeeper` → 模板目录：`templates/zookeeper/`

2. **模板文件必须以.tmpl结尾**
   - `zoo.cfg.tmpl` → 输出为 `zoo.cfg`

3. **YAML缩进必须使用空格**
   - 使用2个空格，不要使用Tab

4. **变量引用使用globalVar函数**
   - ✅ `{{ globalVar "data_base_dir" }}`
   - ❌ `{{ .Vars.data_base_dir }}` (可能无法正确解析)

## 十、常见问题

**Q: 如何验证配置文件语法？**
```bash
go run main.go
cat run.log  # 查看日志
```

**Q: 如何查看生成的配置？**
```bash
cat output/192.168.10.10/zookeeper/zoo.cfg
```

**Q: 如何添加新的服务？**
1. 在config.yaml中添加服务定义
2. 创建templates/<service>/目录
3. 添加模板文件

无需修改代码！

## 十一、项目结构

```
bigdata-deploy-generator/
├── main.go              # 主程序（完全动态）
├── config.yaml          # 配置文件
├── templates/           # 模板库
│   ├── zookeeper/
│   ├── kafka/
│   └── ...
├── output/              # 输出目录
│   ├── README.md
│   ├── 192.168.10.10/
│   └── ...
├── go.mod
├── go.sum
└── README.md            # 本文档
```

## 十二、联系与支持

如有问题或建议，请提交Issue或PR。

---

**版本**: v2.0  
**更新日期**: 2025-01-XX  
**作者**: BigData Platform Team
