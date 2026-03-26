#!/bin/bash
# Hadoop 3 配置文件部署脚本
# 使用方法: bash deploy_config.sh
# 参考: tmp/hadoop3/tasks/03_configuration.yml

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

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $1" >&2
}

# ============================================================
# Hadoop 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
CONFIG_DIR="${HADOOP_HOME}/etc/hadoop"
NAMESERVICE="zhugeio"

# ============================================================
# 开始部署
# ============================================================
echo "============================================"
echo "Hadoop 3 配置文件部署脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "配置目录: ${CONFIG_DIR}"
echo "Nameservice: ${NAMESERVICE}"
echo "============================================"

# ============================================================
# 步骤1: 检查安装目录
# ============================================================
log_info "步骤1: 检查安装目录..."
if [ ! -d "${HADOOP_HOME}" ]; then
    log_error "Hadoop未安装，请先执行 install.sh"
    exit 1
fi
log_info "安装目录检查通过"

# ============================================================
# 步骤2: 备份现有配置
# ============================================================
log_info "步骤2: 备份现有配置..."
if [ -d "${CONFIG_DIR}" ]; then
    BACKUP_DIR="${CONFIG_DIR}.bak.$(date '+%Y%m%d%H%M%S')"
    run_as_root cp -r "${CONFIG_DIR}" "${BACKUP_DIR}"
    log_info "配置已备份到: ${BACKUP_DIR}"
fi

# ============================================================
# 步骤3: 部署配置文件
# ============================================================
log_info "步骤3: 部署配置文件..."

# 当前脚本所在目录（配置文件存放位置）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 需要部署的配置文件列表
CONFIG_FILES=(
    "core-site.xml"
    "hdfs-site.xml"
    "yarn-site.xml"
    "mapred-site.xml"
    "hadoop-env.sh"
)

for cfg_file in "${CONFIG_FILES[@]}"; do
    if [ -f "${SCRIPT_DIR}/${cfg_file}" ]; then
        run_as_root cp "${SCRIPT_DIR}/${cfg_file}" "${CONFIG_DIR}/${cfg_file}"
        run_as_root chown ${RUN_USER}:${RUN_GROUP} "${CONFIG_DIR}/${cfg_file}"
        log_info "已部署: ${cfg_file}"
    else
        log_info "配置文件不存在，跳过: ${cfg_file}"
    fi
done

# ============================================================
# 步骤4: 配置 hadoop-env.sh 环境变量
# ============================================================
log_info "步骤4: 配置 hadoop-env.sh..."

HADOOP_ENV_APPEND="
# ====== 自动配置（由部署脚本添加）======
export JAVA_HOME=${JAVA_HOME}
export HADOOP_HOME=${HADOOP_HOME}
export HDFS_NAMENODE_USER=${RUN_USER}
export HDFS_DATANODE_USER=${RUN_USER}
export HDFS_SECONDARYNAMENODE_USER=${RUN_USER}
export YARN_RESOURCEMANAGER_USER=${RUN_USER}
export YARN_NODEMANAGER_USER=${RUN_USER}
export HDFS_JOURNALNODE_USER=${RUN_USER}
export HDFS_ZKFC_USER=${RUN_USER}
"

# 检查是否已配置
if ! grep -q "HDFS_NAMENODE_USER" "${CONFIG_DIR}/hadoop-env.sh" 2>/dev/null; then
    echo "$HADOOP_ENV_APPEND" | run_as_root tee -a "${CONFIG_DIR}/hadoop-env.sh" > /dev/null
    log_info "hadoop-env.sh 环境变量配置完成"
else
    log_info "hadoop-env.sh 已配置，跳过"
fi

run_as_root chown ${RUN_USER}:${RUN_GROUP} "${CONFIG_DIR}/hadoop-env.sh"

# ============================================================
# 步骤5: 创建 workers 文件
# ============================================================
log_info "步骤5: 创建 workers 文件..."

WORKERS_FILE="${CONFIG_DIR}/workers"
echo "dw-worker1" | run_as_root tee -a "${WORKERS_FILE}" > /dev/null
echo "dw-worker2" | run_as_root tee -a "${WORKERS_FILE}" > /dev/null
echo "dw-worker3" | run_as_root tee -a "${WORKERS_FILE}" > /dev/null
run_as_root chown ${RUN_USER}:${RUN_GROUP} "${WORKERS_FILE}"
log_info "workers 文件创建完成"

# 创建 slaves 文件（兼容旧版本）
run_as_root cp "${WORKERS_FILE}" "${CONFIG_DIR}/slaves"
run_as_root chown ${RUN_USER}:${RUN_GROUP} "${CONFIG_DIR}/slaves"
log_info "slaves 文件创建完成"

# ============================================================
# 步骤6: 设置权限
# ============================================================
log_info "步骤6: 设置配置目录权限..."
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${CONFIG_DIR}"
run_as_root chmod -R 755 "${CONFIG_DIR}"
log_info "权限设置完成"

# ============================================================
# 部署完成
# ============================================================
echo ""
echo "============================================"
echo "配置文件部署完成！"
echo "============================================"
echo "配置目录: ${CONFIG_DIR}"
echo ""
echo "已部署的配置文件:"
ls -la "${CONFIG_DIR}"/*.xml "${CONFIG_DIR}"/hadoop-env.sh 2>/dev/null || true
echo ""
echo "后续步骤:"
echo "1. 创建数据目录: bash setup_dirs.sh"
echo "2. 启动服务: bash start_hdfs.sh 或 bash start_yarn.sh"
