#!/bin/bash
# Doris BE 启动脚本
# 执行节点: BE 节点
# 用途: 启动 Doris BE 服务

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
INSTALL_SUBDIR="apache-doris/be"
BE_HEARTBEAT_PORT="<no value>"

# ============================================================
# 辅助函数
# ============================================================
run_as_root() {
    if [ "$RUN_USER" = "root" ]; then
        "$@"
    else
        sudo "$@"
    fi
}

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

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $1" >&2
}

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $1"
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SUCCESS] $1"
}

check_port() {
    local port=$1
    if netstat -tuln 2>/dev/null | grep -q ":${port} " || ss -tuln 2>/dev/null | grep -q ":${port} "; then
        return 0
    fi
    return 1
}

# ============================================================
# 开始启动
# ============================================================
echo "============================================"
echo "Doris BE 启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "============================================"

DORIS_HOME="${INSTALL_BASE_DIR}/${INSTALL_SUBDIR}"
BE_HOME="${DORIS_HOME}/be"
BE_BIN="${BE_HOME}/bin/start_be.sh"
BE_PID_FILE="${BE_HOME}/bin/be.pid"

# ============================================================
# 步骤1: 检查前置条件
# ============================================================
log_info "步骤1: 检查前置条件..."

if [ ! -d "$BE_HOME" ]; then
    log_error "BE 目录不存在: $BE_HOME"
    exit 1
fi

if [ ! -f "$BE_BIN" ]; then
    log_error "BE 启动脚本不存在: $BE_BIN"
    exit 1
fi

log_info "前置条件检查通过"

# ============================================================
# 步骤2: 检查是否已运行
# ============================================================
log_info "步骤2: 检查 BE 状态..."

# 检查 PID 文件
if [ -f "$BE_PID_FILE" ]; then
    PID=$(cat "$BE_PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        log_info "BE 已在运行 (PID: $PID)"
        exit 0
    else
        log_warn "清理残留 PID 文件"
        rm -f "$BE_PID_FILE"
    fi
fi

# 检查端口
if check_port "$BE_HEARTBEAT_PORT"; then
    log_warn "BE 端口 ${BE_HEARTBEAT_PORT} 已被占用"
    exit 1
fi

# ============================================================
# 步骤3: 启动 BE
# ============================================================
log_info "步骤3: 启动 BE..."

# 设置环境变量
export JAVA_HOME="/data/jdk/"

log_info "启动命令: ${BE_BIN} --daemon"
run_as_user ${BE_BIN} --daemon

# 等待启动
sleep 5

# ============================================================
# 步骤4: 验证启动
# ============================================================
log_info "步骤4: 验证启动..."

# 检查 PID 文件
if [ -f "$BE_PID_FILE" ]; then
    PID=$(cat "$BE_PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        log_success "BE 启动成功 (PID: $PID)"
    else
        log_error "BE 启动失败，进程未运行"
        exit 1
    fi
else
    # 尝试通过进程名查找
    PID=$(ps -ef | grep "lib/doris_be" | grep -v grep | awk '{print $2}')
    if [ -n "$PID" ]; then
        log_success "BE 启动成功 (PID: $PID)"
    else
        log_error "BE 启动失败，未找到进程"
        exit 1
    fi
fi

# 检查端口
sleep 3
if check_port "$BE_HEARTBEAT_PORT"; then
    log_success "BE 端口 ${BE_HEARTBEAT_PORT} 正在监听"
else
    log_warn "BE 端口 ${BE_HEARTBEAT_PORT} 未监听，请检查日志"
fi

# ============================================================
# 启动完成
# ============================================================
echo ""
echo "============================================"
echo "Doris BE 启动完成！"
echo "============================================"
echo ""
echo "检查状态:"
echo "  jps | grep -v grep | grep -i doris"
echo "  或: ps -ef | grep doris_be"
echo ""
echo "查看日志:"
echo "  tail -f ${BE_HOME}/log/be.log"
echo ""
echo "BE 节点信息:"
echo "  主机名: $(hostname)"
echo "  Heartbeat 端口: ${BE_HEARTBEAT_PORT}"
echo ""
echo "注意: BE 启动后需要在 FE 中执行 add_be.sh 将其加入集群"
