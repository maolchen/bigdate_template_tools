#!/bin/bash
# Hadoop ZKFailoverController 安装脚本
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
# ZKFC配置
# ============================================================
HADOOP_INSTALL_DIR="${INSTALL_BASE_DIR}/hadoop"
NAMESERVICE="zhugeio"

# ============================================================
# 创建目录
# ============================================================
echo "创建ZKFC相关目录..."
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/logs"
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/pids"

# ============================================================
# 在ZooKeeper中创建HA节点（提示信息）
# ============================================================
echo "提示: 需要在ZooKeeper中创建HA节点"
echo "执行命令: zkCli.sh -server <zookeeper_host>:2181"
echo "在ZooKeeper中执行:"
echo "  create /hadoop-ha \"\""
echo "  create /hadoop-ha/${NAMESERVICE} \"\""

echo ""
echo "Hadoop ZKFC 安装准备完成！"
echo "请确保:"
echo "1. Hadoop已安装在: ${HADOOP_INSTALL_DIR}"
echo "2. ZooKeeper集群已启动"
echo "3. 在ZooKeeper中创建 /hadoop-ha/${NAMESERVICE} 节点"
echo "运行用户: ${RUN_USER}"
