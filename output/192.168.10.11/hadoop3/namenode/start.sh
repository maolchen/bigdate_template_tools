#!/bin/bash
# Hadoop NameNode 启动脚本
# 使用方法: bash start.sh [--format] [--format-zk]
#   --format     格式化NameNode（首次启动时使用）
#   --format-zk  格式化ZKFC（首次启动时使用）

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
NAMENODE_DATA_DIR="${DATA_BASE_DIR}/dfs/nn"
NAMESERVICE="zhugeio"
NAMENODE_RPC_PORT="8020"
NAMENODE_HTTP_PORT="50070"
NAMENODE_ID="nn2"

# ============================================================
# 解析参数
# ============================================================
FORMAT_NN=false
FORMAT_ZK=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --format)
            FORMAT_NN=true
            shift
            ;;
        --format-zk)
            FORMAT_ZK=true
            shift
            ;;
        *)
            echo "未知参数: $1"
            echo "使用方法: bash start.sh [--format] [--format-zk]"
            exit 1
            ;;
    esac
done

# ============================================================
# 检查环境
# ============================================================
echo "============================================"
echo "Hadoop NameNode 启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "NameNode ID: ${NAMENODE_ID}"
echo "Nameservice: ${NAMESERVICE}"
echo "HADOOP_HOME: ${HADOOP_HOME}"
echo "数据目录: ${NAMENODE_DATA_DIR}"
echo "RPC端口: ${NAMENODE_RPC_PORT}"
echo "HTTP端口: ${NAMENODE_HTTP_PORT}"
echo "运行用户: ${RUN_USER}"
echo "格式化NameNode: ${FORMAT_NN}"
echo "格式化ZKFC: ${FORMAT_ZK}"
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
# 设置环境变量
# ============================================================
export JAVA_HOME=${JAVA_HOME}
export HADOOP_HOME=${HADOOP_HOME}
export HADOOP_CONF_DIR=${HADOOP_HOME}/etc/hadoop
export PATH=${PATH}:${HADOOP_HOME}/bin:${HADOOP_HOME}/sbin

cd ${HADOOP_HOME}

# ============================================================
# 检查JournalNode状态
# ============================================================
echo ""
echo "检查JournalNode状态..."
JN_PORT="8485"
JN_READY=true
echo "  检查 JournalNode: dw-master1:192.168.10.10:${JN_PORT}"
if timeout 5 bash -c "echo > /dev/tcp/192.168.10.10/${JN_PORT}" 2>/dev/null; then
    echo "    [✓] 连接正常"
else
    echo "    [✗] 无法连接"
    JN_READY=false
fi
echo "  检查 JournalNode: dw-master2:192.168.10.11:${JN_PORT}"
if timeout 5 bash -c "echo > /dev/tcp/192.168.10.11/${JN_PORT}" 2>/dev/null; then
    echo "    [✓] 连接正常"
else
    echo "    [✗] 无法连接"
    JN_READY=false
fi
echo "  检查 JournalNode: dw-master3:192.168.10.12:${JN_PORT}"
if timeout 5 bash -c "echo > /dev/tcp/192.168.10.12/${JN_PORT}" 2>/dev/null; then
    echo "    [✓] 连接正常"
else
    echo "    [✗] 无法连接"
    JN_READY=false
fi

if [ "$JN_READY" = "false" ]; then
    echo ""
    echo "警告: 部分JournalNode不可用"
    echo "请确保所有JournalNode已启动后再继续"
    read -p "是否继续? (y/n): " CONTINUE
    if [ "$CONTINUE" != "y" ]; then
        echo "启动已取消"
        exit 1
    fi
fi

# ============================================================
# 准备数据目录
# ============================================================
echo ""
echo "准备NameNode数据目录..."
run_as_root mkdir -p "${NAMENODE_DATA_DIR}"
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${NAMENODE_DATA_DIR}"

# ============================================================
# 格式化ZKFC（如果需要）
# ============================================================
if [ "$FORMAT_ZK" = "true" ]; then
    echo ""
    echo "格式化ZKFC..."
    echo "注意: 此操作只需要在一个NameNode上执行一次"
    
    if [ "$RUN_USER" = "root" ]; then
        ${HADOOP_HOME}/bin/hdfs zkfc -formatZK
    else
        run_as_root -u ${RUN_USER} ${HADOOP_HOME}/bin/hdfs zkfc -formatZK
    fi
    
    echo "ZKFC格式化完成"
fi

# ============================================================
# 格式化NameNode（如果需要）
# ============================================================
if [ "$FORMAT_NN" = "true" ]; then
    echo ""
    echo "格式化NameNode..."
    echo "警告: 格式化将清空所有HDFS数据！"
    
    # 检查是否已格式化
    if [ -d "${NAMENODE_DATA_DIR}/current" ]; then
        echo "检测到NameNode已格式化"
        read -p "是否重新格式化? 这将清空所有数据! (yes/no): " CONFIRM
        if [ "$CONFIRM" != "yes" ]; then
            echo "跳过格式化"
            FORMAT_NN=false
        fi
    fi
    
    if [ "$FORMAT_NN" = "true" ]; then
        if [ "$RUN_USER" = "root" ]; then
            ${HADOOP_HOME}/bin/hdfs namenode -format -clusterId ${NAMESERVICE}
        else
            run_as_root -u ${RUN_USER} ${HADOOP_HOME}/bin/hdfs namenode -format -clusterId ${NAMESERVICE}
        fi
        echo "NameNode格式化完成"
    fi
fi

# ============================================================
# 检查是否已运行
# ============================================================
if pgrep -f "org.apache.hadoop.hdfs.server.namenode.NameNode" > /dev/null 2>&1; then
    echo ""
    echo "NameNode 已在运行中"
    echo "进程ID: $(pgrep -f 'org.apache.hadoop.hdfs.server.namenode.NameNode')"
    exit 0
fi

# ============================================================
# 启动NameNode
# ============================================================
echo ""
echo "启动NameNode..."

if [ "$RUN_USER" = "root" ]; then
    ${HADOOP_HOME}/bin/hdfs --daemon start namenode
else
    run_as_root -u ${RUN_USER} ${HADOOP_HOME}/bin/hdfs --daemon start namenode
fi

# ============================================================
# 验证启动
# ============================================================
echo ""
echo "等待服务启动..."
sleep 10

if pgrep -f "org.apache.hadoop.hdfs.server.namenode.NameNode" > /dev/null 2>&1; then
    echo "NameNode 启动成功！"
    echo "进程ID: $(pgrep -f 'org.apache.hadoop.hdfs.server.namenode.NameNode')"
    echo "RPC端口: ${NAMENODE_RPC_PORT}"
    echo "HTTP端口: ${NAMENODE_HTTP_PORT}"
    echo ""
    echo "Web UI: http://$(hostname):${NAMENODE_HTTP_PORT}"
    
    # 检查端口
    if command -v ss > /dev/null 2>&1; then
        echo ""
        echo "端口监听状态:"
        ss -tlnp 2>/dev/null | grep -E "${NAMENODE_RPC_PORT}|${NAMENODE_HTTP_PORT}" || echo "端口未监听"
    fi
else
    echo "错误: NameNode 启动失败"
    echo "请检查日志: ${HADOOP_HOME}/logs/hadoop-*-namenode-*.log"
    exit 1
fi

echo ""
echo "============================================"
echo "NameNode 启动完成"
echo "============================================"
echo ""
echo "后续步骤:"
echo "1. 确保ZKFC已启动（如使用HA）"
echo "2. 启动DataNode节点"
echo "3. 检查HDFS状态: hdfs dfsadmin -report"
