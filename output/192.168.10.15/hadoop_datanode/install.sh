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
HADOOP_USER="bigdata"
HADOOP_GROUP="bigdata"

# ============================================================
# DataNode配置
# ============================================================
HADOOP_INSTALL_DIR="${INSTALL_BASE_DIR}/hadoop"
DATANODE_DATA_DIR="${DATA_BASE_DIR}/dfs/dn"

# ============================================================
# 创建目录
# ============================================================
echo "创建DataNode数据目录..."
mkdir -p "${DATANODE_DATA_DIR}"
mkdir -p "${DATA_BASE_DIR}/hadoop/logs"
mkdir -p "${DATA_BASE_DIR}/hadoop/pids"
mkdir -p "${DATA_BASE_DIR}/hadoop/hdfs-sockets/dn"

# ============================================================
# 设置权限
# ============================================================
echo "设置DataNode目录权限..."
chown -R ${HADOOP_USER}:${HADOOP_GROUP} "${DATANODE_DATA_DIR}"

echo "Hadoop DataNode 安装准备完成！"
echo "数据目录: ${DATANODE_DATA_DIR}"
echo "请确保Hadoop已安装在: ${HADOOP_INSTALL_DIR}"
