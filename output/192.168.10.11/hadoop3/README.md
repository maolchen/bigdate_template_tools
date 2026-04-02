# Hadoop 3 部署说明

## 脚本职责分离（重要！）

为避免 **systemd** 与 **脚本启动** 的冲突，所有脚本遵循以下职责分离原则：

| 脚本类型 | 职责 | 执行的操作 | 是否启动服务 |
|---------|------|-----------|------------|
| **init_*.sh** | 初始化 | 格式化、引导、ZKFC格式化 | **否** |
| **start_*.sh** | 启动服务 | 部署systemd服务 + systemctl start | **是（通过systemd）** |
| **systemd/*.service** | 服务定义 | systemctl管理的单位文件 | - |
| **stop.sh** | 停止服务 | systemctl stop 或 kill | - |

**关键原则**：
1. **所有服务统一由 systemd 管理**，确保 `systemctl status` 状态准确
2. **init 脚本只做初始化**，不启动服务
3. **start 脚本通过 systemctl 启动服务**，不直接调用 `hdfs --daemon start`

## 部署流程概览

Hadoop 3 的部署分为以下几个阶段：

```
1. 安装二进制包（所有节点）
   ↓
2. 部署配置文件（所有节点）
   ↓
3. 创建数据目录（所有节点）
   ↓
4. 按正确顺序启动服务
   ↓
5. 验证集群状态
```

## 服务分布拓扑示例（3节点部署）

以下示例展示在 **3 台节点（dw-master1、dw-master2、dw-master3）** 上部署 Hadoop 3 的服务分布：

### 节点角色分配

| 节点 | JournalNode | NameNode | ZKFC | DataNode | ResourceManager | NodeManager |
|------|:-----------:|:--------:|:----:|:--------:|:---------------:|:-----------:|
| **dw-master1** | ✅ | ✅ Active | ✅ | ✅ | ✅ Active | ✅ |
| **dw-master2** | ✅ | ✅ Standby | ✅ | ✅ | ✅ Standby | ✅ |
| **dw-master3** | ✅ | ❌ | ❌ | ✅ | ❌ | ✅ |

### 组件分布说明

| 组件 | 部署节点 | 说明 |
|------|----------|------|
| **JournalNode** | dw-master1, dw-master2, dw-master3 | 所有 3 节点都部署，提供 HDFS HA 的共享编辑日志 |
| **NameNode** | dw-master1(Active), dw-master2(Standby) | 主备部署，dw-master1 为主，dw-master2 为备 |
| **ZKFC** | dw-master1, dw-master2 | 仅部署在 NameNode 节点，自动故障转移 |
| **DataNode** | dw-master1, dw-master2, dw-master3 | 所有 3 节点都部署，存储数据块 |
| **ResourceManager** | dw-master1(Active), dw-master2(Standby) | 主备部署，与 NameNode 同节点 |
| **NodeManager** | dw-master1, dw-master2, dw-master3 | 所有 3 节点都部署，执行 YARN 任务 |

### 部署顺序（3节点场景）

```
1. 所有节点执行基础准备
   dw-master1, dw-master2, dw-master3:
   └─ install_binary.sh
   └─ deploy_config.sh
   └─ setup_dirs.sh

2. 启动 JournalNode（所有 3 节点）
   dw-master1, dw-master2, dw-master3:
   └─ start_journalnode.sh

3. 初始化并启动 NameNode（dw-master1）
   dw-master1:
   └─ init_namenode_master.sh
   └─ systemctl start hadoop-hdfs-namenode

4. 初始化并启动 NameNode（dw-master2）
   dw-master2:
   └─ init_namenode_standby.sh
   └─ systemctl start hadoop-hdfs-namenode

5. 启动 DataNode（所有 3 节点）
   dw-master1, dw-master2, dw-master3:
   └─ start_datanode.sh

6. 启动 ZKFC（NameNode 节点）
   dw-master1, dw-master2:
   └─ start_zkfc.sh

7. 启动 YARN（所有 3 节点）
   dw-master1, dw-master2, dw-master3:
   └─ start_yarn.sh
```

### 配置要点（config.yaml）

```yaml
serviceTop:
  hadoop3:
    nodes: [dw-master1, dw-master2, dw-master3]


```

---

## 正确的启动顺序

**关键顺序（必须严格遵守）**：

```
1. 启动 JournalNode（所有 JournalNode 节点）
   └─ 执行: bash start_journalnode.sh
   └─ 内部: 部署systemd服务 + systemctl start hadoop-hdfs-journalnode
   
2. 初始化主 NameNode（仅主节点）
   └─ 执行: bash init_namenode_master.sh
   └─ 内部: hdfs namenode -format + hdfs zkfc -formatZK
   └─ 注意: 只初始化，不启动服务！
   
3. 启动主 NameNode（仅主节点）
   └─ 执行: sudo systemctl start hadoop-hdfs-namenode
   
4. 初始化备 NameNode（仅备节点）
   └─ 执行: bash init_namenode_standby.sh
   └─ 内部: hdfs namenode -bootstrapStandby
   └─ 注意: 只初始化，不启动服务！
   
5. 启动备 NameNode（仅备节点）
   └─ 执行: sudo systemctl start hadoop-hdfs-namenode
   
6. 启动 DataNode（所有 DataNode 节点）
   └─ 执行: bash start_datanode.sh
   └─ 内部: systemctl start hadoop-hdfs-datanode
   
7. 启动 ZKFC（所有 NameNode 节点）
   └─ 执行: bash start_zkfc.sh
   └─ 内部: systemctl start hadoop-hdfs-zkfc
   └─ ZKFC 自动监控 NameNode 状态并实现故障转移
   
8. 启动 YARN（ResourceManager & NodeManager）
   └─ 执行: bash start_yarn.sh
   └─ 内部: systemctl start hadoop-yarn-resourcemanager / nodemanager
```

**重要提示**：
- `init_*.sh` 脚本只执行初始化，**不会启动服务**
- 服务启动必须通过 `systemctl` 或 `start_*.sh` 脚本（内部调用 systemctl）
- 统一使用 systemd 管理，确保 `systemctl status` 状态准确

## 脚本清单

### 通用脚本（所有节点执行）
| 脚本 | 用途 | 是否启动服务 |
|------|------|------------|
| `install_binary.sh` | 安装Hadoop二进制包 | 否 |
| `deploy_config.sh` | 部署配置文件 | 否 |
| `setup_dirs.sh` | 创建数据目录 | 否 |

### 初始化脚本（只初始化，不启动服务）
| 脚本 | 用途 | 执行节点 | 启动服务 |
|------|------|----------|----------|
| `init_namenode_master.sh` | 格式化 NameNode + ZKFC | 主 NameNode | **否** |
| `init_namenode_standby.sh` | 引导备 NameNode | 备 NameNode | **否** |

### 启动脚本（部署systemd + systemctl启动）
| 脚本 | 用途 | 执行节点 | 启动服务 |
|------|------|----------|----------|
| `start_journalnode.sh` | 启动 JournalNode | JournalNode 节点 | **是（systemd）** |
| `start_datanode.sh` | 启动 DataNode | DataNode 节点 | **是（systemd）** |
| `start_zkfc.sh` | 启动 ZKFC | NameNode 节点 | **是（systemd）** |
| `start_yarn.sh` | 启动 YARN | YARN 节点 | **是（systemd）** |

### 管理脚本
| 脚本 | 用途 |
|------|------|
| `check.sh` | 验证服务状态 |
| `stop.sh` | 停止所有服务（调用 systemctl stop） |

### Systemd服务文件
```
systemd/
├── hadoop-hdfs-journalnode.service
├── hadoop-hdfs-namenode.service
├── hadoop-hdfs-datanode.service
├── hadoop-hdfs-zkfc.service
├── hadoop-yarn-resourcemanager.service
└── hadoop-yarn-nodemanager.service
```

## 详细部署步骤

### 步骤1：安装二进制包（所有节点）

**执行节点**：所有Hadoop节点

```bash
bash install_binary.sh
```

**可选参数**：
- `--force` - 强制重新安装

**作用**：
- 解压Hadoop二进制包
- 配置环境变量
- 设置目录权限

---

### 步骤2：部署配置文件（所有节点）

**执行节点**：所有Hadoop节点

```bash
bash deploy_config.sh
```

**作用**：
- 部署配置文件
- 创建workers文件

---

### 步骤3：创建数据目录（所有节点）

**执行节点**：所有Hadoop节点

```bash
bash setup_dirs.sh
```

**作用**：
- 创建HDFS数据目录
- 创建YARN目录
- 创建Socket目录

---

### 步骤4：启动JournalNode

**执行节点**：所有JournalNode节点

```bash
bash start_journalnode.sh
```

**验证**：
```bash
jps | grep JournalNode
netstat -tuln | grep 8485
```

---

### 步骤5：初始化主NameNode

**执行节点**：第一个NameNode节点（主节点）

```bash
# 1. 初始化（只执行初始化，不启动服务）
bash init_namenode_master.sh

# 2. 启动 NameNode 服务（通过 systemd）
sudo systemctl start hadoop-hdfs-namenode
```

**init_namenode_master.sh 会自动执行**：
1. 格式化NameNode（如未格式化过）
2. 格式化ZKFC（如未格式化过）
3. 部署并启用 systemd 服务

**注意**：此脚本只初始化，**不会启动服务**！必须手动执行 `systemctl start`

**验证**：
```bash
# 检查 systemd 状态
sudo systemctl status hadoop-hdfs-namenode

# 检查进程
jps | grep NameNode

# 检查端口
netstat -tuln | grep 8020
```
hdfs haadmin -getAllServiceState
```

---

### 步骤6：初始化备NameNode

**执行节点**：第二个NameNode节点（备节点）

```bash
# 1. 初始化（只执行初始化，不启动服务）
bash init_namenode_standby.sh

# 2. 启动 NameNode 服务（通过 systemd）
sudo systemctl start hadoop-hdfs-namenode
```

**init_namenode_standby.sh 会自动执行**：
1. 引导备NameNode（bootstrapStandby，如未执行过）
2. 部署并启用 systemd 服务

**注意**：此脚本只初始化，**不会启动服务**！必须手动执行 `systemctl start`

**验证**：
```bash
# 检查 systemd 状态
sudo systemctl status hadoop-hdfs-namenode

# 检查 NameNode HA 状态
hdfs haadmin -getAllServiceState
```

---

### 步骤7：启动DataNode

**执行节点**：所有DataNode节点

```bash
bash start_datanode.sh
```

**说明**：此脚本会自动部署 systemd 服务并启动

**验证**：
```bash
# 检查 systemd 状态
sudo systemctl status hadoop-hdfs-datanode

# 检查进程
jps | grep DataNode

# 查看 DataNode 报告
hdfs dfsadmin -report
```
```

---

### 步骤8：启动ZKFC

**执行节点**：所有NameNode节点

```bash
bash start_zkfc.sh
```

**说明**：此脚本会自动部署 systemd 服务并启动

**验证**：
```bash
# 检查 systemd 状态
sudo systemctl status hadoop-hdfs-zkfc

# 检查进程
jps | grep DFSZKFailoverController

# 检查 HA 状态
hdfs haadmin -getAllServiceState
```

---

### 步骤9：启动YARN

**执行节点**：所有YARN节点

```bash
bash start_yarn.sh
```

**说明**：此脚本会自动部署 systemd 服务并启动 ResourceManager/NodeManager

**验证**：
```bash
# 检查 ResourceManager 状态
sudo systemctl status hadoop-yarn-resourcemanager

# 检查 NodeManager 状态
sudo systemctl status hadoop-yarn-nodemanager

# 检查进程
jps | grep -E 'ResourceManager|NodeManager'

# 查看节点列表
yarn node -list
```

---

### 步骤10：验证集群状态

**执行节点**：任意Hadoop节点

```bash
bash check.sh
```

---

## 服务管理

### 日常管理命令（推荐）

所有服务都通过 systemd 管理，使用统一的命令格式：

```bash
# 查看服务状态
sudo systemctl status hadoop-hdfs-namenode
sudo systemctl status hadoop-hdfs-datanode
sudo systemctl status hadoop-hdfs-journalnode
sudo systemctl status hadoop-hdfs-zkfc
sudo systemctl status hadoop-yarn-resourcemanager
sudo systemctl status hadoop-yarn-nodemanager

# 启动服务
sudo systemctl start hadoop-hdfs-namenode

# 停止服务
sudo systemctl stop hadoop-hdfs-namenode

# 重启服务
sudo systemctl restart hadoop-hdfs-namenode

# 查看日志
sudo journalctl -u hadoop-hdfs-namenode -f
```

### 批量停止服务

```bash
# 停止所有 Hadoop 服务
bash stop.sh
```

**说明**：`stop.sh` 会调用 `systemctl stop` 停止所有 Hadoop 相关服务，比手动逐个停止更方便。

1. **一键停止所有服务**：可以快速停止所有Hadoop组件
2. **细粒度控制**：可以停止单个组件（如 `bash stop.sh --component namenode`）
3. **备用方案**：在没有 systemd 的环境中也可使用
4. **正确的停止顺序**：自动按正确的逆序停止服务（避免依赖问题）

**建议**：
- 日常使用：使用 systemd 管理单个服务
- 维护/升级：使用 stop.sh 批量停止服务

### 使用脚本停止

```bash
# 停止所有服务
bash stop.sh --all

# 停止单个组件
bash stop.sh --component namenode
```

### 使用systemd管理

```bash
# 启动服务
systemctl start hadoop-hdfs-journalnode
systemctl start hadoop-hdfs-namenode
systemctl start hadoop-hdfs-datanode
systemctl start hadoop-hdfs-zkfc
systemctl start hadoop-yarn-resourcemanager
systemctl start hadoop-yarn-nodemanager

# 停止服务
systemctl stop hadoop-hdfs-namenode

# 设置开机自启
systemctl enable hadoop-hdfs-namenode

# 查看状态
systemctl status hadoop-hdfs-namenode
```

---

## Web UI

- **NameNode Web UI**: http://<namenode-host>:9870
- **ResourceManager Web UI**: http://<resourcemanager-host>:8088
- **DataNode Web UI**: http://<datanode-host>:9864
- **NodeManager Web UI**: http://<nodemanager-host>:8042

---

## 故障排查

### JournalNode无法启动

**检查项**：
1. 数据目录是否存在：`ls -la /data/dfs/jn`
2. 目录权限是否正确：`stat /data/dfs/jn`
3. 端口是否被占用：`netstat -tuln | grep 8485`

### NameNode格式化失败

**可能原因**：
1. JournalNode未启动或数量不足
2. JournalNode端口不可达

**解决方案**：
```bash
# 检查JournalNode状态
for jn in journalnode1 journalnode2 journalnode3; do
    echo "Checking $jn..."
    timeout 5 bash -c "echo > /dev/tcp/$jn/8485" && echo "OK" || echo "FAIL"
done
```

### ZKFC无法启动

**可能原因**：
1. NameNode未启动
2. ZooKeeper未运行
3. ZKFC未格式化

**解决方案**：
```bash
# 检查ZooKeeper状态
echo stat | nc <zookeeper-host> 2181
```

---

## 注意事项

1. **启动顺序至关重要**：必须严格按照上述顺序启动
2. **格式化仅执行一次**：首次部署时执行，不要重复执行
3. **initializeSharedEdits的重要性**：在JournalNode中初始化共享编辑日志
4. **ZKFC格式化时机**：必须在启动NameNode进程之前
5. **统一使用systemd**：避免脚本启动与systemd冲突

---

## 目录结构

```
/data/
├── localization/hadoop/          # HADOOP_HOME
│   ├── bin/
│   ├── sbin/
│   ├── lib/
│   ├── etc/hadoop/               # 配置文件目录
│   ├── logs/                     # 日志目录
│   └── pids/                     # PID文件目录
│
├── dfs/
│   ├── nn/                       # NameNode数据目录
│   ├── dn/                       # DataNode数据目录
│   ├── jn/                       # JournalNode数据目录
│   └── tmp/                      # 临时目录
│
└── hadoop/
    ├── hdfs-sockets/dn/          # Socket目录（短路读取）
    └── hadoop-yarn/
        ├── cache/                # NodeManager本地缓存
        └── containers/           # NodeManager容器日志
```
