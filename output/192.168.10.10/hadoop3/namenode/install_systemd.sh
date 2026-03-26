#!/bin/bash
# Hadoop 3 Systemd 服务安装脚本
# 使用方法: bash install_systemd.sh
# 功能: 安装并启用 Hadoop 服务的 systemd 服务文件

set -e

# ============================================================
# 全局变量（遵循 GLOBAL_VARS_GUIDE.md 规范）
# ============================================================
INSTALL_BASE_DIR="/data/localization"
RUN_USER="bigdata"
RUN_GROUP="bigdata"

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

# ============================================================
# 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
SYSTEMD_DIR="/etc/systemd/system"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ============================================================
# 开始安装
# ============================================================
echo "============================================"
echo "Hadoop 3 Systemd 服务安装脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "服务目录: ${SYSTEMD_DIR}"
echo "运行用户: ${RUN_USER}"
echo "============================================"

# ============================================================
# 步骤1: 检查 systemd 目录
# ============================================================
log_info "步骤1: 检查 systemd 目录..."

if [ ! -d "${SYSTEMD_DIR}" ]; then
    log_error "Systemd 目录不存在: ${SYSTEMD_DIR}"
    log_error "此系统可能不支持 systemd"
    exit 1
fi

log_info "Systemd 目录检查通过"

# ============================================================
# 步骤2: 安装服务文件
# ============================================================
log_info "步骤2: 安装服务文件..."

# 定义需要安装的服务（根据当前节点角色）
SERVICES_TO_INSTALL=()

# JournalNode 服务
if [ -f "${SCRIPT_DIR}/hadoop-hdfs-journalnode.service" ]; then
    SERVICES_TO_INSTALL+=("hadoop-hdfs-journalnode")
fi

# ZKFC 服务
if [ -f "${SCRIPT_DIR}/hadoop-hdfs-zkfc.service" ]; then
    SERVICES_TO_INSTALL+=("hadoop-hdfs-zkfc")
fi

# NameNode 服务
if [ -f "${SCRIPT_DIR}/hadoop-hdfs-namenode.service" ]; then
    SERVICES_TO_INSTALL+=("hadoop-hdfs-namenode")
fi

# DataNode 服务
if [ -f "${SCRIPT_DIR}/hadoop-hdfs-datanode.service" ]; then
    SERVICES_TO_INSTALL+=("hadoop-hdfs-datanode")
fi

# ResourceManager 服务
if [ -f "${SCRIPT_DIR}/hadoop-yarn-resourcemanager.service" ]; then
    SERVICES_TO_INSTALL+=("hadoop-yarn-resourcemanager")
fi

# NodeManager 服务
if [ -f "${SCRIPT_DIR}/hadoop-yarn-nodemanager.service" ]; then
    SERVICES_TO_INSTALL+=("hadoop-yarn-nodemanager")
fi

if [ ${#SERVICES_TO_INSTALL[@]} -eq 0 ]; then
    log_warn "未找到任何服务文件"
    exit 0
fi

# 安装服务文件
for service_name in "${SERVICES_TO_INSTALL[@]}"; do
    SERVICE_FILE="${SCRIPT_DIR}/${service_name}.service"
    
    if [ -f "${SERVICE_FILE}" ]; then
        run_as_root cp "${SERVICE_FILE}" "${SYSTEMD_DIR}/${service_name}.service"
        run_as_root chmod 644 "${SYSTEMD_DIR}/${service_name}.service"
        log_info "已安装服务文件: ${service_name}.service"
    fi
done

# ============================================================
# 步骤3: 重载 systemd 配置
# ============================================================
log_info "步骤3: 重载 systemd 配置..."
run_as_root systemctl daemon-reload
log_info "Systemd 配置已重载"

# ============================================================
# 步骤4: 启用服务（开机自启）
# ============================================================
log_info "步骤4: 启用服务..."

for service_name in "${SERVICES_TO_INSTALL[@]}"; do
    if run_as_root systemctl is-enabled ${service_name} > /dev/null 2>&1; then
        log_info "服务已启用: ${service_name}"
    else
        run_as_root systemctl enable ${service_name}
        log_info "已启用服务: ${service_name}"
    fi
done

# ============================================================
# 步骤5: 显示服务状态
# ============================================================
log_info "步骤5: 显示服务状态..."

echo ""
echo "已安装的 Hadoop 服务:"
echo "============================================"
for service_name in "${SERVICES_TO_INSTALL[@]}"; do
    STATUS=$(run_as_root systemctl is-enabled ${service_name} 2>/dev/null || echo "未启用")
    echo "  ${service_name}: ${STATUS}"
done
echo "============================================"

# ============================================================
# 安装完成
# ============================================================
echo ""
echo "============================================"
echo "Systemd 服务安装完成！"
echo "============================================"
echo ""
echo "服务启动顺序说明:"
echo "  1. ZooKeeper (外部依赖)"
echo "  2. hadoop-hdfs-journalnode"
echo "  3. hadoop-hdfs-zkfc"
echo "  4. hadoop-hdfs-namenode"
echo "  5. hadoop-hdfs-datanode"
echo "  6. hadoop-yarn-resourcemanager"
echo "  7. hadoop-yarn-nodemanager"
echo ""
echo "管理命令:"
echo "  启动服务: systemctl start <service_name>"
echo "  停止服务: systemctl stop <service_name>"
echo "  查看状态: systemctl status <service_name>"
echo "  查看日志: journalctl -u <service_name> -f"
