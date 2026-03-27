#!/bin/bash
# Hadoop 3 数据目录设置脚本
# 使用方法: bash setup_dirs.sh
# 说明: 在所有Hadoop节点执行，创建HDFS和YARN数据目录
# 参考: tmp/hadoop3/tasks/04_dir_setup.yml

set -e

# ============================================================
# 全局变量（遵循 GLOBAL_VARS_GUIDE.md 规范）
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
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
# Hadoop 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
NAMENODE_DATA_DIR="${DATA_BASE_DIR}/dfs/nn"
DATANODE_DATA_DIR="${DATA_BASE_DIR}/dfs/dn"
JOURNALNODE_DATA_DIR="${DATA_BASE_DIR}/dfs/jn"
TMP_DATA_DIR="${DATA_BASE_DIR}/dfs/tmp"
SOCKETS_DIR="${DATA_BASE_DIR}/hadoop/hdfs-sockets"
YARN_DIR="${DATA_BASE_DIR}/hadoop/hadoop-yarn"

# ============================================================
# 开始设置
# ============================================================
echo "============================================"
echo "Hadoop 3 数据目录设置脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "数据基础目录: ${DATA_BASE_DIR}"
echo "运行用户: ${RUN_USER}"
echo "============================================"

# ============================================================
# 步骤1: 检查安装目录
# ============================================================
log_info "步骤1: 检查安装目录..."
if [ ! -d "${HADOOP_HOME}" ]; then
    log_error "Hadoop未安装，请先执行 install_binary.sh"
    exit 1
fi
log_info "安装目录检查通过"

# ============================================================
# 步骤2: 备份现有数据目录（如果存在）
# ============================================================
log_info "步骤2: 检查现有数据目录..."

backup_if_exists() {
    local dir_path="$1"
    local dir_name="$2"
    
    if [ -d "${dir_path}" ] && [ "$(ls -A ${dir_path} 2>/dev/null)" ]; then
        log_warn "${dir_name} 目录已存在且非空: ${dir_path}"
        BACKUP_DIR="${dir_path}.bak.$(date '+%Y%m%d%H%M%S')"
        run_as_root mv "${dir_path}" "${BACKUP_DIR}"
        log_info "已备份到: ${BACKUP_DIR}"
    fi
}

# 检查各数据目录
backup_if_exists "${NAMENODE_DATA_DIR}" "NameNode"
backup_if_exists "${DATANODE_DATA_DIR}" "DataNode"
backup_if_exists "${JOURNALNODE_DATA_DIR}" "JournalNode"

# ============================================================
# 步骤3: 创建 HDFS 目录结构
# ============================================================
log_info "步骤3: 创建 HDFS 目录结构..."

# 定义需要创建的目录（路径:权限）
HDFS_DIRS=(
    "dfs:0750"
    "dfs/jn:0750"
    "dfs/nn:0700"
    "dfs/dn:0750"
    "dfs/tmp:0750"
)

for dir_info in "${HDFS_DIRS[@]}"; do
    dir_path="${DATA_BASE_DIR}/${dir_info%:*}"
    dir_mode="${dir_info#*:}"
    
    if [ ! -d "${dir_path}" ]; then
        run_as_root mkdir -p "${dir_path}"
        run_as_root chmod "${dir_mode}" "${dir_path}"
        run_as_root chown ${RUN_USER}:${RUN_GROUP} "${dir_path}"
        log_info "创建目录: ${dir_path} (权限: ${dir_mode})"
    else
        log_info "目录已存在: ${dir_path}"
        run_as_root chown ${RUN_USER}:${RUN_GROUP} "${dir_path}"
    fi
done

# ============================================================
# 步骤4: 创建短路读取 Socket 目录
# ============================================================
log_info "步骤4: 创建短路读取 Socket 目录..."

if [ ! -d "${SOCKETS_DIR}/dn" ]; then
    run_as_root mkdir -p "${SOCKETS_DIR}/dn"
fi
run_as_root chmod 755 "${SOCKETS_DIR}/dn"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${SOCKETS_DIR}"
run_as_root chmod g-w "${SOCKETS_DIR}/dn"
log_info "Socket目录: ${SOCKETS_DIR}/dn"

# ============================================================
# 步骤5: 创建 YARN 目录结构
# ============================================================
log_info "步骤5: 创建 YARN 目录结构..."

YARN_DIRS=(
    "hadoop/hadoop-yarn:0755"
    "hadoop/hadoop-yarn/cache:0755"
    "hadoop/hadoop-yarn/containers:0755"
)

for dir_info in "${YARN_DIRS[@]}"; do
    dir_path="${DATA_BASE_DIR}/${dir_info%:*}"
    dir_mode="${dir_info#*:}"
    
    if [ ! -d "${dir_path}" ]; then
        run_as_root mkdir -p "${dir_path}"
        run_as_root chmod "${dir_mode}" "${dir_path}"
        run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${dir_path}"
        log_info "创建目录: ${dir_path} (权限: ${dir_mode})"
    else
        log_info "目录已存在: ${dir_path}"
        run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${dir_path}"
    fi
done

# ============================================================
# 步骤6: 创建日志和 PID 目录
# ============================================================
log_info "步骤6: 创建日志和 PID 目录..."

run_as_root mkdir -p "${HADOOP_HOME}/logs"
run_as_root chmod 755 "${HADOOP_HOME}/logs"
run_as_root chown ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/logs"

run_as_root mkdir -p "${HADOOP_HOME}/pids"
run_as_root chmod 755 "${HADOOP_HOME}/pids"
run_as_root chown ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/pids"

log_info "日志目录: ${HADOOP_HOME}/logs"
log_info "PID目录: ${HADOOP_HOME}/pids"

# ============================================================
# 步骤7: 验证目录结构
# ============================================================
log_info "步骤7: 验证目录结构..."

echo ""
echo "已创建的目录结构:"
echo "============================================"
echo "HDFS目录:"
ls -la "${DATA_BASE_DIR}/dfs/" 2>/dev/null || echo "  未创建"
echo ""
echo "Hadoop目录:"
ls -la "${DATA_BASE_DIR}/hadoop/" 2>/dev/null || echo "  未创建"
echo "============================================"

# ============================================================
# 设置完成
# ============================================================
echo ""
echo "============================================"
echo "数据目录设置完成！"
echo "============================================"
echo ""
echo "目录结构:"
echo "  NameNode数据: ${NAMENODE_DATA_DIR}"
echo "  DataNode数据: ${DATANODE_DATA_DIR}"
echo "  JournalNode数据: ${JOURNALNODE_DATA_DIR}"
echo "  Socket目录: ${SOCKETS_DIR}/dn"
echo "  YARN目录: ${YARN_DIR}"
echo "  日志目录: ${HADOOP_HOME}/logs"
echo "  PID目录: ${HADOOP_HOME}/pids"
echo ""
echo "后续步骤:"
echo "1. 启动 JournalNode: bash start_journalnode.sh"
echo "2. 初始化主 NameNode: bash init_namenode_master.sh"
echo "3. 初始化备 NameNode: bash init_namenode_standby.sh"
