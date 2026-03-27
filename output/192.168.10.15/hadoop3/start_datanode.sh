#!/bin/bash
# Hadoop 3 DataNode 启动脚本
# 使用方法: bash start_datanode.sh
# 说明: 使用 systemd 启动 DataNode 服务
# 注意: 必须确保 NameNode 已启动并处于 Active 状态

set -e

# ============================================================
# 全局变量
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

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $1" >&2
}

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $1"
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SUCCESS] $1"
}

check_port() {
    local host=$1
    local port=$2
    if timeout 5 bash -c "echo > /dev/tcp/${host}/${port}" 2>/dev/null; then
        return 0
    fi
    return 1
}

# ============================================================
# Hadoop 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
HADOOP_CONF_DIR="${HADOOP_HOME}/etc/hadoop"
DN_DATA_DIR="${DATA_BASE_DIR}/dfs/dn"
SOCKETS_DIR="${DATA_BASE_DIR}/hadoop/hdfs-sockets/dn"
NN_RPC_PORT="8020"
DN_DATA_PORT="9866"
DN_HTTP_PORT="9864"
SERVICE_NAME="hadoop-hdfs-datanode"

# ============================================================
# 开始启动
# ============================================================
echo "============================================"
echo "Hadoop 3 DataNode 启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "DataNode 数据目录: ${DN_DATA_DIR}"
echo "============================================"

# ============================================================
# 步骤1: 检查前置条件
# ============================================================
log_info "步骤1: 检查前置条件..."

# 检查Hadoop安装
if [ ! -d "${HADOOP_HOME}" ]; then
    log_error "Hadoop未安装，请先执行 install_binary.sh"
    exit 1
fi

# 检查配置文件
if [ ! -f "${HADOOP_CONF_DIR}/hdfs-site.xml" ]; then
    log_error "配置文件不存在，请先执行 deploy_config.sh"
    exit 1
fi

# 检查数据目录
if [ ! -d "${DN_DATA_DIR}" ]; then
    log_error "数据目录不存在，请先执行 setup_dirs.sh"
    exit 1
fi

log_info "前置条件检查通过"

# ============================================================
# 步骤2: 检查 NameNode 状态
# ============================================================
log_info "步骤2: 检查 NameNode 状态..."
log_warn "必须确保至少有一个 NameNode 处于 Active 状态"

NN_READY=false

if [ "$NN_READY" = "false" ]; then
    log_warn "无法连接到任何 NameNode"
    log_warn "请确保 NameNode 已启动并处于 Active 状态"
    read -p "是否继续启动 DataNode? (y/n): " CONTINUE
    if [ "$CONTINUE" != "y" ]; then
        log_info "启动已取消"
        exit 0
    fi
fi

# ============================================================
# 步骤3: 准备数据目录和 Socket 目录
# ============================================================
log_info "步骤3: 准备数据目录和 Socket 目录..."

# 创建数据目录
if [ ! -d "${DN_DATA_DIR}" ]; then
    run_as_root mkdir -p "${DN_DATA_DIR}"
    run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${DN_DATA_DIR}"
    log_info "数据目录已创建: ${DN_DATA_DIR}"
else
    run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${DN_DATA_DIR}"
    log_info "数据目录已存在: ${DN_DATA_DIR}"
fi

# 创建 Socket 目录（短路读取）
if [ ! -d "${SOCKETS_DIR}" ]; then
    run_as_root mkdir -p "${SOCKETS_DIR}"
fi
run_as_root chmod 755 "${SOCKETS_DIR}"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${SOCKETS_DIR}"
run_as_root chmod g-w "${SOCKETS_DIR}"
log_info "Socket 目录已准备: ${SOCKETS_DIR}"

# ============================================================
# 步骤4: 部署并启用 systemd 服务
# ============================================================
log_info "步骤4: 部署 systemd 服务文件..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_FILE="${SCRIPT_DIR}/systemd/${SERVICE_NAME}.service"

if [ ! -f "${SERVICE_FILE}" ]; then
    log_error "未找到 systemd 服务文件: ${SERVICE_FILE}"
    exit 1
fi

# 部署服务文件
run_as_root cp "${SERVICE_FILE}" /etc/systemd/system/
run_as_root chmod 644 /etc/systemd/system/${SERVICE_NAME}.service
run_as_root systemctl daemon-reload
log_info "systemd 服务文件已部署"

# 启用开机自启动
run_as_root systemctl enable ${SERVICE_NAME}
log_info "systemd 服务已启用"

# ============================================================
# 步骤5: 检查服务状态
# ============================================================
log_info "步骤5: 检查 DataNode 状态..."

if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
    log_info "DataNode 已在运行"
    systemctl status ${SERVICE_NAME} --no-pager
    exit 0
fi

# ============================================================
# 步骤6: 使用 systemctl 启动服务
# ============================================================
log_info "步骤6: 使用 systemctl 启动 DataNode..."

run_as_root systemctl start ${SERVICE_NAME}
sleep 3

# ============================================================
# 步骤7: 验证启动
# ============================================================
log_info "步骤7: 验证启动..."

# 检查 systemd 状态
if systemctl is-active --quiet ${SERVICE_NAME}; then
    log_success "DataNode 启动成功"
    systemctl status ${SERVICE_NAME} --no-pager
else
    log_error "DataNode 启动失败"
    log_error "查看日志: sudo journalctl -u ${SERVICE_NAME} -n 50"
    exit 1
fi

# 检查端口
sleep 2
if check_port "localhost" "${DN_DATA_PORT}"; then
    log_success "数据传输端口 ${DN_DATA_PORT} 正在监听"
else
    log_warn "数据传输端口 ${DN_DATA_PORT} 未监听"
fi

if check_port "localhost" "${DN_HTTP_PORT}"; then
    log_success "HTTP 端口 ${DN_HTTP_PORT} 正在监听"
else
    log_warn "HTTP 端口 ${DN_HTTP_PORT} 未监听"
fi

# ============================================================
# 启动完成
# ============================================================
echo ""
echo "============================================"
echo "DataNode 启动完成！"
echo "============================================"
echo ""
echo "管理命令:"
echo "  查看状态: sudo systemctl status ${SERVICE_NAME}"
echo "  停止服务: sudo systemctl stop ${SERVICE_NAME}"
echo "  重启服务: sudo systemctl restart ${SERVICE_NAME}"
echo "  查看日志: sudo journalctl -u ${SERVICE_NAME} -f"
echo ""
echo "HDFS 命令:"
echo "  hdfs dfsadmin -report"
echo ""
echo "后续步骤:"
echo "  启动 ZKFC: bash start_zkfc.sh"
