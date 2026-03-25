#!/bin/bash
# Hadoop NodeManager 安装脚本
# 使用方法: bash install.sh

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
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
# NodeManager配置
# ============================================================
HADOOP_INSTALL_DIR="${INSTALL_BASE_DIR}/hadoop"
NODEMANAGER_LOCAL_DIR="${DATA_BASE_DIR}/hadoop3/hadoop-yarn/cache"
NODEMANAGER_LOG_DIR="${DATA_BASE_DIR}/hadoop3/hadoop-yarn/containers"

# ============================================================
# 创建目录
# ============================================================
echo "创建NodeManager数据目录..."
run_as_root mkdir -p "${NODEMANAGER_LOCAL_DIR}"
run_as_root mkdir -p "${NODEMANAGER_LOG_DIR}"
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/logs"
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/pids"

# ============================================================
# 设置权限（符合规范6）
# ============================================================
echo "设置NodeManager目录权限..."
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${NODEMANAGER_LOCAL_DIR}"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${NODEMANAGER_LOG_DIR}"

echo "Hadoop NodeManager 安装准备完成！"
echo "本地目录: ${NODEMANAGER_LOCAL_DIR}"
echo "日志目录: ${NODEMANAGER_LOG_DIR}"
echo "请确保Hadoop已安装在: ${HADOOP_INSTALL_DIR}"
echo "运行用户: ${RUN_USER}"
