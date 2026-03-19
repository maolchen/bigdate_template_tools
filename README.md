# 大数据平台离线配置生成器 v4.0 - 拓扑配置分离版

## 一、v4核心改进

### 配置结构分离

将配置分为4个独立部分，**各司其职，互不干扰**：

```
config.yaml (217行)
├── global:        全局参数 (25行) - 基础目录、JDK、用户等
├── nodes:         节点池 (46行) - IP/hostname映射
├── serviceTop:    服务拓扑 (52行) - 服务部署位置 ← 修改部署只需看这里
└── serverConfig:  服务配置 (79行) - 服务参数 ← 很少需要修改
```

### 修改操作效率

| 操作场景 | 需要阅读的行数 | 说明 |
|---------|--------------|------|
| 修改部署位置 | **52行** (serviceTop) | 只需看服务拓扑部分 |
| 修改IP地址 | **46行** (nodes) | 只需看节点池部分 |
| 修改服务参数 | **79行** (serverConfig) | 很少需要修改 |

---

## 二、配置文件结构详解

### 2.1 global - 全局配置

```yaml
global:
  # 基础目录
  install_base_dir: "/data/localization"
  data_base_dir: "/data/bigdata"
  
  # JDK配置
  jdk_version: "1.8.0_65"
  jdk_path: "/data/jdk"
  
  # 系统用户
  user: "bigdata"
  timezone: "Asia/Shanghai"
```

**说明**：全局参数几乎不需要修改。

### 2.2 nodes - 节点池

```yaml
nodes:
  realtime-dw1:
    ip: "192.168.10.10"
    hostname: "realtime-dw1"
  
  realtime-dw2:
    ip: "192.168.10.11"
    hostname: "realtime-dw2"
  
  # 允许节点复用：同一IP可以有多个节点别名
  # realtime-kafka1:
  #   ip: "192.168.10.13"  # 与其他节点共用物理机
```

**说明**：
- 修改IP只需改这里
- 支持节点复用（一个IP对应多个节点别名）
- 节点别名可以根据业务命名（如realtime-dw1、realtime-kafka1）

### 2.3 serviceTop - 服务拓扑

```yaml
serviceTop:
  # 基础服务
  zookeeper:
    nodes: [realtime-dw1, realtime-dw2, realtime-dw3]
    id_auto_derive: true  # 自动推导myid
  
  kafka:
    nodes: [realtime-kafka1, realtime-kafka2, realtime-kafka3]
    id_auto_derive: true  # 自动推导broker_id
  
  # Hadoop生态
  hadoop_namenode:
    nodes: [realtime-dw1, realtime-dw2]
    id_auto_derive: true  # 自动推导namenode_id
```

**说明**：
- **修改部署位置只需看这里**
- 使用节点别名列表指定部署位置
- `id_auto_derive: true` 自动推导ID

### 2.4 serverConfig - 服务配置

```yaml
serverConfig:
  zookeeper:
    version: "3.8.3"
    client_port: 2181
    server_port: 2888
    election_port: 3888
  
  kafka:
    version: "3.5.1"
    broker_port: 9092
    num_partitions: 3
```

**说明**：
- 服务参数配置
- 一般不需要修改

---

## 三、常见操作示例

### 3.1 修改部署位置

**场景**：将Kafka从realtime-kafka节点迁移到realtime-dw节点

```yaml
# 只需修改serviceTop部分
serviceTop:
  kafka:
    nodes: [realtime-dw1, realtime-dw2, realtime-dw3]  # 修改这行即可
    id_auto_derive: true
```

### 3.2 修改IP地址

**场景**：将realtime-dw1的IP从192.168.10.10改为10.20.30.40

```yaml
# 只需修改nodes部分
nodes:
  realtime-dw1:
    ip: "10.20.30.40"  # 只改这一处，所有服务自动生效
    hostname: "realtime-dw1"
```

### 3.3 添加新服务

**步骤1：在serviceTop添加拓扑**
```yaml
serviceTop:
  doris_fe:
    nodes: [realtime-dw1, realtime-dw2]
    id_auto_derive: true
```

**步骤2：在serverConfig添加配置**
```yaml
serverConfig:
  doris_fe:
    version: "2.0.3"
    http_port: 8030
    query_port: 9030
```

**步骤3：创建模板文件**
```
templates/doris_fe/fe.conf.tmpl
templates/doris_fe/install.sh.tmpl
```

### 3.4 节点复用

**场景**：realtime-kafka1和realtime-dw3共用一台物理机

```yaml
nodes:
  realtime-dw3:
    ip: "192.168.10.12"
    hostname: "realtime-dw3"
  
  realtime-kafka1:
    ip: "192.168.10.12"  # 与realtime-dw3共用IP
    hostname: "realtime-kafka1"

serviceTop:
  hadoop_datanode:
    nodes: [realtime-dw1, realtime-dw2, realtime-dw3]
  
  kafka:
    nodes: [realtime-kafka1, realtime-kafka2, realtime-kafka3]
```

---

## 四、ID自动推导规则

| 服务 | ID字段 | 推导规则 | 示例 |
|------|--------|---------|------|
| zookeeper | myid | 索引+1 | realtime-dw1→1, realtime-dw2→2 |
| kafka | broker_id | 索引+1 | realtime-kafka1→1, realtime-kafka2→2 |
| hadoop_namenode | namenode_id | nn+索引+1 | realtime-dw1→nn1, realtime-dw2→nn2 |
| doris_fe | fe_id | 索引+1 | 节点1→1, 节点2→2 |

---

## 五、使用方法

### 5.1 运行生成器

```bash
go run main.go
```

### 5.2 输出目录结构

```
output/
├── README.md
├── 192.168.10.10/
│   ├── zookeeper/
│   │   ├── install.sh
│   │   ├── zoo.cfg
│   │   └── myid           # 自动推导：1
│   ├── hadoop_namenode/
│   └── ...
└── 192.168.10.13/
    └── kafka/
        ├── install.sh
        └── server.properties  # broker.id=1 (自动推导)
```

---

## 六、与旧版对比

### 配置修改效率对比

| 操作 | v3版本 | v4版本 | 效率提升 |
|------|--------|--------|---------|
| 修改部署位置 | 阅读全文件(~200行) | 只读serviceTop(~50行) | **75%** |
| 修改IP | 阅读nodes部分 | 阅读nodes部分 | 相同 |
| 修改服务参数 | 阅读全文件 | 只读serverConfig | **60%** |
| 配置文件可读性 | 拓扑配置混合 | 拓扑配置分离 | **大幅提升** |

---

## 七、最佳实践

### 7.1 节点命名规范

```yaml
# 按业务模块命名
nodes:
  realtime-dw1:     # 实时数仓节点1
  realtime-kafka1:  # Kafka节点1
  realtime-es1:     # ES节点1
```

### 7.2 服务拓扑组织

```yaml
serviceTop:
  # 按服务类型分组
  # 基础服务
  zookeeper: ...
  kafka: ...
  
  # 计算引擎
  hadoop_namenode: ...
  spark_master: ...
  
  # 数据存储
  elasticsearch: ...
```

---

## 八、注意事项

1. **节点别名一致性**：serviceTop中的节点别名必须与nodes中定义的一致
2. **模板目录名**：必须与服务名一致（如`zookeeper` → `templates/zookeeper/`）
3. **模板文件后缀**：必须以`.tmpl`结尾

---

**版本**: v4.0 拓扑配置分离版  
**核心特性**: 拓扑与配置分离 + 修改操作效率提升75%+  
**维护效率**: 修改部署只需阅读50行，而非全文件
