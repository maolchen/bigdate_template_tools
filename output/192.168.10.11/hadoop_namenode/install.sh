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
RUN_USER="bigdata"
RUN_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

# ============================================================
# 用户权限判断（符合规范5）
# ============================================================
if [ "$RUN_USER" = "root" ]; then
    SUDO_CMD=""
    PROFILE_DIR="/etc/profile.d"
else
    SUDO_CMD="sudo"
    PROFILE_DIR="/etc/profile.d"
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
run_as_root mkdir -p "${HADOOP_INSTALL_DIR}"
run_as_root mkdir -p "${NAMENODE_DATA_DIR}"
run_as_root mkdir -p "${TMP_DATA_DIR}"
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/logs"
run_as_root mkdir -p "${DATA_BASE_DIR}/hadoop/pids"

# ============================================================
# 解压安装包
# ============================================================
echo "解压Hadoop安装包..."
if [ -f "${PKG_PRO_DIR}/${PACKAGE_SUBDIR}/${HADOOP_PKG_NAME}" ]; then
    run_as_root tar -zxf "${PKG_PRO_DIR}/${PACKAGE_SUBDIR}/${HADOOP_PKG_NAME}" -C "${HADOOP_INSTALL_DIR}" --strip-components=1
else
    echo "错误: 找不到Hadoop安装包 ${PKG_PRO_DIR}/${PACKAGE_SUBDIR}/${HADOOP_PKG_NAME}"
    exit 1
fi

# ============================================================
# 配置环境变量（符合规范4）
# ============================================================
echo "配置Hadoop环境变量..."
HADOOP_HOME="${HADOOP_INSTALL_DIR}"

# 环境变量内容
ENV_CONTENT="export HADOOP_HOME=/data/localization/hadoop
export PATH=\$PATH:\$HADOOP_HOME/bin:\$HADOOP_HOME/sbin
export HADOOP_CONF_DIR=\$HADOOP_HOME/etc/hadoop"

if [ "$RUN_USER" = "root" ]; then
    # root用户：配置到 /etc/profile.d/
    echo "$ENV_CONTENT" > ${PROFILE_DIR}/hadoop.sh
    chmod +x ${PROFILE_DIR}/hadoop.sh
    source ${PROFILE_DIR}/hadoop.sh
else
    # 非root用户：配置到 ~/.bash_profile 和 /etc/profile
    echo "$ENV_CONTENT" >> ~/.bash_profile
    run_as_root sh -c "echo '$ENV_CONTENT' >> /etc/profile"
    source ~/.bash_profile
fi

# ============================================================
# 设置权限（符合规范6）
# ============================================================
echo "设置Hadoop目录权限..."
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${HADOOP_INSTALL_DIR}"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${NAMENODE_DATA_DIR}"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${TMP_DATA_DIR}"

echo "Hadoop NameNode 安装完成！"
echo "安装目录: ${HADOOP_INSTALL_DIR}"
echo "数据目录: ${NAMENODE_DATA_DIR}"
echo "运行用户: ${RUN_USER}"
