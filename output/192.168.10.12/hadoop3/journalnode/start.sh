#!/bin/bash
# Hadoop JournalNode 启动脚本
# 使用方法: bash start.sh

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

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
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
JOURNALNODE_DATA_DIR="${DATA_BASE_DIR}/dfs/jn"
JOURNALNODE_PORT="8485"

# ============================================================
# 检查环境
# ============================================================
echo "============================================"
echo "Hadoop JournalNode 启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "HADOOP_HOME: ${HADOOP_HOME}"
echo "数据目录: ${JOURNALNODE_DATA_DIR}"
echo "端口: ${JOURNALNODE_PORT}"
echo "运行用户: ${RUN_USER}"
echo "============================================"

# 检查Java环境
if [ ! -d "${JAVA_HOME}" ]; then
    echo "错误: JAVA_HOME 目录不存在: ${JAVA_HOME}"
    exit 1
fi

# 检查Hadoop安装
if [ ! -d "${HADOOP_HOME}" ]; then
    echo "错误: HADOOP_HOME 目录不存在: ${HADOOP_HOME}"
    echo "请先执行 install.sh 完成安装"
    exit 1
fi

# 检查配置文件
if [ ! -f "${HADOOP_HOME}/etc/hadoop/hdfs-site.xml" ]; then
    echo "错误: HDFS配置文件不存在"
    echo "请先部署配置文件到 ${HADOOP_HOME}/etc/hadoop/"
    exit 1
fi

# ============================================================
# 准备数据目录
# ============================================================
echo "准备JournalNode数据目录..."
run_as_root mkdir -p "${JOURNALNODE_DATA_DIR}"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${JOURNALNODE_DATA_DIR}"

# ============================================================
# 启动JournalNode
# ============================================================
echo "启动JournalNode..."

cd ${HADOOP_HOME}

# 设置环境变量
export JAVA_HOME=${JAVA_HOME}
export HADOOP_HOME=${HADOOP_HOME}
export HADOOP_CONF_DIR=${HADOOP_HOME}/etc/hadoop
export PATH=${PATH}:${HADOOP_HOME}/bin:${HADOOP_HOME}/sbin

# 检查是否已运行
if pgrep -f "org.apache.hadoop.hdfs.qjournal.server.JournalNode" > /dev/null 2>&1; then
    echo "JournalNode 已在运行中"
    exit 0
fi

# 启动服务
if [ "$RUN_USER" = "root" ]; then
    ${HADOOP_HOME}/bin/hdfs --daemon start journalnode
else
    run_as_root -u ${RUN_USER} ${HADOOP_HOME}/bin/hdfs --daemon start journalnode
fi

# ============================================================
# 验证启动
# ============================================================
echo "等待服务启动..."
sleep 5

if pgrep -f "org.apache.hadoop.hdfs.qjournal.server.JournalNode" > /dev/null 2>&1; then
    echo "JournalNode 启动成功！"
    echo "进程ID: $(pgrep -f 'org.apache.hadoop.hdfs.qjournal.server.JournalNode')"
    echo "监听端口: ${JOURNALNODE_PORT}"
    
    # 检查端口
    if command -v netstat > /dev/null 2>&1; then
        echo ""
        echo "端口监听状态:"
        netstat -tlnp 2>/dev/null | grep ${JOURNALNODE_PORT} || echo "端口 ${JOURNALNODE_PORT} 未监听"
    fi
else
    echo "错误: JournalNode 启动失败"
    echo "请检查日志: ${HADOOP_HOME}/logs/hadoop-*-journalnode-*.log"
    exit 1
fi

echo ""
echo "============================================"
echo "JournalNode 启动完成"
echo "============================================"
