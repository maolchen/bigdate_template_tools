#!/bin/bash
# Hadoop 3 服务停止脚本
# 使用方法: bash stop.sh [服务名]
#   服务名: all, hdfs, yarn, namenode, datanode, resourcemanager, nodemanager, journalnode, zkfc
# 参考: tmp/hadoop3/tasks/05_service_startup.yml

set -e

# ============================================================
# 全局变量（遵循 GLOBAL_VARS_GUIDE.md 规范）
# ============================================================
INSTALL_BASE_DIR="/data/localization"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

# ============================================================
# 辅助函数
# ============================================================
run_as_user() {
    if [ "$RUN_USER" = "root" ]; then
        "$@"
    else
        sudo -u ${RUN_USER} "$@"
    fi
}

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $1"
}

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $1"
}

stop_service() {
    local service_name="$1"
    local process_pattern="$2"
    local stop_cmd="$3"
    
    if pgrep -f "${process_pattern}" > /dev/null 2>&1; then
        log_info "停止 ${service_name}..."
        run_as_user ${stop_cmd}
        sleep 3
        
        if pgrep -f "${process_pattern}" > /dev/null 2>&1; then
            log_warn "${service_name} 停止超时，尝试强制终止"
            pkill -9 -f "${process_pattern}" 2>/dev/null || true
            sleep 2
        fi
        
        if pgrep -f "${process_pattern}" > /dev/null 2>&1; then
            log_warn "${service_name} 停止失败"
            return 1
        else
            log_info "${service_name} 已停止"
        fi
    else
        log_info "${service_name} 未运行"
    fi
    return 0
}

# ============================================================
# Hadoop 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"

# 设置环境变量
export JAVA_HOME=${JAVA_HOME}
export HADOOP_HOME=${HADOOP_HOME}
export HADOOP_CONF_DIR=${HADOOP_HOME}/etc/hadoop
export PATH=${PATH}:${HADOOP_HOME}/bin:${HADOOP_HOME}/sbin

# ============================================================
# 参数解析
# ============================================================
SERVICE="${1:-all}"

# ============================================================
# 开始停止
# ============================================================
echo "============================================"
echo "Hadoop 3 服务停止脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "停止范围: ${SERVICE}"
echo "HADOOP_HOME: ${HADOOP_HOME}"
echo "============================================"

cd ${HADOOP_HOME}

case "$SERVICE" in
    all)
        log_info "停止所有服务..."
        
        # 停止顺序（与启动顺序相反）
        # 1. NodeManager
        stop_service "NodeManager" \
            "org.apache.hadoop.yarn.server.nodemanager.NodeManager" \
            "${HADOOP_HOME}/bin/yarn --daemon stop nodemanager"
        
        # 2. ResourceManager
        stop_service "ResourceManager" \
            "org.apache.hadoop.yarn.server.resourcemanager.ResourceManager" \
            "${HADOOP_HOME}/bin/yarn --daemon stop resourcemanager"
        
        # 3. DataNode
        stop_service "DataNode" \
            "org.apache.hadoop.hdfs.server.datanode.DataNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop datanode"
        
        # 4. NameNode
        stop_service "NameNode" \
            "org.apache.hadoop.hdfs.server.namenode.NameNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop namenode"
        
        # 5. ZKFC
        stop_service "ZKFC" \
            "org.apache.hadoop.hdfs.tools.DFSZKFailoverController" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop zkfc"
        
        # 6. JournalNode
        stop_service "JournalNode" \
            "org.apache.hadoop.hdfs.qjournal.server.JournalNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop journalnode"
        ;;
    
    hdfs)
        log_info "停止 HDFS 服务..."
        
        stop_service "DataNode" \
            "org.apache.hadoop.hdfs.server.datanode.DataNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop datanode"
        
        stop_service "NameNode" \
            "org.apache.hadoop.hdfs.server.namenode.NameNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop namenode"
        
        stop_service "ZKFC" \
            "org.apache.hadoop.hdfs.tools.DFSZKFailoverController" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop zkfc"
        
        stop_service "JournalNode" \
            "org.apache.hadoop.hdfs.qjournal.server.JournalNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop journalnode"
        ;;
    
    yarn)
        log_info "停止 YARN 服务..."
        
        stop_service "NodeManager" \
            "org.apache.hadoop.yarn.server.nodemanager.NodeManager" \
            "${HADOOP_HOME}/bin/yarn --daemon stop nodemanager"
        
        stop_service "ResourceManager" \
            "org.apache.hadoop.yarn.server.resourcemanager.ResourceManager" \
            "${HADOOP_HOME}/bin/yarn --daemon stop resourcemanager"
        ;;
    
    namenode)
        stop_service "NameNode" \
            "org.apache.hadoop.hdfs.server.namenode.NameNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop namenode"
        ;;
    
    datanode)
        stop_service "DataNode" \
            "org.apache.hadoop.hdfs.server.datanode.DataNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop datanode"
        ;;
    
    resourcemanager)
        stop_service "ResourceManager" \
            "org.apache.hadoop.yarn.server.resourcemanager.ResourceManager" \
            "${HADOOP_HOME}/bin/yarn --daemon stop resourcemanager"
        ;;
    
    nodemanager)
        stop_service "NodeManager" \
            "org.apache.hadoop.yarn.server.nodemanager.NodeManager" \
            "${HADOOP_HOME}/bin/yarn --daemon stop nodemanager"
        ;;
    
    journalnode)
        stop_service "JournalNode" \
            "org.apache.hadoop.hdfs.qjournal.server.JournalNode" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop journalnode"
        ;;
    
    zkfc)
        stop_service "ZKFC" \
            "org.apache.hadoop.hdfs.tools.DFSZKFailoverController" \
            "${HADOOP_HOME}/bin/hdfs --daemon stop zkfc"
        ;;
    
    *)
        echo "未知服务: ${SERVICE}"
        echo "使用方法: bash stop.sh [all|hdfs|yarn|namenode|datanode|resourcemanager|nodemanager|journalnode|zkfc]"
        exit 1
        ;;
esac

# ============================================================
# 停止完成
# ============================================================
echo ""
echo "============================================"
echo "服务停止完成！"
echo "============================================"
echo ""
echo "剩余进程:"
jps 2>/dev/null | grep -E "NameNode|DataNode|JournalNode|DFSZKFailoverController|ResourceManager|NodeManager" || echo "  无 Hadoop 进程"
