#!/bin/bash
# Doris FE 启动脚本
# 执行节点: FE 节点
# 用途: 启动 Doris FE 服务

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
INSTALL_SUBDIR="apache-doris/fe"
FE_EDIT_LOG_PORT="<no value>"

# 获取当前节点信息
CURRENT_IP="$(hostname -I | awk '{print $1}')"
CURRENT_HOSTNAME="$(hostname)"

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
echo "Doris FE 启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "当前 IP: ${CURRENT_IP}"
echo "============================================"

DORIS_HOME="${INSTALL_BASE_DIR}/${INSTALL_SUBDIR}"
FE_HOME="${DORIS_HOME}/fe"
FE_BIN="${FE_HOME}/bin/start_fe.sh"
FE_PID_FILE="${FE_HOME}/bin/fe.pid"

# ============================================================
# 步骤1: 检查前置条件
# ============================================================
log_info "步骤1: 检查前置条件..."

if [ ! -d "$FE_HOME" ]; then
    log_error "FE 目录不存在: $FE_HOME"
    exit 1
fi

if [ ! -f "$FE_BIN" ]; then
    log_error "FE 启动脚本不存在: $FE_BIN"
    exit 1
fi

log_info "前置条件检查通过"

# ============================================================
# 步骤2: 检查是否已运行
# ============================================================
log_info "步骤2: 检查 FE 状态..."

# 检查 PID 文件
if [ -f "$FE_PID_FILE" ]; then
    PID=$(cat "$FE_PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        log_info "FE 已在运行 (PID: $PID)"
        exit 0
    else
        log_warn "清理残留 PID 文件"
        rm -f "$FE_PID_FILE"
    fi
fi

# 检查端口
if check_port "$FE_EDIT_LOG_PORT"; then
    log_warn "FE 端口 ${FE_EDIT_LOG_PORT} 已被占用"
    exit 1
fi

# ============================================================
# 步骤3: 判断 FE 角色
# ============================================================
log_info "步骤3: 判断 FE 启动模式..."

# 获取第一个 FE 节点作为 helper
FIRST_FE_IP="192.168.10.10"
FIRST_FE_HOST="dw-master1"

# 检查当前节点是否为第一个 FE
IS_FIRST_FE=false
if [ "$CURRENT_IP" = "$FIRST_FE_IP" ] || [ "$CURRENT_HOSTNAME" = "$FIRST_FE_HOST" ]; then
    IS_FIRST_FE=true
    log_info "当前节点是第一个 FE (Master)，将以独立模式启动"
fi

# ============================================================
# 步骤4: 启动 FE
# ============================================================
log_info "步骤4: 启动 FE..."

# 设置环境变量
export JAVA_HOME="/data/jdk/"

if [ "$IS_FIRST_FE" = "true" ]; then
    # 第一个 FE 独立启动
    log_info "启动命令: ${FE_BIN} --daemon"
    run_as_user ${FE_BIN} --daemon
else
    # 其他 FE 使用 helper 参数加入集群
    log_info "启动命令: ${FE_BIN} --helper ${FIRST_FE_HOST}:${FE_EDIT_LOG_PORT} --daemon"
    run_as_user ${FE_BIN} --helper ${FIRST_FE_HOST}:${FE_EDIT_LOG_PORT} --daemon
fi

# 等待启动
sleep 5

# ============================================================
# 步骤5: 验证启动
# ============================================================
log_info "步骤5: 验证启动..."

# 检查 PID 文件
if [ -f "$FE_PID_FILE" ]; then
    PID=$(cat "$FE_PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        log_success "FE 启动成功 (PID: $PID)"
    else
        log_error "FE 启动失败，进程未运行"
        exit 1
    fi
else
    # 尝试通过进程名查找
    PID=$(ps -ef | grep "org.apache.doris.DorisFE" | grep -v grep | awk '{print $2}')
    if [ -n "$PID" ]; then
        log_success "FE 启动成功 (PID: $PID)"
    else
        log_error "FE 启动失败，未找到进程"
        exit 1
    fi
fi

# 检查端口
sleep 3
if check_port "$FE_EDIT_LOG_PORT"; then
    log_success "FE 端口 ${FE_EDIT_LOG_PORT} 正在监听"
else
    log_warn "FE 端口 ${FE_EDIT_LOG_PORT} 未监听，请检查日志"
fi

# ============================================================
# 启动完成
# ============================================================
echo ""
echo "============================================"
echo "Doris FE 启动完成！"
echo "============================================"
echo ""
echo "检查状态:"
echo "  jps | grep DorisFE"
echo "  或: ps -ef | grep DorisFE"
echo ""
echo "查看日志:"
echo "  tail -f ${FE_HOME}/log/fe.log"
echo ""
echo "Web UI:"
echo "  http://${CURRENT_IP}:<no value>"
echo ""
if [ "$IS_FIRST_FE" = "true" ]; then
    echo "此 FE 是 Master 节点"
else
    echo "此 FE 已加入集群，使用 helper: ${FIRST_FE_HOST}:${FE_EDIT_LOG_PORT}"
fi
