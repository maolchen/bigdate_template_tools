#!/bin/bash
# ============================================================
# NTP Master 安装脚本
# 服务: ntp_master
# 节点: dw-master1 (192.168.10.10)
# ============================================================

set -e

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ==================== 变量定义 ====================
NTP_CONF_FILE="/etc/ntp.conf"
NTP_SERVICE="ntpd"
STRATUM="10"

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
check_ntp_installed() {
    if command -v ntpd &> /dev/null; then
        return 0
    fi
    return 1
}

check_ntp_service() {
    if systemctl is-active ntpd &> /dev/null; then
        return 0
    fi
    return 1
}

# ==================== 安装函数 ====================
install_ntp() {
    log_info "开始安装NTP Master..."
    
    # 检查是否已安装
    if check_ntp_installed; then
        log_info "NTP已安装，跳过安装步骤"
        return 0
    fi
    
    # 尝试使用yum安装
    if command -v yum &> /dev/null; then
        log_info "使用yum安装ntp..."
        run_as_root yum install -y ntp
    elif command -v dnf &> /dev/null; then
        log_info "使用dnf安装ntp..."
        run_as_root dnf install -y ntp
    elif command -v apt-get &> /dev/null; then
        log_info "使用apt-get安装ntp..."
        run_as_root apt-get update
        run_as_root apt-get install -y ntp
    else
        log_error "无法找到可用的包管理器（yum/dnf/apt-get）"
        return 1
    fi
    
    # 验证安装
    if check_ntp_installed; then
        log_success "NTP安装成功"
        return 0
    else
        log_error "NTP安装失败"
        return 1
    fi
}

# ==================== 配置函数 ====================
configure_ntp() {
    log_info "配置NTP Master..."
    
    # 备份原有配置
    if [ -f "$NTP_CONF_FILE" ]; then
        log_info "备份原有配置文件..."
        run_as_root cp -f "$NTP_CONF_FILE" "${NTP_CONF_FILE}.bak.$(date +%Y%m%d%H%M%S)"
    fi
    
    # 从当前脚本目录复制配置文件
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local conf_source="${script_dir}/ntp.conf"
    
    if [ -f "$conf_source" ]; then
        log_info "从 $conf_source 复制配置文件..."
        run_as_root cp -f "$conf_source" "$NTP_CONF_FILE"
    else
        log_error "配置文件源不存在: $conf_source"
        return 1
    fi
    
    # 创建日志和统计目录
    run_as_root mkdir -p /var/log/ntpstats
    run_as_root mkdir -p /var/lib/ntp
    
    # 设置权限
    run_as_root chown -R ntp:ntp /var/log/ntpstats 2>/dev/null || true
    run_as_root chown -R ntp:ntp /var/lib/ntp 2>/dev/null || true
    
    log_success "NTP Master配置完成"
    return 0
}

# ==================== 服务管理函数 ====================
manage_service() {
    log_info "管理NTP服务..."
    
    # 停止chrony服务（如果存在，会与ntp冲突）
    if systemctl is-active chronyd &> /dev/null; then
        log_info "停止chrony服务（与ntp冲突）..."
        run_as_root systemctl stop chronyd
        run_as_root systemctl disable chronyd
    fi
    
    # 重载systemd
    run_as_root systemctl daemon-reload
    
    # 停止服务（如果正在运行）
    if check_ntp_service; then
        log_info "停止现有的NTP服务..."
        run_as_root systemctl stop ntpd
    fi
    
    # 启用并启动服务
    log_info "启用并启动NTP服务..."
    run_as_root systemctl enable ntpd
    run_as_root systemctl start ntpd
    
    # 等待服务启动
    sleep 3
    
    # 验证服务状态
    if check_ntp_service; then
        log_success "NTP服务启动成功"
    else
        log_error "NTP服务启动失败"
        return 1
    fi
    
    return 0
}

# ==================== 验证函数 ====================
verify_installation() {
    log_info "验证NTP Master安装..."
    
    # 检查ntpd命令
    if ! command -v ntpd &> /dev/null; then
        log_error "ntpd命令不可用"
        return 1
    fi
    
    # 检查版本
    local version=$(ntpd --version 2>&1 | head -1)
    log_info "NTP版本: $version"
    
    # 检查服务状态
    if systemctl is-active ntpd &>/dev/null; then
        log_success "NTP服务运行正常"
    else
        log_error "NTP服务未运行"
        return 1
    fi
    
    # 检查监听端口
    if ss -tuln | grep -q ":123 "; then
        log_success "NTP端口(123)正在监听"
    else
        log_warn "NTP端口(123)未监听，可能需要检查配置"
    fi
    
    # 检查NTP同步状态
    log_info "检查NTP同步状态..."
    sleep 2
    if ntpq -p &>/dev/null; then
        log_info "NTP同步状态:"
        ntpq -p
    fi
    
    return 0
}

# ==================== 主函数 ====================
main() {
    echo ""
    echo "========================================"
    echo "  NTP Master 安装脚本"
    echo "  节点: dw-master1 (192.168.10.10)"
    echo "  Stratum: ${STRATUM}"
    echo "========================================"
    echo ""
    
    # 执行安装
    if ! install_ntp; then
        log_error "NTP安装失败！"
        exit 1
    fi
    
    # 执行配置
    if ! configure_ntp; then
        log_error "NTP配置失败！"
        exit 1
    fi
    
    # 管理服务
    if ! manage_service; then
        log_error "NTP服务管理失败！"
        exit 1
    fi
    
    # 验证安装
    if verify_installation; then
        log_success "NTP Master安装完成！"
        echo ""
        log_info "安装信息:"
        echo "  - 节点角色: NTP Master"
        echo "  - 本地时钟: 127.127.1.0"
        echo "  - Stratum: ${STRATUM}"
        echo "  - 配置文件: ${NTP_CONF_FILE}"
        echo "  - 服务状态: $(systemctl is-active ntpd 2>/dev/null || echo '未运行')"
        echo ""
        exit 0
    else
        log_error "NTP安装验证失败！"
        exit 1
    fi
}

# 执行主函数
main "$@"
