# 大数据平台离线配置生成器 v3.0 - 极简版

## 一、v3核心改进

### 1. 节点池集中管理
**修改IP只需改一处**，无需遍历整个配置文件：

```yaml
# 所有服务器的IP/hostname集中在这里定义
nodes:
  node1:
    ip: "192.168.10.10"
    hostname: "master01"
  node2:
    ip: "192.168.10.11"
    hostname: "master02"
  # ...
```

### 2. 服务拓扑简洁描述
**用节点别名列表描述部署位置**，清晰直观：

```yaml
services:
  zookeeper:
    nodes: [node1, node2, node3]  # 部署在node1, node2, node3
    id_auto_derive: true          # 自动推导myid
    
  kafka:
    nodes: [node4, node5, node6]
    id_auto_derive: true          # 自动推导broker_id
```

### 3. ID自动推导
**myid/broker_id自动计算**，无需手动维护：

| 服务 | 节点别名 | 自动推导ID |
|------|---------|-----------|
| zookeeper | node1 | myid=1 |
| zookeeper | node2 | myid=2 |
| zookeeper | node3 | myid=3 |
| kafka | node4 | broker_id=1 |
| kafka | node5 | broker_id=2 |
| kafka | node6 | broker_id=3 |

---

## 二、配置文件结构

### 2.1 完整结构

```yaml
# 1. 全局配置（几乎不需要修改）
global:
  install_base_dir: "/data/localization"
  data_base_dir: "/data/bigdata"
  user: "bigdata"
  timezone: "Asia/Shanghai"
  # ...

# 2. 节点池（IP修改只需改这里）
nodes:
  node1:
    ip: "192.168.10.10"
    hostname: "master01"
  node2:
    ip: "192.168.10.11"
    hostname: "master02"
  # ...

# 3. 服务拓扑（描述服务部署位置）
services:
  zookeeper:
    nodes: [node1, node2, node3]
    id_auto_derive: true  # 自动推导myid
    vars:
      client_port: 2181
      # ...

# 4. 节点特化配置（可选）
node_overrides:
  node4:
    spark_worker:
      memory: "8g"  # node4内存更大
```

### 2.2 配置优先级

```
node_overrides > services.vars > global
```

---

## 三、常见操作示例

### 3.1 修改服务器IP

**只需要修改nodes部分**，所有服务自动生效：

```yaml
# 修改前
nodes:
  node1:
    ip: "192.168.10.10"
    
# 修改后
nodes:
  node1:
    ip: "10.20.30.40"  # 只改这一处
```

### 3.2 添加新服务

**只需在services下添加**，无需修改代码：

```yaml
services:
  # 添加Doris FE
  doris_fe:
    nodes: [node1, node2]
    id_auto_derive: true  # 自动推导fe_id
    vars:
      version: "2.0.3"
      http_port: 8030
      
  # 添加Doris BE
  doris_be:
    nodes: [node3, node4, node5]
    vars:
      version: "2.0.3"
      be_port: 9060
```

然后创建模板文件：
```
templates/doris_fe/fe.conf.tmpl
templates/doris_fe/install.sh.tmpl
templates/doris_be/be.conf.tmpl
templates/doris_be/install.sh.tmpl
```

### 3.3 调整服务部署位置

**只需修改nodes列表**：

```yaml
services:
  # 将Kafka从node4,5,6迁移到node7,8,9
  kafka:
    nodes: [node7, node8, node9]  # 只改这行
    id_auto_derive: true
```

### 3.4 为特定节点设置特殊配置

```yaml
node_overrides:
  # node4内存更大，提高Spark Worker配置
  node4:
    spark_worker:
      memory: "8g"
      cores: 8
      
  # node9仅作为ES数据节点
  node9:
    elasticsearch:
      node_master: false
      node_data: true
```

---

## 四、ID自动推导规则

| 服务 | ID字段名 | 推导规则 | 示例 |
|------|---------|---------|------|
| zookeeper | myid | 索引+1 | node1→1, node2→2, node3→3 |
| kafka | broker_id | 索引+1 | node4→1, node5→2, node6→3 |
| hadoop_namenode | namenode_id | nn+索引+1 | node1→nn1, node2→nn2 |
| doris_fe | fe_id | 索引+1 | node1→1, node2→2 |
| 其他服务 | instance_id | 索引+1 | 通用 |

---

## 五、使用方法

### 5.1 运行生成器

```bash
go run main.go
```

### 5.2 输出目录结构

```
output/
├── README.md                    # 部署总览
├── 192.168.10.10/              # 按IP组织
│   ├── zookeeper/
│   │   ├── install.sh
│   │   ├── zoo.cfg
│   │   └── myid               # 自动推导：1
│   ├── kafka/
│   └── ...
├── 192.168.10.11/
│   └── zookeeper/
│       └── myid               # 自动推导：2
└── ...
```

### 5.3 交付流程

```bash
# 1. 上传到目标主机
scp -r output/192.168.10.10 user@192.168.10.10:/home/user/

# 2. 在目标主机执行
cd /home/user/192.168.10.10
bash zookeeper/install.sh

# 3. 启动服务
./zookeeper/start.sh
```

---

## 六、模板编写指南

### 6.1 可用变量

```go
.Global        // 全局配置
.ServiceVars   // 服务配置
.HostVars      // 主机配置（包含自动推导的ID）
.IP            // 主机IP
.Hostname      // 主机名
.NodeName      // 节点别名（如node1）
.NodeIndex     // 节点索引（从0开始）
.AutoID        // 自动推导的ID（索引+1）
.Derived       // 衍生变量
```

### 6.2 自动推导ID使用

```properties
# Zookeeper模板
dataDir={{ globalVar "data_base_dir" }}/zookeeper/data

# 使用自动推导的myid
# {{ .AutoID }} 或 {{ .HostVars.myid }}

# 服务器列表（自动生成）
{{- range serviceNodes "zookeeper" }}
server.{{ .auto_id }}={{ .ip }}:2888:3888
{{- end }}
```

### 6.3 自定义函数

```go
{{ globalVar "data_base_dir" }}           // 获取全局变量
{{ serviceNodes "zookeeper" }}            // 获取服务节点列表
{{ serviceIPs "kafka" }}                  // 获取服务IP列表
{{ .Derived.zk_connect_string }}          // ZK连接串
{{ join .List "," }}                      // 列表连接
{{ add 1 2 }}                             // 加法
```

---

## 七、与旧版对比

### 7.1 配置复杂度对比

| 操作 | v2版本 | v3极简版 |
|------|--------|---------|
| 修改IP | 需要遍历整个配置文件 | 只改nodes部分 |
| 修改hostname | 每个服务下都要改 | 只改nodes部分 |
| 设置myid | 手动指定 | 自动推导 |
| 添加新服务 | 定义hosts、vars | 定义nodes、vars |
| 配置行数 | ~200行 | ~150行 |

### 7.2 维护效率对比

**场景：将node1的IP从192.168.10.10改为10.20.30.40**

**v2版本**：需要修改以下位置
```yaml
zookeeper:
  hosts:
    192.168.10.10:  # 改这里
kafka:
  # ...
hadoop_namenode:
  hosts:
    192.168.10.10:  # 还要改这里
# ... 需要遍历所有服务
```

**v3极简版**：只改一处
```yaml
nodes:
  node1:
    ip: "10.20.30.40"  # 只改这一处，所有服务自动生效
```

---

## 八、最佳实践

### 8.1 节点命名规范

```yaml
# 按角色命名
node1, node2, node3: master节点
node4, node5, node6: worker节点
node7, node8, node9: 专用节点（如ES）
```

### 8.2 服务拓扑设计

```yaml
# 基础服务放在前面的节点
services:
  zookeeper:
    nodes: [node1, node2, node3]
    
  # 数据服务放在中间节点
  kafka:
    nodes: [node4, node5, node6]
    
  # 计算服务放在worker节点
  spark_worker:
    nodes: [node4, node5, node6]
```

### 8.3 环境隔离

```yaml
# 开发环境
nodes:
  node1:
    ip: "192.168.1.10"
    
# 生产环境 - 只需替换nodes部分
nodes:
  node1:
    ip: "10.20.30.10"
```

---

## 九、注意事项

1. **节点别名必须一致**：services中的节点别名必须与nodes中定义的一致
2. **id_auto_derive默认false**：需要自动推导ID的服务需要显式设置为true
3. **模板目录名与服务名一致**：`services.zookeeper` → `templates/zookeeper/`
4. **模板文件必须以.tmpl结尾**：`zoo.cfg.tmpl` → 输出为 `zoo.cfg`

---

## 十、项目结构

```
bigdata-deploy-generator/
├── main.go              # 主程序（完全动态）
├── config.yaml          # 极简配置文件（~150行）
├── templates/           # 模板库
│   ├── zookeeper/
│   │   ├── zoo.cfg.tmpl
│   │   ├── myid.tmpl
│   │   └── install.sh.tmpl
│   ├── kafka/
│   │   ├── server.properties.tmpl
│   │   └── install.sh.tmpl
│   └── ...
├── output/              # 输出目录
│   ├── README.md
│   └── <ip>/
└── README.md
```

---

**版本**: v3.0 极简版  
**核心特性**: 节点池集中管理 + ID自动推导 + 配置极简化  
**维护效率**: 提升80%+
