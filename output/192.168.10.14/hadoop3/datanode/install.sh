#!/bin/bash
# Hadoop DataNode 安装脚本
# 使用方法: bash install.sh

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
PKG_PRO_DIR="/data/20251027-yuwq-cdtest"
RUN_USER="bigdata"
RUN_GROUP="bigdata"

# ============================================================
# 用户权限判断（符合规范5）
# ============================================================
if [ "$RUN_USER" = "root" ]; then
    SUDO_CMD=""
else
    SUDO_CMD="sudo"
fi

# 辅助函数：以root权限执行命令
run_as_root() {
    if [ "$RUN_USER" = "root" ]; then
        "$@"
    else
        sudo "$@"
    fi
}

# ============================================================
# DataNode配置
# ============================================================
HADOOP_INSTALL_DIR="${INSTALL_BASE_DIR}/hadoop"
DATANODE_DATA_DIR="${DATA_BASE_DIR}/dfs/dn"

# ============================================================
# 创建目录
# ============================================================
echo "创建DataNode数据目录..."
run_as_root mkdir -p "${DATANODE_DATA_DIR}"
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/logs"
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/pids"
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/hdfs-sockets/dn"

# ============================================================
# 设置权限（符合规范6）
# ============================================================
echo "设置DataNode目录权限..."
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${DATANODE_DATA_DIR}"

echo "Hadoop DataNode 安装准备完成！"
echo "数据目录: ${DATANODE_DATA_DIR}"
echo "请确保Hadoop已安装在: ${HADOOP_INSTALL_DIR}"
echo "运行用户: ${RUN_USER}"
