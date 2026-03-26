#!/bin/bash
# Hadoop 3 YARN ResourceManager 服务启动脚本
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

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $1"
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
YARN_CLUSTER_ID="zhugeio2"
RM_ID="rm2"
RM_PORT="8032"
RM_WEB_PORT="8089"

# ============================================================
# 开始启动
# ============================================================
echo "============================================"
echo "Hadoop 3 ResourceManager 服务启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "ResourceManager ID: ${RM_ID}"
echo "YARN集群ID: ${YARN_CLUSTER_ID}"
echo "HADOOP_HOME: ${HADOOP_HOME}"
echo "RPC端口: ${RM_PORT}"
echo "Web端口: ${RM_WEB_PORT}"
echo "运行用户: ${RUN_USER}"
echo "============================================"

# 设置环境变量
export JAVA_HOME=${JAVA_HOME}
export HADOOP_HOME=${HADOOP_HOME}
export HADOOP_CONF_DIR=${HADOOP_HOME}/etc/hadoop
export YARN_CONF_DIR=${HADOOP_HOME}/etc/hadoop
export PATH=${PATH}:${HADOOP_HOME}/bin:${HADOOP_HOME}/sbin

# ============================================================
# 步骤1: 检查前置条件
# ============================================================
log_info "步骤1: 检查前置条件..."

if [ ! -d "${HADOOP_HOME}" ]; then
    log_error "Hadoop未安装，请先执行 install.sh"
    exit 1
fi

if [ ! -f "${HADOOP_CONF_DIR}/yarn-site.xml" ]; then
    log_error "配置文件不存在，请先执行 deploy_config.sh"
    exit 1
fi

log_info "前置条件检查通过"

# ============================================================
# 步骤2: 检查 HDFS 状态
# ============================================================
log_info "步骤2: 检查 HDFS 状态..."

NAMENODE_RPC_PORT="<no value>"
HDFS_READY=false
if check_port "192.168.10.10" "${NAMENODE_RPC_PORT}"; then
    log_info "NameNode dw-master1:192.168.10.10:${NAMENODE_RPC_PORT} 可连接"
    HDFS_READY=true
fi
if check_port "192.168.10.11" "${NAMENODE_RPC_PORT}"; then
    log_info "NameNode dw-master2:192.168.10.11:${NAMENODE_RPC_PORT} 可连接"
    HDFS_READY=true
fi

if [ "$HDFS_READY" = "false" ]; then
    log_warn "无法连接到 NameNode"
    log_warn "请确保 HDFS 已启动"
    read -p "是否继续启动 ResourceManager? (y/n): " CONTINUE
    if [ "$CONTINUE" != "y" ]; then
        log_info "启动已取消"
        exit 0
    fi
fi

# ============================================================
# 步骤3: 检查 ZooKeeper 状态
# ============================================================
log_info "步骤3: 检查 ZooKeeper 状态..."

ZK_READY=false
ZK_PORT=2181
if check_port "192.168.10.10" "${ZK_PORT}"; then
    log_info "ZooKeeper dw-master1:192.168.10.10:${ZK_PORT} 可连接"
    ZK_READY=true
fi
if check_port "192.168.10.11" "${ZK_PORT}"; then
    log_info "ZooKeeper dw-master2:192.168.10.11:${ZK_PORT} 可连接"
    ZK_READY=true
fi
if check_port "192.168.10.12" "${ZK_PORT}"; then
    log_info "ZooKeeper dw-master3:192.168.10.12:${ZK_PORT} 可连接"
    ZK_READY=true
fi

if [ "$ZK_READY" = "false" ]; then
    log_warn "无法连接到 ZooKeeper"
    log_warn "ResourceManager HA 需要 ZooKeeper 支持"
fi

# ============================================================
# 步骤4: 准备日志和PID目录
# ============================================================
log_info "步骤4: 准备日志和PID目录..."

run_as_root mkdir -p "${HADOOP_HOME}/logs"
run_as_root mkdir -p "${HADOOP_HOME}/pids"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/logs"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/pids"

# ============================================================
# 步骤5: 检查是否已运行
# ============================================================
log_info "步骤5: 检查服务状态..."

if is_process_running "org.apache.hadoop.yarn.server.resourcemanager.ResourceManager"; then
    log_info "ResourceManager 已在运行中"
    echo "进程ID: $(pgrep -f 'ResourceManager')"
    exit 0
fi

# ============================================================
# 步骤6: 启动 ResourceManager
# ============================================================
log_info "步骤6: 启动 ResourceManager..."

cd ${HADOOP_HOME}
run_as_user ${HADOOP_HOME}/bin/yarn --daemon start resourcemanager

sleep 10

# ============================================================
# 步骤7: 验证启动
# ============================================================
log_info "步骤7: 验证启动..."

if is_process_running "org.apache.hadoop.yarn.server.resourcemanager.ResourceManager"; then
    PID=$(pgrep -f 'org.apache.hadoop.yarn.server.resourcemanager.ResourceManager')
    log_info "ResourceManager 启动成功 (PID: ${PID})"
else
    log_error "ResourceManager 启动失败"
    log_error "请检查日志: ${HADOOP_HOME}/logs/yarn-*-resourcemanager-*.log"
    exit 1
fi

# 检查端口
if check_port "localhost" "${RM_PORT}"; then
    log_info "RPC端口 ${RM_PORT} 正在监听"
else
    log_warn "RPC端口 ${RM_PORT} 未监听"
fi

if check_port "localhost" "${RM_WEB_PORT}"; then
    log_info "Web端口 ${RM_WEB_PORT} 正在监听"
    echo ""
    echo "Web UI: http://$(hostname):${RM_WEB_PORT}"
fi

# ============================================================
# 步骤8: 检查 RM HA 状态
# ============================================================
log_info "步骤8: 检查 RM HA 状态..."
sleep 5

echo ""
echo "ResourceManager HA 状态:"
STATE=$(${HADOOP_HOME}/bin/yarn rmadmin -getServiceState "rm1" 2>/dev/null || echo "未知")
echo "  rm1 (dw-master1): ${STATE}"
STATE=$(${HADOOP_HOME}/bin/yarn rmadmin -getServiceState "rm2" 2>/dev/null || echo "未知")
echo "  rm2 (dw-master2): ${STATE}"

# ============================================================
# 启动完成
# ============================================================
echo ""
echo "============================================"
echo "ResourceManager 启动完成！"
echo "============================================"
echo "进程ID: ${PID}"
echo "RPC端口: ${RM_PORT}"
echo "Web端口: ${RM_WEB_PORT}"
echo ""
echo "验证命令:"
echo "  yarn rmadmin -getAllServiceState"
echo "  yarn node -list"
