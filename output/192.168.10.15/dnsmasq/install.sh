#!/bin/bash
# ============================================================
# Dnsmasq 安装脚本
# 服务: dnsmasq
# 节点: realtime-kafka3 (192.168.10.15)
# ============================================================

set -e

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ==================== 变量定义 ====================
DNSMASQ_CONF_DIR="/etc/dnsmasq.d"
DNSMASQ_CONF_FILE="/etc/dnsmasq.conf"
DNSMASQ_SERVICE="dnsmasq"
LISTEN_ADDRESS="192.168.10.15"

# ==================== 权限辅助函数 ====================
run_as_root() {
    if [ "$(id -u)" -ne 0 ]; then
        sudo "$@"
    else
        "$@"
    fi
}

# ==================== 日志函数 ====================
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# ==================== 检查函数 ====================
check_dnsmasq_installed() {
    if command -v dnsmasq &> /dev/null; then
        return 0
    fi
    return 1
}

check_dnsmasq_service() {
    if systemctl is-active dnsmasq &> /dev/null; then
        return 0
    fi
    return 1
}

# ==================== 安装函数 ====================
install_dnsmasq() {
    log_info "开始安装Dnsmasq..."
    
    # 检查是否已安装
    if check_dnsmasq_installed; then
        log_info "Dnsmasq已安装，跳过安装步骤"
        return 0
    fi
    
    # 尝试使用yum安装
    if command -v yum &> /dev/null; then
        log_info "使用yum安装dnsmasq..."
        run_as_root yum install -y dnsmasq
    elif command -v dnf &> /dev/null; then
        log_info "使用dnf安装dnsmasq..."
        run_as_root dnf install -y dnsmasq
    elif command -v apt-get &> /dev/null; then
        log_info "使用apt-get安装dnsmasq..."
        run_as_root apt-get update
        run_as_root apt-get install -y dnsmasq
    else
        log_error "无法找到可用的包管理器（yum/dnf/apt-get）"
        return 1
    fi
    
    # 验证安装
    if check_dnsmasq_installed; then
        log_success "Dnsmasq安装成功"
        return 0
    else
        log_error "Dnsmasq安装失败"
        return 1
    fi
}

# ==================== 配置函数 ====================
configure_dnsmasq() {
    log_info "配置Dnsmasq..."
    
    # 备份原有配置
    if [ -f "$DNSMASQ_CONF_FILE" ]; then
        log_info "备份原有配置文件..."
        run_as_root cp -f "$DNSMASQ_CONF_FILE" "${DNSMASQ_CONF_FILE}.bak.$(date +%Y%m%d%H%M%S)"
    fi
    
    # 创建配置目录
    if [ ! -d "$DNSMASQ_CONF_DIR" ]; then
        log_info "创建配置目录: $DNSMASQ_CONF_DIR"
        run_as_root mkdir -p "$DNSMASQ_CONF_DIR"
    fi
    
    # 从当前脚本目录复制配置文件
    # 注意：配置文件 dnsmasq.conf 由模板引擎生成，与本脚本在同一目录
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local conf_source="${script_dir}/dnsmasq.conf"
    
    if [ -f "$conf_source" ]; then
        log_info "从 $conf_source 复制配置文件..."
        run_as_root cp -f "$conf_source" "$DNSMASQ_CONF_FILE"
    else
        log_error "配置文件源不存在: $conf_source"
        return 1
    fi
    
    # 验证配置文件
    log_info "验证配置文件..."
    if dnsmasq --test --conf-file="$DNSMASQ_CONF_FILE" 2>/dev/null; then
        log_success "配置文件验证通过"
    else
        log_error "配置文件验证失败，请检查配置"
        return 1
    fi
    
    return 0
}

# ==================== 服务管理函数 ====================
manage_service() {
    log_info "管理Dnsmasq服务..."
    
    # 重载systemd
    run_as_root systemctl daemon-reload
    
    # 停止服务（如果正在运行）
    if check_dnsmasq_service; then
        log_info "停止现有的Dnsmasq服务..."
        run_as_root systemctl stop dnsmasq
    fi
    
    # 启用并启动服务
    log_info "启用并启动Dnsmasq服务..."
    run_as_root systemctl enable dnsmasq
    run_as_root systemctl start dnsmasq
    
    # 等待服务启动
    sleep 2
    
    # 验证服务状态
    if check_dnsmasq_service; then
        log_success "Dnsmasq服务启动成功"
    else
        log_error "Dnsmasq服务启动失败"
        return 1
    fi
    
    return 0
}

# ==================== 验证函数 ====================
verify_installation() {
    log_info "验证Dnsmasq安装..."
    
    # 检查dnsmasq命令
    if ! command -v dnsmasq &> /dev/null; then
        log_error "dnsmasq命令不可用"
        return 1
    fi
    
    # 检查版本
    local version=$(dnsmasq -v 2>&1 | head -1)
    log_info "Dnsmasq版本: $version"
    
    # 检查服务状态
    if systemctl is-active dnsmasq &>/dev/null; then
        log_success "Dnsmasq服务运行正常"
    else
        log_error "Dnsmasq服务未运行"
        return 1
    fi
    
    # 检查监听端口
    if ss -tuln | grep -q ":53 "; then
        log_success "DNS端口(53)正在监听"
    else
        log_warn "DNS端口(53)未监听，可能需要检查配置"
    fi
    
    # 测试DNS解析
    log_info "测试DNS解析..."
    if command -v dig &> /dev/null; then
        if dig @127.0.0.1 localhost +short &>/dev/null; then
            log_success "DNS解析测试通过"
        else
            log_warn "DNS解析测试失败，可能需要进一步配置"
        fi
    else
        log_info "dig命令未安装，跳过DNS解析测试"
    fi
    
    return 0
}

# ==================== 主函数 ====================
main() {
    echo ""
    echo "========================================"
    echo "  Dnsmasq 安装脚本"
    echo "  节点: realtime-kafka3 (192.168.10.15)"
    echo "========================================"
    echo ""
    
    # 执行安装
    if ! install_dnsmasq; then
        log_error "Dnsmasq安装失败！"
        exit 1
    fi
    
    # 执行配置
    if ! configure_dnsmasq; then
        log_error "Dnsmasq配置失败！"
        exit 1
    fi
    
    # 管理服务
    if ! manage_service; then
        log_error "Dnsmasq服务管理失败！"
        exit 1
    fi
    
    # 验证安装
    if verify_installation; then
        log_success "Dnsmasq安装完成！"
        echo ""
        log_info "安装信息:"
        echo "  - 监听地址: ${LISTEN_ADDRESS}"
        echo "  - 配置文件: ${DNSMASQ_CONF_FILE}"
        echo "  - 配置目录: ${DNSMASQ_CONF_DIR}"
        echo "  - 服务状态: $(systemctl is-active dnsmasq 2>/dev/null || echo '未运行')"
        echo ""
        exit 0
    else
        log_error "Dnsmasq安装验证失败！"
        exit 1
    fi
}

# 执行主函数
main "$@"
