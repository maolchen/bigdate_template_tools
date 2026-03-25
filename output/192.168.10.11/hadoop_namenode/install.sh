#!/bin/bash
# Hadoop NameNode 安装脚本
# 使用方法: bash install.sh

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
SOFTWARE_DIR="/data/softwares"
PKG_PRO_DIR="/data/20251027-yuwq-cdtest"
HADOOP_USER="bigdata"
HADOOP_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

# ============================================================
# Hadoop配置
# ============================================================
HADOOP_VERSION="3.3.6"
HADOOP_INSTALL_DIR="${INSTALL_BASE_DIR}/hadoop"
HADOOP_PKG_NAME="hadoop-3.3.6.tar.gz"
PACKAGE_SUBDIR="realtime-new-doris/realtime/hadoop3"
NAMENODE_DATA_DIR="${DATA_BASE_DIR}/dfs/nn"
TMP_DATA_DIR="${DATA_BASE_DIR}/dfs/tmp"

# ============================================================
# 创建目录
# ============================================================
echo "创建Hadoop目录..."
mkdir -p "${HADOOP_INSTALL_DIR}"
mkdir -p "${NAMENODE_DATA_DIR}"
mkdir -p "${TMP_DATA_DIR}"
mkdir -p "${DATA_BASE_DIR}/hadoop/logs"
mkdir -p "${DATA_BASE_DIR}/hadoop/pids"

# ============================================================
# 解压安装包
# ============================================================
echo "解压Hadoop安装包..."
if [ -f "${PKG_PRO_DIR}/${PACKAGE_SUBDIR}/${HADOOP_PKG_NAME}" ]; then
    tar -zxf "${PKG_PRO_DIR}/${PACKAGE_SUBDIR}/${HADOOP_PKG_NAME}" -C "${HADOOP_INSTALL_DIR}" --strip-components=1
else
    echo "错误: 找不到Hadoop安装包 ${PKG_PRO_DIR}/${PACKAGE_SUBDIR}/${HADOOP_PKG_NAME}"
    exit 1
fi

# ============================================================
# 配置环境变量
# ============================================================
echo "配置Hadoop环境变量..."
HADOOP_HOME="${HADOOP_INSTALL_DIR}"

# 创建环境变量脚本
cat > /etc/profile.d/hadoop.sh <<'EOF'
export HADOOP_HOME=/data/localization/hadoop
export PATH=$PATH:$HADOOP_HOME/bin:$HADOOP_HOME/sbin
export HADOOP_CONF_DIR=$HADOOP_HOME/etc/hadoop
EOF

chmod +x /etc/profile.d/hadoop.sh
source /etc/profile.d/hadoop.sh

# ============================================================
# 设置权限
# ============================================================
echo "设置Hadoop目录权限..."
chown -R ${HADOOP_USER}:${HADOOP_GROUP} "${HADOOP_INSTALL_DIR}"
chown -R ${HADOOP_USER}:${HADOOP_GROUP} "${NAMENODE_DATA_DIR}"
chown -R ${HADOOP_USER}:${HADOOP_GROUP} "${TMP_DATA_DIR}"

echo "Hadoop NameNode 安装完成！"
echo "安装目录: ${HADOOP_INSTALL_DIR}"
echo "数据目录: ${NAMENODE_DATA_DIR}"
