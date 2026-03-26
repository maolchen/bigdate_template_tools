#!/bin/bash
# Hadoop 3 数据目录设置脚本
# 使用方法: bash setup_dirs.sh
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
# 步骤1: 检查并备份现有目录
# ============================================================
log_info "步骤1: 检查并备份现有目录..."

backup_if_exists() {
    local dir_path="$1"
    local dir_name="$2"
    
    if [ -d "${dir_path}" ]; then
        if [ "$(ls -A ${dir_path} 2>/dev/null)" ]; then
            log_warn "${dir_name} 目录已存在且非空: ${dir_path}"
            BACKUP_DIR="${dir_path}.bak.$(date '+%Y%m%d%H%M%S')"
            run_as_root mv "${dir_path}" "${BACKUP_DIR}"
            log_info "已备份到: ${BACKUP_DIR}"
        else
            log_info "${dir_name} 目录已存在但为空，直接使用"
        fi
    fi
}

# 检查 NameNode 数据目录
backup_if_exists "${NAMENODE_DATA_DIR}" "NameNode"

# 检查 DataNode 数据目录  
backup_if_exists "${DATANODE_DATA_DIR}" "DataNode"

# 检查 JournalNode 数据目录
backup_if_exists "${JOURNALNODE_DATA_DIR}" "JournalNode"

# ============================================================
# 步骤2: 创建 HDFS 目录结构
# ============================================================
log_info "步骤2: 创建 HDFS 目录结构..."

# 定义需要创建的目录
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
    
    run_as_root mkdir -p "${dir_path}"
    run_as_root chmod "${dir_mode}" "${dir_path}"
    run_as_root chown ${RUN_USER}:${RUN_GROUP} "${dir_path}"
    log_info "创建目录: ${dir_path} (权限: ${dir_mode})"
done

# ============================================================
# 步骤3: 创建短路读取 Socket 目录
# ============================================================
log_info "步骤3: 创建短路读取 Socket 目录..."

run_as_root mkdir -p "${SOCKETS_DIR}"
run_as_root chmod 0755 "${SOCKETS_DIR}"
run_as_root chown ${RUN_USER}:${RUN_GROUP} "${SOCKETS_DIR}"
# 移除组写权限（参考ansible）
run_as_root chmod g-w "${SOCKETS_DIR}"
log_info "Socket目录创建完成: ${SOCKETS_DIR}"

# ============================================================
# 步骤4: 创建 YARN 目录结构
# ============================================================
log_info "步骤4: 创建 YARN 目录结构..."

YARN_DIRS=(
    "hadoop/hadoop-yarn:0755"
    "hadoop/hadoop-yarn/cache:0755"
    "hadoop/hadoop-yarn/containers:0755"
)

for dir_info in "${YARN_DIRS[@]}"; do
    dir_path="${DATA_BASE_DIR}/${dir_info%:*}"
    dir_mode="${dir_info#*:}"
    
    run_as_root mkdir -p "${dir_path}"
    run_as_root chmod "${dir_mode}" "${dir_path}"
    run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${dir_path}"
    log_info "创建目录: ${dir_path} (权限: ${dir_mode})"
done

# ============================================================
# 步骤5: 创建日志和 PID 目录
# ============================================================
log_info "步骤5: 创建日志和 PID 目录..."

run_as_root mkdir -p "${HADOOP_HOME}/logs"
run_as_root chmod 755 "${HADOOP_HOME}/logs"
run_as_root chown ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/logs"

run_as_root mkdir -p "${HADOOP_HOME}/pids"
run_as_root chmod 755 "${HADOOP_HOME}/pids"
run_as_root chown ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/pids"

log_info "日志目录: ${HADOOP_HOME}/logs"
log_info "PID目录: ${HADOOP_HOME}/pids"

# ============================================================
# 步骤6: 验证目录结构
# ============================================================
log_info "步骤6: 验证目录结构..."

echo ""
echo "已创建的目录结构:"
echo "============================================"
run_as_root ls -la "${DATA_BASE_DIR}/dfs/" 2>/dev/null || true
run_as_root ls -la "${DATA_BASE_DIR}/hadoop/" 2>/dev/null || true
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
echo "  Socket目录: ${SOCKETS_DIR}"
echo "  YARN目录: ${YARN_DIR}"
echo "  日志目录: ${HADOOP_HOME}/logs"
echo "  PID目录: ${HADOOP_HOME}/pids"
echo ""
echo "后续步骤:"
echo "1. 启动HDFS服务: bash start_hdfs.sh"
echo "2. 启动YARN服务: bash start_yarn.sh"
