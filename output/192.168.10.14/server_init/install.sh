#!/bin/bash
# ============================================================
# 服务器初始化脚本
# 节点: dw-worker2
# IP: 192.168.10.14
# Hostname: dw-worker2
# 兼容系统: CentOS 7/8/9, 麒麟, 统信UOS等RedHat系列
# ============================================================

set -o pipefail

# ============================================================
# 全局变量
# ============================================================
SCRIPT_NAME="server_init"
LOG_FILE="/var/log/server_init.log"
MARKER_FILE="/etc/.server_init_done"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ============================================================
# 权限辅助函数
# ============================================================
run_as_root() {
    if [ "$(id -u)" -ne 0 ]; then
        sudo "$@"
    else
        "$@"
    fi
}

# ============================================================
# 工具函数
# ============================================================

log() {
    local level=$1
    shift
    local msg="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${msg}" | run_as_root tee -a "$LOG_FILE"
}

log_info() {
    log "INFO" "${GREEN}[成功]${NC} $@"
}

log_warn() {
    log "WARN" "${YELLOW}[警告]${NC} $@"
}

log_error() {
    log "ERROR" "${RED}[失败]${NC} $@"
}

# 检测操作系统类型
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_ID="${ID}"
        OS_VERSION_ID="${VERSION_ID}"
        OS_PRETTY_NAME="${PRETTY_NAME}"
    elif [ -f /etc/redhat-release ]; then
        OS_ID="centos"
        OS_PRETTY_NAME=$(cat /etc/redhat-release)
    else
        OS_ID="unknown"
        OS_PRETTY_NAME="Unknown Linux"
    fi
    log_info "检测到系统: ${OS_PRETTY_NAME}"
}

# 获取系统服务管理器
get_service_manager() {
    if command -v systemctl >/dev/null 2>&1; then
        echo "systemd"
    elif command -v service >/dev/null 2>&1; then
        echo "sysvinit"
    else
        echo "unknown"
    fi
}

# 检查命令是否存在
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# ============================================================
# 1. 关闭防火墙
# ============================================================
disable_firewall() {
    log_info "========== 开始关闭防火墙 =========="
    
    # 处理 firewalld
    if command_exists firewall-cmd; then
        if systemctl is-active firewalld >/dev/null 2>&1; then
            run_as_root systemctl stop firewalld && log_info "停止 firewalld 服务" || log_error "停止 firewalld 服务失败"
        else
            log_info "firewalld 服务已停止"
        fi
        if systemctl is-enabled firewalld >/dev/null 2>&1; then
            run_as_root systemctl disable firewalld && log_info "禁用 firewalld 开机自启" || log_error "禁用 firewalld 开机自启失败"
        else
            log_info "firewalld 已禁用开机自启"
        fi
    else
        log_info "未安装 firewalld，跳过"
    fi
    
    # 处理 iptables（CentOS 6/7）
    if command_exists iptables; then
        if systemctl is-active iptables >/dev/null 2>&1 2>/dev/null; then
            run_as_root systemctl stop iptables 2>/dev/null && log_info "停止 iptables 服务" || true
            run_as_root systemctl disable iptables 2>/dev/null && log_info "禁用 iptables 开机自启" || true
        fi
        if service iptables status >/dev/null 2>&1; then
            run_as_root service iptables stop && log_info "停止 iptables 服务" || true
            run_as_root chkconfig iptables off 2>/dev/null && log_info "禁用 iptables 开机自启" || true
        fi
    fi
    
    log_info "========== 防火墙配置完成 =========="
}

# ============================================================
# 2. 关闭SELinux
# ============================================================
disable_selinux() {
    log_info "========== 开始关闭SELinux =========="
    
    if ! command_exists getenforce; then
        log_info "系统未安装SELinux，跳过"
        return 0
    fi
    
    local current_status=$(getenforce 2>/dev/null)
    
    if [ "$current_status" = "Disabled" ]; then
        log_info "SELinux 已处于 Disabled 状态"
    else
        # 临时关闭
        run_as_root setenforce 0 2>/dev/null && log_info "临时关闭 SELinux" || log_warn "临时关闭 SELinux 失败（可能需要重启）"
        
        # 永久关闭（修改配置文件）
        if [ -f /etc/selinux/config ]; then
            if grep -q "^SELINUX=disabled" /etc/selinux/config; then
                log_info "SELinux 配置文件已设置为 disabled"
            elif grep -q "^SELINUX=" /etc/selinux/config; then
                run_as_root sed -i 's/^SELINUX=.*/SELINUX=disabled/g' /etc/selinux/config
                log_info "修改 SELinux 配置文件为 disabled"
            else
                run_as_root sed -i '/^SELINUX=/d' /etc/selinux/config
                run_as_root bash -c "echo 'SELINUX=disabled' >> /etc/selinux/config"
                log_info "添加 SELinux 配置: SELINUX=disabled"
            fi
        fi
    fi
    
    log_info "========== SELinux配置完成 =========="
}

# ============================================================
# 3. 关闭Swap
# ============================================================
disable_swap() {
    log_info "========== 开始关闭Swap =========="
    
    # 临时关闭
    local swap_count=$(swapon -s | wc -l)
    if [ "$swap_count" -gt 0 ]; then
        run_as_root swapoff -a && log_info "临时关闭所有 swap 分区" || log_error "关闭 swap 失败"
    else
        log_info "Swap 已关闭"
    fi
    
    # 永久关闭（注释掉 /etc/fstab 中的 swap 行）
    if [ -f /etc/fstab ]; then
        if grep -q "^[^#].*swap" /etc/fstab; then
            run_as_root sed -i 's/^\([^#].*swap.*\)/#\1/g' /etc/fstab
            log_info "注释 /etc/fstab 中的 swap 配置"
        else
            log_info "/etc/fstab 中无 swap 配置，跳过"
        fi
    fi
    
    log_info "========== Swap配置完成 =========="
}

# ============================================================
# 4. 加载内核模块并设置开机自动加载
# ============================================================
load_kernel_modules() {
    log_info "========== 开始加载内核模块 =========="
    local modules="ip_vs ip_vs_rr ip_vs_wrr ip_vs_sh nf_conntrack br_netfilter "
    
    for module in $modules; do
        # 检查模块是否已加载
        if lsmod | grep -q "^${module}" 2>/dev/null; then
            log_info "内核模块已加载: $module"
        else
            run_as_root modprobe $module && log_info "加载内核模块: $module" || log_warn "加载内核模块失败: $module（可能需要安装额外包）"
        fi
        
        # 设置开机自动加载
        local conf_file="/etc/modules-load.d/${module}.conf"
        if [ -f "$conf_file" ] && grep -q "^${module}$" "$conf_file"; then
            log_info "模块开机自启已配置: $module"
        else
            run_as_root mkdir -p /etc/modules-load.d
            run_as_root bash -c "echo '$module' > '$conf_file'"
            log_info "配置模块开机自启: $module"
        fi
    done
    
    # CentOS 7 需要安装 ipvsadm 才能加载 ip_vs 相关模块
    if ! lsmod | grep -q "ip_vs" 2>/dev/null; then
        log_warn "ip_vs 模块未加载，可能需要安装 ipvsadm 包"
    fi
    
    log_info "========== 内核模块配置完成 =========="
}

# ============================================================
# 5. 配置sysctl内核参数
# ============================================================
configure_sysctl() {
    log_info "========== 开始配置sysctl参数 =========="
    
    local sysctl_file="/etc/sysctl.d/99-bigdata.conf"
    
    # 需要先加载 br_netfilter 模块，否则 bridge-nf-call 参数可能不生效
    run_as_root modprobe br_netfilter 2>/dev/null || true
    
    # 直接覆盖文件内容（独立配置文件，不影响系统默认配置）
    run_as_root bash -c "cat > '$sysctl_file' << 'SYSCTL_EOF'
# BigData Platform Kernel Parameters
# Generated by server_init script
SYSCTL_EOF"
    run_as_root bash -c "echo 'fs.file-max=6553560' >> '$sysctl_file'"
    run_as_root bash -c "echo 'net.core.somaxconn=65536' >> '$sysctl_file'"
    run_as_root bash -c "echo 'net.ipv4.ip_forward=1' >> '$sysctl_file'"
    run_as_root bash -c "echo 'net.ipv4.tcp_max_syn_backlog=8192' >> '$sysctl_file'"
    run_as_root bash -c "echo 'vm.dirty_background_ratio=5' >> '$sysctl_file'"
    run_as_root bash -c "echo 'vm.max_map_count=2000000' >> '$sysctl_file'"
    run_as_root bash -c "echo 'vm.panic_on_oom=0' >> '$sysctl_file'"
    run_as_root bash -c "echo 'vm.swappiness=0' >> '$sysctl_file'"
    
    log_info "写入 sysctl 配置文件: $sysctl_file"
    
    # 应用配置
    run_as_root sysctl -p "$sysctl_file" >/dev/null 2>&1 && log_info "应用 sysctl 配置成功" || log_warn "部分 sysctl 配置应用失败"
    
    log_info "========== sysctl配置完成 =========="
}

# ============================================================
# 6. 配置limits文件描述符限制
# ============================================================
configure_limits() {
    log_info "========== 开始配置文件描述符限制 =========="
    
    local limits_file="/etc/security/limits.conf"
    local limits_d_file="/etc/security/limits.d/90-nofile.conf"
    
    # 创建 limits.d 目录
    run_as_root mkdir -p /etc/security/limits.d
    
    # 备份原文件（如果需要）
    if [ -f "$limits_file" ] && [ ! -f "${limits_file}.bak" ]; then
        run_as_root cp "$limits_file" "${limits_file}.bak"
    fi
    
    # 定义limits配置数组
    local limits_configs=(
        "* hard memlock unlimited"
        "* hard nofile 102400"
        "* hard nproc 102400"
        "* soft memlock unlimited"
        "* soft nofile 102400"
        "* soft nproc 102400"
        "root hard nproc unlimited"
        "root soft nproc unlimited"
    )
    
    for config in "${limits_configs[@]}"; do
        local pattern=$(echo "$config" | awk '{print $1" "$2}')
        if grep -q "$pattern" "$limits_file" 2>/dev/null; then
            log_info "limits 配置已存在: $config"
        else
            run_as_root bash -c "echo '$config' >> '$limits_file'"
            log_info "添加 limits 配置: $config"
        fi
    done
    
    # 创建 limits.d/90-nofile.conf（部分系统优先读取此文件）
    run_as_root bash -c "echo '# BigData Platform Limits Configuration' > '$limits_d_file'"
    for config in "${limits_configs[@]}"; do
        run_as_root bash -c "echo '$config' >> '$limits_d_file'"
    done
    log_info "创建 $limits_d_file"
    
    # 确保 pam_limits.so 已启用
    for pam_file in /etc/pam.d/login /etc/pam.d/sshd /etc/pam.d/su; do
        if [ -f "$pam_file" ]; then
            if ! grep -q "pam_limits.so" "$pam_file"; then
                run_as_root bash -c "echo 'session required pam_limits.so' >> '$pam_file'"
                log_info "添加 pam_limits.so 到 $pam_file"
            fi
        fi
    done
    
    log_info "========== 文件描述符限制配置完成 =========="
}

# ============================================================
# 7. 禁用透明大页
# ============================================================
disable_thp() {
    log_info "========== 开始禁用透明大页 =========="
    
    # 检查当前状态
    local thp_path="/sys/kernel/mm/transparent_hugepage/enabled"
    
    if [ -f "$thp_path" ]; then
        local current=$(cat "$thp_path" | grep -o '\[.*\]' | tr -d '[]')
        
        if [ "$current" = "never" ]; then
            log_info "透明大页已禁用"
        else
            # 临时禁用
            run_as_root bash -c "echo never > /sys/kernel/mm/transparent_hugepage/enabled" 2>/dev/null && log_info "临时禁用透明大页" || log_warn "临时禁用透明大页失败"
            run_as_root bash -c "echo never > /sys/kernel/mm/transparent_hugepage/defrag" 2>/dev/null || true
        fi
    else
        log_info "系统不支持透明大页配置"
    fi
    
    # 永久禁用（通过 rc.local 或 systemd）
    local rc_local="/etc/rc.d/rc.local"
    local thp_cmds='
# Disable Transparent Huge Pages
if [ -f /sys/kernel/mm/transparent_hugepage/enabled ]; then
    echo never > /sys/kernel/mm/transparent_hugepage/enabled
    echo never > /sys/kernel/mm/transparent_hugepage/defrag
fi
'
    
    if [ -f "$rc_local" ]; then
        if grep -q "transparent_hugepage" "$rc_local"; then
            log_info "rc.local 已配置透明大页禁用"
        else
            run_as_root bash -c "echo '$thp_cmds' >> '$rc_local'"
            run_as_root chmod +x "$rc_local"
            log_info "添加透明大页禁用配置到 rc.local"
        fi
    else
        # 使用 systemd 服务（CentOS 8/9, 麒麟V10等）
        local systemd_service="/etc/systemd/system/disable-thp.service"
        run_as_root bash -c "cat > '$systemd_service' << 'EOF'
[Unit]
Description=Disable Transparent Huge Pages (THP)
DefaultDependencies=no
After=sysinit.target local-fs.target
Before=basic.target

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'echo never > /sys/kernel/mm/transparent_hugepage/enabled'
ExecStart=/bin/sh -c 'echo never > /sys/kernel/mm/transparent_hugepage/defrag'

[Install]
WantedBy=basic.target
EOF"
        run_as_root systemctl daemon-reload
        run_as_root systemctl enable disable-thp.service >/dev/null 2>&1
        log_info "创建 systemd 服务禁用透明大页"
    fi
    
    # GRUB 配置（最彻底的方式）
    if [ -f /etc/default/grub ]; then
        if grep -q "transparent_hugepage=never" /etc/default/grub; then
            log_info "GRUB 已配置 transparent_hugepage=never"
        else
            run_as_root sed -i 's/GRUB_CMDLINE_LINUX="/GRUB_CMDLINE_LINUX="transparent_hugepage=never /g' /etc/default/grub
            log_info "添加 GRUB 参数: transparent_hugepage=never"
            log_warn "需要执行 grub2-mkconfig 并重启生效"
        fi
    fi
    
    log_info "========== 透明大页配置完成 =========="
}

# ============================================================
# 8. 创建安装目录
# ============================================================
create_directories() {
    log_info "========== 开始创建安装目录 =========="

    local user="bigdata"
    local group="bigdata"

    # 创建用户和组（如果不存在）
    if ! id "$user" >/dev/null 2>&1; then
        run_as_root groupadd -f "$group" 2>/dev/null || true
        run_as_root useradd -g "$group" -s /bin/bash "$user" && log_info "创建用户: $user" || log_warn "用户可能已存在: $user"
    else
        log_info "用户已存在: $user"
    fi
    
    
    local dirs=(
        "/data/localization"
        "/data/tmp_install_dir"
    )
    local data_dir=/data 


    
    run_as_root setfacl -R -m u:$user:rwx $data_dir

    
    for dir in "${dirs[@]}"; do
        if [ -d "$dir" ]; then
            log_info "目录已存在: $dir"
        else
            run_as_root mkdir -p "$dir" && log_info "创建目录: $dir" || log_error "创建目录失败: $dir"
        fi
    done
    

    # 设置目录所有者
    for dir in "${dirs[@]}"; do
        run_as_root chown -R "$user:$group" "$dir" 2>/dev/null && log_info "设置目录所有者: $dir -> $user:$group" || true
    done
    
    log_info "========== 安装目录创建完成 =========="
}

# ============================================================
# 9. 设置时区
# ============================================================
set_timezone() {
    log_info "========== 开始设置时区 =========="
    
    local timezone="Asia/Shanghai"
    local current_tz=$(timedatectl show 2>/dev/null | grep '^Timezone=' | cut -d'=' -f2)
    
    if [ "$current_tz" = "$timezone" ]; then
        log_info "时区已正确设置: $timezone"
    else
        # 方式1: timedatectl（推荐）
        if command_exists timedatectl; then
            run_as_root timedatectl set-timezone "$timezone" && log_info "设置时区: $timezone" || log_error "设置时区失败"
        # 方式2: 符号链接
        elif [ -f "/usr/share/zoneinfo/$timezone" ]; then
            run_as_root ln -sf "/usr/share/zoneinfo/$timezone" /etc/localtime
            log_info "设置时区: $timezone"
            # 写入 /etc/sysconfig/clock（CentOS 6/7）
            if [ -d /etc/sysconfig ]; then
                run_as_root bash -c "echo 'ZONE=\"$timezone\"' > /etc/sysconfig/clock"
                run_as_root bash -c "echo 'UTC=false' >> /etc/sysconfig/clock"
                run_as_root bash -c "echo 'ARC=false' >> /etc/sysconfig/clock"
            fi
        else
            log_error "时区文件不存在: /usr/share/zoneinfo/$timezone"
        fi
    fi
    
    # 同步硬件时钟
    if command_exists hwclock; then
        run_as_root hwclock --systohc 2>/dev/null && log_info "同步硬件时钟" || true
    fi
    
    log_info "========== 时区设置完成 =========="
}

# ============================================================
# 10. 配置/etc/hosts
# ============================================================
configure_hosts() {
    log_info "========== 开始配置/etc/hosts =========="
    
    local hosts_file="/etc/hosts"
    
    # 获取第一个节点别名用于判断是否已配置过
    
    # 通过检查第一个节点别名判断是否已配置
    if grep -q "dw-master1$" "$hosts_file" 2>/dev/null; then
        log_info "/etc/hosts 已配置过，跳过"
    else
        # 添加所有节点信息
        run_as_root bash -c "echo '192.168.10.10 dw-master1 dw-master1' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.11 dw-master2 dw-master2' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.12 dw-master3 dw-master3' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.13 dw-worker1 dw-worker1' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.14 dw-worker2 dw-worker2' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.15 dw-worker3 dw-worker3' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.20 etl-ssdb etl-ssdb' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.16 realtime-es1 realtime-es1' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.17 realtime-es2 realtime-es2' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.18 realtime-es3 realtime-es3' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.13 realtime-kafka1 realtime-kafka1' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.14 realtime-kafka2 realtime-kafka2' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.15 realtime-kafka3 realtime-kafka3' >> '$hosts_file'"
        run_as_root bash -c "echo '192.168.10.19 realtime-redis realtime-redis' >> '$hosts_file'"
        log_info "添加所有节点的 hosts 映射"
    fi
    
    log_info "========== /etc/hosts配置完成 =========="
}

# ============================================================
# 11. 优化journald日志存储（改为磁盘存储）
# ============================================================
optimize_journald() {
    log_info "========== 开始优化journald配置 =========="
    
    local journald_conf="/etc/systemd/journald.conf"
    local journald_conf_d="/etc/systemd/journald.conf.d"
    
    if [ ! -f "$journald_conf" ]; then
        log_info "journald 配置文件不存在，跳过"
        return 0
    fi
    
    # 创建 journald.conf.d 目录
    run_as_root mkdir -p "$journald_conf_d"
    
    # 创建自定义配置文件
    local custom_conf="$journald_conf_d/99-bigdata.conf"
    
    # 存储方式改为磁盘
    if grep -q "^Storage=auto" "$custom_conf" 2>/dev/null; then
        log_info "journald Storage 已配置为 auto"
    else
        run_as_root bash -c "cat > '$custom_conf' << 'EOF'
[Journal]
# 存储方式：auto（自动选择，优先持久化存储）
Storage=auto
# 压缩：启用
Compress=yes
# 日志最大大小
SystemMaxUse=2G
# 单个日志文件最大大小
SystemMaxFileSize=100M
# 日志保留天数
MaxRetentionSec=7day
EOF"
        log_info "创建 journald 优化配置: $custom_conf"
    fi
    
    # 重启 journald 服务
    if command_exists systemctl; then
        run_as_root systemctl restart systemd-journald 2>/dev/null && log_info "重启 journald 服务" || log_warn "重启 journald 服务失败"
    fi
    
    log_info "========== journald配置完成 =========="
}

# ============================================================
# 12. 配置crontab权限
# ============================================================
configure_cron() {
    log_info "========== 开始配置crontab权限 =========="
    
    local cron_allow="/etc/cron.allow"
    local cron_deny="/etc/cron.deny"
    
    # 如果存在 cron.deny，需要处理
    if [ -f "$cron_deny" ]; then
        # 备份
        run_as_root mv "$cron_deny" "${cron_deny}.bak" 2>/dev/null
        log_info "备份 cron.deny -> cron.deny.bak"
    fi
    
    # 创建 cron.allow 文件
    if [ ! -f "$cron_allow" ]; then
        run_as_root touch "$cron_allow"
        run_as_root chmod 600 "$cron_allow"
        log_info "创建 cron.allow 文件"
    fi
    
    # 添加允许的用户（root 和 global.user）
    local cron_users="root bigdata"
    
    for user in $cron_users; do
        if grep -q "^${user}$" "$cron_allow" 2>/dev/null; then
            log_info "用户已有 crontab 权限: $user"
        else
            run_as_root bash -c "echo '$user' >> '$cron_allow'"
            log_info "添加 crontab 权限: $user"
        fi
    done
    
    # 确保 crond 服务运行
    if command_exists systemctl; then
        if ! systemctl is-active crond >/dev/null 2>&1; then
            run_as_root systemctl start crond && log_info "启动 crond 服务" || log_warn "启动 crond 服务失败"
        fi
        if ! systemctl is-enabled crond >/dev/null 2>&1; then
            run_as_root systemctl enable crond && log_info "设置 crond 开机自启" || true
        fi
    fi
    
    log_info "========== crontab权限配置完成 =========="
}

# ============================================================
# 主函数
# ============================================================
main() {
    log_info "============================================"
    log_info "服务器初始化开始"
    log_info "节点: dw-worker2"
    log_info "IP: 192.168.10.14"
    log_info "系统: $(cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'=' -f2 | tr -d '\"')"
    log_info "============================================"
    
    # 检测操作系统
    detect_os
    
    # 执行初始化步骤
    disable_firewall
    disable_selinux
    disable_swap
    load_kernel_modules
    configure_sysctl
    configure_limits
    disable_thp
    create_directories
    set_timezone
    configure_hosts
    optimize_journald
    configure_cron
    
    # 创建完成标记
    run_as_root touch "$MARKER_FILE"
    
    log_info "============================================"
    log_info "服务器初始化完成！"
    log_info "日志文件: $LOG_FILE"
    log_info "标记文件: $MARKER_FILE"
    log_info "============================================"
    
    echo ""
    echo "================================================================"
    echo "重要提示："
    echo "1. SELinux 配置已修改，建议重启系统以完全生效"
    echo "2. 透明大页已禁用，建议执行 'grub2-mkconfig -o /boot/grub2/grub.cfg' 并重启"
    echo "3. 文件描述符限制已配置，新登录会话生效"
    echo "================================================================"
}

# 执行主函数
main "$@"
