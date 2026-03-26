#!/bin/bash
# Hadoop JournalNode 检查脚本
# 使用方法: bash check.sh

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
RUN_USER="bigdata"
JAVA_HOME="/data/jdk/"

# ============================================================
# Hadoop配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
JOURNALNODE_DATA_DIR="${DATA_BASE_DIR}/dfs/jn"
JOURNALNODE_PORT="8485"

# ============================================================
# 检查函数
# ============================================================

check_process() {
    echo "1. 检查进程状态..."
    if pgrep -f "org.apache.hadoop.hdfs.qjournal.server.JournalNode" > /dev/null 2>&1; then
        PID=$(pgrep -f "org.apache.hadoop.hdfs.qjournal.server.JournalNode")
        echo "   [✓] JournalNode 进程运行中 (PID: ${PID})"
        return 0
    else
        echo "   [✗] JournalNode 进程未运行"
        return 1
    fi
}

check_port() {
    echo ""
    echo "2. 检查端口监听..."
    if command -v ss > /dev/null 2>&1; then
        if ss -tln | grep -q ":${JOURNALNODE_PORT} "; then
            echo "   [✓] 端口 ${JOURNALNODE_PORT} 正在监听"
            return 0
        else
            echo "   [✗] 端口 ${JOURNALNODE_PORT} 未监听"
            return 1
        fi
    elif command -v netstat > /dev/null 2>&1; then
        if netstat -tln 2>/dev/null | grep -q ":${JOURNALNODE_PORT} "; then
            echo "   [✓] 端口 ${JOURNALNODE_PORT} 正在监听"
            return 0
        else
            echo "   [✗] 端口 ${JOURNALNODE_PORT} 未监听"
            return 1
        fi
    else
        echo "   [!] 无法检查端口（缺少 ss/netstat 命令）"
        return 2
    fi
}

check_data_dir() {
    echo ""
    echo "3. 检查数据目录..."
    if [ -d "${JOURNALNODE_DATA_DIR}" ]; then
        echo "   [✓] 数据目录存在: ${JOURNALNODE_DATA_DIR}"
        
        # 检查目录内容
        DIR_SIZE=$(du -sh "${JOURNALNODE_DATA_DIR}" 2>/dev/null | awk '{print $1}')
        echo "   [i] 数据目录大小: ${DIR_SIZE}"
        
        # 检查目录权限
        DIR_OWNER=$(ls -ld "${JOURNALNODE_DATA_DIR}" | awk '{print $3":"$4}')
        if [ "$DIR_OWNER" = "${RUN_USER}:${RUN_USER}" ] || [ "$DIR_OWNER" = "${RUN_USER}" ]; then
            echo "   [✓] 目录权限正确: ${DIR_OWNER}"
        else
            echo "   [!] 目录权限: ${DIR_OWNER} (期望: ${RUN_USER}:${RUN_USER})"
        fi
        return 0
    else
        echo "   [✗] 数据目录不存在: ${JOURNALNODE_DATA_DIR}"
        return 1
    fi
}

check_logs() {
    echo ""
    echo "4. 检查日志文件..."
    LOG_DIR="${HADOOP_HOME}/logs"
    if [ -d "${LOG_DIR}" ]; then
        # 查找JournalNode日志
        JN_LOG=$(ls -t ${LOG_DIR}/hadoop-*-journalnode-*.log 2>/dev/null | head -1)
        if [ -n "${JN_LOG}" ]; then
            echo "   [✓] 日志文件: ${JN_LOG}"
            
            # 检查最近的错误
            ERROR_COUNT=$(grep -c "ERROR\|Exception\|FATAL" "${JN_LOG}" 2>/dev/null || echo "0")
            if [ "${ERROR_COUNT}" -gt 0 ]; then
                echo "   [!] 发现 ${ERROR_COUNT} 条错误/异常日志"
                echo "   最近错误:"
                grep -E "ERROR|Exception|FATAL" "${JN_LOG}" 2>/dev/null | tail -3 | while read line; do
                    echo "      ${line}"
                done
            else
                echo "   [✓] 无错误日志"
            fi
        else
            echo "   [!] 未找到JournalNode日志文件"
        fi
    else
        echo "   [✗] 日志目录不存在: ${LOG_DIR}"
    fi
}

check_http() {
    echo ""
    echo "5. 检查HTTP服务..."
    JN_HTTP_PORT=$((JOURNALNODE_PORT - 1))
    
    if command -v curl > /dev/null 2>&1; then
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${JN_HTTP_PORT}/jmx" 2>/dev/null || echo "000")
        if [ "${HTTP_CODE}" = "200" ]; then
            echo "   [✓] HTTP服务正常 (端口: ${JN_HTTP_PORT})"
        else
            echo "   [!] HTTP服务响应: ${HTTP_CODE}"
        fi
    else
        echo "   [!] 无法检查HTTP服务（缺少 curl 命令）"
    fi
}

# ============================================================
# 执行检查
# ============================================================
echo "============================================"
echo "Hadoop JournalNode 健康检查"
echo "============================================"
echo "主机名: $(hostname)"
echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "============================================"

ERRORS=0

check_process || ERRORS=$((ERRORS + 1))
check_port || ERRORS=$((ERRORS + 1))
check_data_dir
check_logs
check_http

echo ""
echo "============================================"
if [ ${ERRORS} -eq 0 ]; then
    echo "检查结果: [✓] 正常"
else
    echo "检查结果: [✗] 发现 ${ERRORS} 个问题"
fi
echo "============================================"

exit ${ERRORS}
