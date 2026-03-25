#!/bin/bash
# Hadoop NodeManager 安装脚本
# 使用方法: bash install.sh

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
HADOOP_USER="bigdata"
HADOOP_GROUP="bigdata"

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
mkdir -p "${NODEMANAGER_LOCAL_DIR}"
mkdir -p "${NODEMANAGER_LOG_DIR}"
mkdir -p "${DATA_BASE_DIR}/hadoop/logs"
mkdir -p "${DATA_BASE_DIR}/hadoop/pids"

# ============================================================
# 设置权限
# ============================================================
echo "设置NodeManager目录权限..."
chown -R ${HADOOP_USER}:${HADOOP_GROUP} "${NODEMANAGER_LOCAL_DIR}"
chown -R ${HADOOP_USER}:${HADOOP_GROUP} "${NODEMANAGER_LOG_DIR}"

echo "Hadoop NodeManager 安装准备完成！"
echo "本地目录: ${NODEMANAGER_LOCAL_DIR}"
echo "日志目录: ${NODEMANAGER_LOG_DIR}"
echo "请确保Hadoop已安装在: ${HADOOP_INSTALL_DIR}"
