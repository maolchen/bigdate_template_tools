#!/bin/bash
# Doris BE 安装配置脚本
# 执行节点: BE 节点
# 用途: 安装 Doris BE 并配置

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

# Doris 配置
INSTALL_SUBDIR="apache-doris/be"
BE_DATA_SUBDIR="<no value>"
PRIORITY_NETWORKS="<no value>"

# 端口配置
BE_PORT="9060"
BE_WEBSERVER_PORT="<no value>"
BE_HEARTBEAT_PORT="<no value>"
BE_BRPC_PORT="<no value>"

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

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SUCCESS] $1"
}

# ============================================================
# 开始安装
# ============================================================
echo "============================================"
echo "Doris BE 安装配置脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "============================================"

DORIS_HOME="${INSTALL_BASE_DIR}/${INSTALL_SUBDIR}"
BE_HOME="${DORIS_HOME}/be"
BE_CONF="${BE_HOME}/conf/be.conf"
BE_DATA_DIR="${DATA_BASE_DIR}/${BE_DATA_SUBDIR}"

# ============================================================
# 步骤1: 检查前置条件
# ============================================================
log_info "步骤1: 检查前置条件..."

# 检查 JAVA_HOME
if [ ! -d "$JAVA_HOME" ]; then
    log_error "JAVA_HOME 不存在: $JAVA_HOME"
    exit 1
fi

# 检查 Doris FE 是否已安装
if [ ! -d "$DORIS_HOME" ]; then
    log_error "Doris 安装目录不存在: $DORIS_HOME"
    log_error "请先在 FE 节点安装 Doris，然后将安装目录分发到本节点"
    exit 1
fi

log_info "前置条件检查通过"

# ============================================================
# 步骤2: 创建 BE 数据目录
# ============================================================
log_info "步骤2: 创建 BE 数据目录..."

run_as_root mkdir -p "$BE_DATA_DIR"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "$BE_DATA_DIR"
log_success "BE 数据目录: $BE_DATA_DIR"

# ============================================================
# 步骤3: 修改 BE 配置文件
# ============================================================
log_info "步骤3: 修改 BE 配置文件..."

# 修改端口
log_info "修改 BE 端口..."
run_as_user sed -i "s#^be_port = .*#be_port = ${BE_PORT}#" "$BE_CONF"
run_as_user sed -i "s#^webserver_port = .*#webserver_port = ${BE_WEBSERVER_PORT}#" "$BE_CONF"
run_as_user sed -i "s#^heartbeat_service_port = .*#heartbeat_service_port = ${BE_HEARTBEAT_PORT}#" "$BE_CONF"
run_as_user sed -i "s#^brpc_port = .*#brpc_port = ${BE_BRPC_PORT}#" "$BE_CONF"

# 添加额外配置
log_info "添加 BE 参数配置..."

# priority_networks
if ! grep -q "priority_networks" "$BE_CONF"; then
    echo "priority_networks = ${PRIORITY_NETWORKS}" >> "$BE_CONF"
    log_info "已添加 priority_networks: ${PRIORITY_NETWORKS}"
fi

# JAVA_HOME
if ! grep -q "^JAVA_HOME" "$BE_CONF"; then
    echo "JAVA_HOME = ${JAVA_HOME}" >> "$BE_CONF"
    log_info "已添加 JAVA_HOME"
fi

# 设置数据目录
if ! grep -q "^storage_root_path" "$BE_CONF"; then
    echo "storage_root_path = ${BE_DATA_DIR}" >> "$BE_CONF"
    log_info "已添加 storage_root_path"
fi

log_success "BE 配置完成"

# ============================================================
# 步骤4: 复制 JDBC 驱动（如果存在）
# ============================================================
log_info "步骤4: 检查 JDBC 驱动..."

SOFTWARE_DIR="/data/softwares"
JDBC_DRIVERS_DIR="${SOFTWARE_DIR}/jdbc_drivers"
if [ -d "$JDBC_DRIVERS_DIR" ]; then
    run_as_user cp -r "$JDBC_DRIVERS_DIR" "${BE_HOME}/"
    log_success "已复制 JDBC 驱动到 BE"
fi

# ============================================================
# 步骤5: 设置目录权限
# ============================================================
log_info "步骤5: 设置目录权限..."

run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "$DORIS_HOME"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "$BE_DATA_DIR"
log_success "权限设置完成"

# ============================================================
# 安装完成
# ============================================================
echo ""
echo "============================================"
echo "Doris BE 安装配置完成！"
echo "============================================"
echo ""
echo "Doris 路径: ${DORIS_HOME}"
echo "BE 数据路径: ${BE_DATA_DIR}"
echo "BE 配置: ${BE_CONF}"
echo ""
echo "端口配置:"
echo "  BE 端口: ${BE_PORT}"
echo "  WebServer 端口: ${BE_WEBSERVER_PORT}"
echo "  Heartbeat 端口: ${BE_HEARTBEAT_PORT}"
echo "  BRPC 端口: ${BE_BRPC_PORT}"
echo ""
echo "后续步骤:"
echo "  1. 确保 FE 已启动"
echo "  2. 启动 BE: bash start_be.sh"
echo "  3. 在 FE 节点添加 BE: bash add_be.sh"
