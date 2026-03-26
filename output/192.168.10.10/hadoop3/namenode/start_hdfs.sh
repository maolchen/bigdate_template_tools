#!/bin/bash
# Hadoop 3 HDFS 服务启动脚本
# 使用方法: bash start_hdfs.sh [--format] [--format-zk]
#   --format     格式化NameNode（首次启动时使用，仅Master节点）
#   --format-zk  格式化ZKFC（首次启动时使用，仅第一个NameNode）
# 参考: tmp/hadoop3/tasks/05_service_startup.yml

set -e

# ============================================================
# 全局变量（遵循 GLOBAL_VARS_GUIDE.md 规范）
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

# ============================================================
# 辅助函数
# ============================================================
run_as_root() {
    if [ "$RUN_USER" = "root" ]; then
        "$@"
    else
        sudo "$@"
    fi
}

run_as_user() {
    if [ "$RUN_USER" = "root" ]; then
        "$@"
    else
        sudo -u ${RUN_USER} "$@"
    fi
}

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $1"
}

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $1" >&2
}

# 检查进程是否运行
is_process_running() {
    local process_name="$1"
    pgrep -f "${process_name}" > /dev/null 2>&1
}

# 等待进程启动
wait_for_process() {
    local process_name="$1"
    local max_wait="${2:-30}"
    local count=0
    
    while [ $count -lt $max_wait ]; do
        if is_process_running "${process_name}"; then
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    return 1
}

# 检查端口
check_port() {
    local host="$1"
    local port="$2"
    timeout 5 bash -c "echo > /dev/tcp/${host}/${port}" 2>/dev/null
}

# ============================================================
# Hadoop 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
NAMESERVICE="zhugeio"
NAMENODE_ID="nn1"
NAMENODE_RPC_PORT="8020"
NAMENODE_HTTP_PORT="50070"
JOURNALNODE_PORT="8485"
NAMENODE_DATA_DIR="${DATA_BASE_DIR}/dfs/nn"

# 获取当前节点在NameNode列表中的索引（用于判断Master/Standby）
IS_FIRST_NAMENODE=true

# ============================================================
# 参数解析
# ============================================================
FORMAT_NN=false
FORMAT_ZK=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --format)
            FORMAT_NN=true
            shift
            ;;
        --format-zk)
            FORMAT_ZK=true
            shift
            ;;
        *)
            log_error "未知参数: $1"
            echo "使用方法: bash start_hdfs.sh [--format] [--format-zk]"
            exit 1
            ;;
    esac
done

# ============================================================
# 开始启动
# ============================================================
echo "============================================"
echo "Hadoop 3 HDFS 服务启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "NameNode ID: ${NAMENODE_ID}"
echo "Nameservice: ${NAMESERVICE}"
echo "HADOOP_HOME: ${HADOOP_HOME}"
echo "是否为第一个NameNode: ${IS_FIRST_NAMENODE:-false}"
echo "格式化NameNode: ${FORMAT_NN}"
echo "格式化ZKFC: ${FORMAT_ZK}"
echo "运行用户: ${RUN_USER}"
echo "============================================"

# 设置环境变量
export JAVA_HOME=${JAVA_HOME}
export HADOOP_HOME=${HADOOP_HOME}
export HADOOP_CONF_DIR=${HADOOP_HOME}/etc/hadoop
export PATH=${PATH}:${HADOOP_HOME}/bin:${HADOOP_HOME}/sbin

cd ${HADOOP_HOME}

# ============================================================
# 步骤1: 检查前置条件
# ============================================================
log_info "步骤1: 检查前置条件..."

# 检查安装目录
if [ ! -d "${HADOOP_HOME}" ]; then
    log_error "Hadoop未安装，请先执行 install.sh"
    exit 1
fi

# 检查配置文件
if [ ! -f "${HADOOP_CONF_DIR}/hdfs-site.xml" ]; then
    log_error "配置文件不存在，请先执行 deploy_config.sh"
    exit 1
fi

# 检查数据目录
if [ ! -d "${NAMENODE_DATA_DIR}" ]; then
    log_warn "数据目录不存在，请先执行 setup_dirs.sh"
fi

log_info "前置条件检查通过"

# ============================================================
# 步骤2: 检查 ZooKeeper 状态
# ============================================================
log_info "步骤2: 检查 ZooKeeper 状态..."

ZK_READY=false
if check_port "192.168.10.10" "2181"; then
    log_info "ZooKeeper dw-master1:192.168.10.10:2181 可连接"
    ZK_READY=true
fi
if check_port "192.168.10.11" "2181"; then
    log_info "ZooKeeper dw-master2:192.168.10.11:2181 可连接"
    ZK_READY=true
fi
if check_port "192.168.10.12" "2181"; then
    log_info "ZooKeeper dw-master3:192.168.10.12:2181 可连接"
    ZK_READY=true
fi

if [ "$ZK_READY" = "false" ]; then
    log_error "无法连接到任何 ZooKeeper 节点"
    log_error "请确保 ZooKeeper 集群已启动"
    exit 1
fi

# ============================================================
# 步骤3: 检查 JournalNode 状态
# ============================================================
log_info "步骤3: 检查 JournalNode 状态..."

JN_READY_COUNT=0
JN_TOTAL=3
if check_port "192.168.10.10" "${JOURNALNODE_PORT}"; then
    log_info "JournalNode dw-master1:192.168.10.10:${JOURNALNODE_PORT} 可连接"
    JN_READY_COUNT=$((JN_READY_COUNT + 1))
else
    log_warn "JournalNode dw-master1:192.168.10.10:${JOURNALNODE_PORT} 无法连接"
fi
if check_port "192.168.10.11" "${JOURNALNODE_PORT}"; then
    log_info "JournalNode dw-master2:192.168.10.11:${JOURNALNODE_PORT} 可连接"
    JN_READY_COUNT=$((JN_READY_COUNT + 1))
else
    log_warn "JournalNode dw-master2:192.168.10.11:${JOURNALNODE_PORT} 无法连接"
fi
if check_port "192.168.10.12" "${JOURNALNODE_PORT}"; then
    log_info "JournalNode dw-master3:192.168.10.12:${JOURNALNODE_PORT} 可连接"
    JN_READY_COUNT=$((JN_READY_COUNT + 1))
else
    log_warn "JournalNode dw-master3:192.168.10.12:${JOURNALNODE_PORT} 无法连接"
fi

log_info "JournalNode 就绪数量: ${JN_READY_COUNT}/${JN_TOTAL}"

if [ ${JN_READY_COUNT} -lt 1 ]; then
    log_error "没有可用的 JournalNode，请先启动 JournalNode 服务"
    exit 1
fi

# ============================================================
# 步骤4: 格式化 ZKFC（仅第一个NameNode，首次启动）
# ============================================================
if [ "$FORMAT_ZK" = "true" ] && [ "${IS_FIRST_NAMENODE:-false}" = "true" ]; then
    log_info "步骤4: 格式化 ZKFC..."
    log_warn "此操作只需要在一个 NameNode 上执行一次"
    
    # 检查是否已格式化
    if ${HADOOP_HOME}/bin/zkCli.sh ls /hadoop-hdfs/${NAMESERVICE} 2>/dev/null | grep -q "zookeeper"; then
        log_info "ZKFC 已经格式化过，跳过"
    else
        echo "yes" | run_as_user ${HADOOP_HOME}/bin/hdfs zkfc -formatZK
        log_info "ZKFC 格式化完成"
    fi
else
    log_info "步骤4: 跳过 ZKFC 格式化（不是第一个 NameNode 或未指定 --format-zk）"
fi

# ============================================================
# 步骤5: 启动 ZKFC（所有 NameNode）
# ============================================================
log_info "步骤5: 启动 ZKFC..."

if is_process_running "org.apache.hadoop.hdfs.tools.DFSZKFailoverController"; then
    log_info "ZKFC 已在运行中"
else
    run_as_user ${HADOOP_HOME}/bin/hdfs --daemon start zkfc
    sleep 5
    
    if is_process_running "org.apache.hadoop.hdfs.tools.DFSZKFailoverController"; then
        log_info "ZKFC 启动成功 (PID: $(pgrep -f 'DFSZKFailoverController'))"
    else
        log_error "ZKFC 启动失败"
        exit 1
    fi
fi

# ============================================================
# 步骤6: 格式化 NameNode（首次启动）
# ============================================================
if [ "$FORMAT_NN" = "true" ]; then
    log_info "步骤6: 格式化 NameNode..."
    
    # 检查是否已格式化
    if [ -d "${NAMENODE_DATA_DIR}/current" ]; then
        log_warn "NameNode 已格式化（存在 current 目录）"
        if [ "${IS_FIRST_NAMENODE:-false}" = "true" ]; then
            read -p "是否重新格式化？这将清空所有数据！(yes/no): " CONFIRM
            if [ "$CONFIRM" != "yes" ]; then
                log_info "跳过格式化"
                FORMAT_NN=false
            fi
        fi
    fi
    
    if [ "$FORMAT_NN" = "true" ]; then
        if [ "${IS_FIRST_NAMENODE:-false}" = "true" ]; then
            # Master: 完整格式化
            log_info "格式化 NameNode Master..."
            echo "yes" | run_as_user ${HADOOP_HOME}/bin/hdfs namenode -format -clusterId ${NAMESERVICE}
        else
            # Standby: 从 Master 同步
            log_info "等待 NameNode Master 启动..."
            sleep 10
            
            log_info "引导 NameNode Standby（从 Master 同步）..."
            run_as_user ${HADOOP_HOME}/bin/hdfs namenode -bootstrapStandby -force
        fi
        log_info "NameNode 格式化完成"
    fi
else
    log_info "步骤6: 跳过 NameNode 格式化（未指定 --format）"
fi

# ============================================================
# 步骤7: 启动 NameNode
# ============================================================
log_info "步骤7: 启动 NameNode..."

if is_process_running "org.apache.hadoop.hdfs.server.namenode.NameNode"; then
    log_info "NameNode 已在运行中"
else
    run_as_user ${HADOOP_HOME}/bin/hdfs --daemon start namenode
    sleep 10
    
    if is_process_running "org.apache.hadoop.hdfs.server.namenode.NameNode"; then
        log_info "NameNode 启动成功 (PID: $(pgrep -f 'NameNode'))"
    else
        log_error "NameNode 启动失败"
        log_error "请检查日志: ${HADOOP_HOME}/logs/hadoop-*-namenode-*.log"
        exit 1
    fi
fi

# ============================================================
# 步骤8: 验证 HDFS 状态
# ============================================================
log_info "步骤8: 验证 HDFS 状态..."
sleep 5

# 获取 NameNode 状态
NN_STATE=$(${HADOOP_HOME}/bin/hdfs haadmin -getServiceState ${NAMENODE_ID} 2>/dev/null || echo "未知")
log_info "当前 NameNode 状态: ${NN_STATE}"

# 显示所有 NameNode 状态
echo ""
echo "NameNode HA 状态:"
STATE=$(${HADOOP_HOME}/bin/hdfs haadmin -getServiceState "nn1" 2>/dev/null || echo "未知")
echo "  nn1 (dw-master1): ${STATE}"
STATE=$(${HADOOP_HOME}/bin/hdfs haadmin -getServiceState "nn2" 2>/dev/null || echo "未知")
echo "  nn2 (dw-master2): ${STATE}"

# ============================================================
# 步骤9: 检查端口
# ============================================================
log_info "步骤9: 检查端口..."

if check_port "localhost" "${NAMENODE_RPC_PORT}"; then
    log_info "RPC端口 ${NAMENODE_RPC_PORT} 正在监听"
else
    log_warn "RPC端口 ${NAMENODE_RPC_PORT} 未监听"
fi

if check_port "localhost" "${NAMENODE_HTTP_PORT}"; then
    log_info "HTTP端口 ${NAMENODE_HTTP_PORT} 正在监听"
    echo ""
    echo "Web UI: http://$(hostname):${NAMENODE_HTTP_PORT}"
else
    log_warn "HTTP端口 ${NAMENODE_HTTP_PORT} 未监听"
fi

# ============================================================
# 启动完成
# ============================================================
echo ""
echo "============================================"
echo "HDFS 服务启动完成！"
echo "============================================"
echo ""
echo "服务状态:"
echo "  ZKFC: $(is_process_running 'DFSZKFailoverController' && echo '运行中' || echo '未运行')"
echo "  NameNode: $(is_process_running 'NameNode' && echo '运行中' || echo '未运行')"
echo ""
echo "验证命令:"
echo "  hdfs haadmin -getAllServiceState"
echo "  hdfs dfsadmin -report"
echo ""
