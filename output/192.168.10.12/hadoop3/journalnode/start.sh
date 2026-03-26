#!/bin/bash
# Hadoop 3 JournalNode 服务启动脚本
# 使用方法: bash start.sh
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

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $1" >&2
}

is_process_running() {
    pgrep -f "$1" > /dev/null 2>&1
}

check_port() {
    local host="$1"
    local port="$2"
    timeout 5 bash -c "echo > /dev/tcp/${host}/${port}" 2>/dev/null
}

# ============================================================
# Hadoop 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
JOURNALNODE_DATA_DIR="${DATA_BASE_DIR}/dfs/jn"
JOURNALNODE_PORT="8485"
JOURNALNODE_ID="jn3"

# ============================================================
# 开始启动
# ============================================================
echo "============================================"
echo "Hadoop 3 JournalNode 服务启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "JournalNode ID: ${JOURNALNODE_ID}"
echo "HADOOP_HOME: ${HADOOP_HOME}"
echo "数据目录: ${JOURNALNODE_DATA_DIR}"
echo "端口: ${JOURNALNODE_PORT}"
echo "运行用户: ${RUN_USER}"
echo "============================================"

# 设置环境变量
export JAVA_HOME=${JAVA_HOME}
export HADOOP_HOME=${HADOOP_HOME}
export HADOOP_CONF_DIR=${HADOOP_HOME}/etc/hadoop
export PATH=${PATH}:${HADOOP_HOME}/bin:${HADOOP_HOME}/sbin

# ============================================================
# 步骤1: 检查前置条件
# ============================================================
log_info "步骤1: 检查前置条件..."

if [ ! -d "${HADOOP_HOME}" ]; then
    log_error "Hadoop未安装，请先执行 install.sh"
    exit 1
fi

if [ ! -f "${HADOOP_CONF_DIR}/hdfs-site.xml" ]; then
    log_error "配置文件不存在，请先执行 deploy_config.sh"
    exit 1
fi

log_info "前置条件检查通过"

# ============================================================
# 步骤2: 准备数据目录
# ============================================================
log_info "步骤2: 准备数据目录..."

if [ ! -d "${JOURNALNODE_DATA_DIR}" ]; then
    run_as_root mkdir -p "${JOURNALNODE_DATA_DIR}"
    run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${JOURNALNODE_DATA_DIR}"
    log_info "数据目录已创建: ${JOURNALNODE_DATA_DIR}"
else
    log_info "数据目录已存在: ${JOURNALNODE_DATA_DIR}"
fi

# ============================================================
# 步骤3: 检查是否已运行
# ============================================================
log_info "步骤3: 检查服务状态..."

if is_process_running "org.apache.hadoop.hdfs.qjournal.server.JournalNode"; then
    log_info "JournalNode 已在运行中"
    echo "进程ID: $(pgrep -f 'JournalNode')"
    exit 0
fi

# ============================================================
# 步骤4: 启动 JournalNode
# ============================================================
log_info "步骤4: 启动 JournalNode..."

cd ${HADOOP_HOME}
run_as_user ${HADOOP_HOME}/bin/hdfs --daemon start journalnode

sleep 5

# ============================================================
# 步骤5: 验证启动
# ============================================================
log_info "步骤5: 验证启动..."

if is_process_running "org.apache.hadoop.hdfs.qjournal.server.JournalNode"; then
    PID=$(pgrep -f 'org.apache.hadoop.hdfs.qjournal.server.JournalNode')
    log_info "JournalNode 启动成功 (PID: ${PID})"
else
    log_error "JournalNode 启动失败"
    log_error "请检查日志: ${HADOOP_HOME}/logs/hadoop-*-journalnode-*.log"
    exit 1
fi

# 检查端口
if check_port "localhost" "${JOURNALNODE_PORT}"; then
    log_info "端口 ${JOURNALNODE_PORT} 正在监听"
else
    log_error "端口 ${JOURNALNODE_PORT} 未监听"
    exit 1
fi

# ============================================================
# 启动完成
# ============================================================
echo ""
echo "============================================"
echo "JournalNode 启动完成！"
echo "============================================"
echo "进程ID: ${PID}"
echo "数据目录: ${JOURNALNODE_DATA_DIR}"
echo "监听端口: ${JOURNALNODE_PORT}"
