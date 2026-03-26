#!/bin/bash
# Hadoop 3 ZKFC 准备脚本
# 使用方法: bash install.sh
# 说明: Hadoop是统一安装的，此脚本仅做环境检查

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

# ============================================================
# 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"

# ============================================================
# 开始准备
# ============================================================
echo "============================================"
echo "Hadoop 3 ZKFC 准备脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "HADOOP_HOME: ${HADOOP_HOME}"
echo "运行用户: ${RUN_USER}"
echo "============================================"

# ============================================================
# 步骤1: 检查Hadoop安装
# ============================================================
log_info "步骤1: 检查Hadoop安装..."

if [ ! -d "${HADOOP_HOME}" ]; then
    log_error "Hadoop未安装: ${HADOOP_HOME}"
    log_error "请先在NameNode节点执行 install.sh 完成Hadoop安装"
    exit 1
fi

if [ ! -f "${HADOOP_HOME}/bin/hdfs" ]; then
    log_error "Hadoop二进制文件不存在"
    exit 1
fi

log_info "Hadoop安装检查通过"

# ============================================================
# 步骤2: 创建日志和PID目录
# ============================================================
log_info "步骤2: 创建日志和PID目录..."

run_as_root mkdir -p "${HADOOP_HOME}/logs"
run_as_root mkdir -p "${HADOOP_HOME}/pids"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/logs"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${HADOOP_HOME}/pids"
log_info "日志目录: ${HADOOP_HOME}/logs"
log_info "PID目录: ${HADOOP_HOME}/pids"

# ============================================================
# 准备完成
# ============================================================
echo ""
echo "============================================"
echo "ZKFC 准备完成！"
echo "============================================"
echo ""
echo "后续步骤:"
echo "1. 确保ZooKeeper集群已启动"
echo "2. 启动服务: bash start.sh [--format]"
