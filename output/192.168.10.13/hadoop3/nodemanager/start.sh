#!/bin/bash
# Hadoop 3 YARN NodeManager 服务启动脚本
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
NM_LOCAL_DIR="${DATA_BASE_DIR}/hadoop3/hadoop-yarn/cache"
NM_LOG_DIR="${DATA_BASE_DIR}/hadoop3/hadoop-yarn/containers"

# ============================================================
# 开始启动
# ============================================================
echo "============================================"
echo "Hadoop 3 NodeManager 服务启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "HADOOP_HOME: ${HADOOP_HOME}"
echo "本地目录: ${NM_LOCAL_DIR}"
echo "日志目录: ${NM_LOG_DIR}"
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
# 步骤2: 检查 ResourceManager 状态
# ============================================================
log_info "步骤2: 检查 ResourceManager 状态..."

RM_PORT="<no value>"
RM_READY=false
if check_port "192.168.10.10" "${RM_PORT}"; then
    log_info "ResourceManager dw-master1:192.168.10.10:${RM_PORT} 可连接"
    RM_READY=true
fi
if check_port "192.168.10.11" "${RM_PORT}"; then
    log_info "ResourceManager dw-master2:192.168.10.11:${RM_PORT} 可连接"
    RM_READY=true
fi

if [ "$RM_READY" = "false" ]; then
    log_warn "无法连接到 ResourceManager"
    log_warn "请确保 ResourceManager 已启动"
    read -p "是否继续启动 NodeManager? (y/n): " CONTINUE
    if [ "$CONTINUE" != "y" ]; then
        log_info "启动已取消"
        exit 0
    fi
fi

# ============================================================
# 步骤3: 准备数据目录
# ============================================================
log_info "步骤3: 准备数据目录..."

# 本地目录
if [ ! -d "${NM_LOCAL_DIR}" ]; then
    run_as_root mkdir -p "${NM_LOCAL_DIR}"
    run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${NM_LOCAL_DIR}"
    log_info "本地目录已创建: ${NM_LOCAL_DIR}"
else
    log_info "本地目录已存在: ${NM_LOCAL_DIR}"
fi

# 日志目录
if [ ! -d "${NM_LOG_DIR}" ]; then
    run_as_root mkdir -p "${NM_LOG_DIR}"
    run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${NM_LOG_DIR}"
    log_info "日志目录已创建: ${NM_LOG_DIR}"
else
    log_info "日志目录已存在: ${NM_LOG_DIR}"
fi

# Hadoop日志和PID目录
run_as_root mkdir -p "${HADOOP_HOME}/logs"
run_as_root mkdir -p "${HADOOP_HOME}/pids"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/logs"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/pids"

# ============================================================
# 步骤4: 检查是否已运行
# ============================================================
log_info "步骤4: 检查服务状态..."

if is_process_running "org.apache.hadoop.yarn.server.nodemanager.NodeManager"; then
    log_info "NodeManager 已在运行中"
    echo "进程ID: $(pgrep -f 'NodeManager')"
    exit 0
fi

# ============================================================
# 步骤5: 启动 NodeManager
# ============================================================
log_info "步骤5: 启动 NodeManager..."

cd ${HADOOP_HOME}
run_as_user ${HADOOP_HOME}/bin/yarn --daemon start nodemanager

sleep 10

# ============================================================
# 步骤6: 验证启动
# ============================================================
log_info "步骤6: 验证启动..."

if is_process_running "org.apache.hadoop.yarn.server.nodemanager.NodeManager"; then
    PID=$(pgrep -f 'org.apache.hadoop.yarn.server.nodemanager.NodeManager')
    log_info "NodeManager 启动成功 (PID: ${PID})"
else
    log_error "NodeManager 启动失败"
    log_error "请检查日志: ${HADOOP_HOME}/logs/yarn-*-nodemanager-*.log"
    exit 1
fi

# ============================================================
# 启动完成
# ============================================================
echo ""
echo "============================================"
echo "NodeManager 启动完成！"
echo "============================================"
echo "进程ID: ${PID}"
echo "本地目录: ${NM_LOCAL_DIR}"
echo "日志目录: ${NM_LOG_DIR}"
echo ""
echo "验证命令（在 ResourceManager 节点执行）:"
echo "  yarn node -list"
