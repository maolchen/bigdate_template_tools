#!/bin/bash
# ============================================================
# Docker 安装脚本
# 服务: docker
# 节点: dw-master3 (192.168.10.12)
# ============================================================

set -e

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ==================== 变量定义 ====================
DOCKER_VERSION="26.1.3"
DOCKER_BIN_PATH="/usr/local/bin"
DOCKER_PACKAGE_DIR="/data/softwares/realtime-new-doris/realtime/web_all/docker"
DOCKER_PACKAGE="${DOCKER_PACKAGE_DIR}/docker-${DOCKER_VERSION}.tgz"
DOCKER_SERVICE_FILE="/etc/systemd/system/docker.service"
DOCKER_SOCKET_FILE="/etc/systemd/system/docker.socket"
DOCKER_USER="docker"
SYSTEM_USER="bigdata"

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
check_docker_installed() {
    if command -v docker &> /dev/null; then
        local current_version=$(docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//')
        if [ "$current_version" = "$DOCKER_VERSION" ]; then
            return 0  # 已安装相同版本
        else
            log_warn "Docker已安装，版本: $current_version，目标版本: $DOCKER_VERSION"
            return 1  # 已安装不同版本
        fi
    fi
    return 2  # 未安装
}

check_docker_service() {
    if systemctl is-active docker &> /dev/null; then
        return 0
    fi
    return 1
}

# ==================== 安装函数 ====================
install_docker() {
    log_info "开始安装Docker ${DOCKER_VERSION}..."
    
    # 1. 检查安装包
    if [ ! -f "$DOCKER_PACKAGE" ]; then
        log_error "Docker安装包不存在: $DOCKER_PACKAGE"
        return 1
    fi
    log_info "安装包检查通过: $DOCKER_PACKAGE"
    
    # 2. 创建docker用户和组（如果不存在）
    if ! id "$DOCKER_USER" &>/dev/null; then
        log_info "创建docker用户组..."
        groupadd -r docker
        log_info "docker用户组创建成功"
    else
        log_info "docker用户组已存在"
    fi
    
    # 3. 解压安装包
    log_info "解压Docker安装包..."
    local tmp_dir="/tmp/docker_install_$$"
    mkdir -p "$tmp_dir"
    tar -xzf "$DOCKER_PACKAGE" -C "$tmp_dir"
    
    # 4. 复制二进制文件到目标目录
    log_info "安装Docker二进制文件到 ${DOCKER_BIN_PATH}..."
    if [ -d "$tmp_dir/docker" ]; then
        cp -f "$tmp_dir/docker/"* "${DOCKER_BIN_PATH}/"
    else
        # 某些安装包可能直接包含文件而不是docker子目录
        cp -f "$tmp_dir/"* "${DOCKER_BIN_PATH}/" 2>/dev/null || true
    fi
    
    # 5. 创建软链接到/usr/bin（如果bin_path不是/usr/bin）
    if [ "$DOCKER_BIN_PATH" != "/usr/bin" ]; then
        log_info "创建软链接到 /usr/bin..."
        for bin_file in docker dockerd containerd containerd-shim containerd-shim-runc-v2 runc ctr; do
            if [ -f "${DOCKER_BIN_PATH}/${bin_file}" ]; then
                ln -sf "${DOCKER_BIN_PATH}/${bin_file}" "/usr/bin/${bin_file}"
            fi
        done
    fi
    
    # 6. 设置权限
    log_info "设置文件权限..."
    chmod +x "${DOCKER_BIN_PATH}"/docker* 2>/dev/null || true
    chmod +x "${DOCKER_BIN_PATH}"/containerd* 2>/dev/null || true
    chmod +x "${DOCKER_BIN_PATH}"/runc 2>/dev/null || true
    chmod +x "${DOCKER_BIN_PATH}"/ctr 2>/dev/null || true
    
    # 7. 清理临时文件
    rm -rf "$tmp_dir"
    log_info "Docker二进制文件安装完成"
    
    # 8. 安装systemd服务文件
    install_systemd_service
    
    # 9. 将系统用户加入docker组
    if id "$SYSTEM_USER" &>/dev/null; then
        log_info "将用户 ${SYSTEM_USER} 加入docker组..."
        usermod -aG docker "$SYSTEM_USER"
        log_info "用户 ${SYSTEM_USER} 已加入docker组"
    fi
    
    # 10. 启动Docker服务
    start_docker_service
    
    return 0
}

install_systemd_service() {
    log_info "配置Docker systemd服务..."
    
    # 创建docker.socket文件
    cat > "$DOCKER_SOCKET_FILE" << 'SOCKETEOF'
[Unit]
Description=Docker Socket for the API

[Socket]
ListenStream=/var/run/docker.sock
SocketMode=0660
SocketUser=root
SocketGroup=docker

[Install]
WantedBy=sockets.target
SOCKETEOF
    
    # 复制docker.service文件（从模板目录）
    # 注意：实际部署时需要将docker.service.tmpl渲染后的文件放到对应位置
    # 这里我们直接使用内嵌的服务配置
    cat > "$DOCKER_SERVICE_FILE" << 'SERVICEEOF'
[Unit]
Description=Docker Application Container Engine
Documentation=https://docs.docker.com
After=network-online.target docker.socket firewalld.service containerd.service
Wants=network-online.target
Requires=docker.socket

[Service]
Type=notify
ExecStart=/usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock
ExecReload=/bin/kill -s HUP $MAINPID
TimeoutSec=0
RestartSec=2
Restart=always
StartLimitBurst=3
StartLimitIntervalSec=60s
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
Delegate=yes
KillMode=process
OOMScoreAdjust=-500

[Install]
WantedBy=multi-user.target
SERVICEEOF
    
    # 重载systemd
    systemctl daemon-reload
    log_info "Docker systemd服务配置完成"
}

start_docker_service() {
    log_info "启动Docker服务..."
    
    # 启用并启动服务
    systemctl enable docker.socket
    systemctl enable docker.service
    systemctl start docker.socket
    systemctl start docker.service
    
    # 等待服务启动
    sleep 3
    
    # 验证服务状态
    if systemctl is-active docker &>/dev/null; then
        log_success "Docker服务启动成功"
    else
        log_error "Docker服务启动失败"
        return 1
    fi
    
    return 0
}

# ==================== 验证函数 ====================
verify_installation() {
    log_info "验证Docker安装..."
    
    # 检查docker命令
    if ! command -v docker &> /dev/null; then
        log_error "docker命令不可用"
        return 1
    fi
    
    # 检查版本
    local current_version=$(docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//')
    log_info "Docker版本: $current_version"
    
    # 检查服务状态
    if systemctl is-active docker &>/dev/null; then
        log_success "Docker服务运行正常"
    else
        log_error "Docker服务未运行"
        return 1
    fi
    
    # 测试docker命令
    log_info "测试docker命令..."
    if docker info &>/dev/null; then
        log_success "Docker命令测试通过"
    else
        log_warn "Docker命令测试失败，可能需要重启系统或重新登录"
    fi
    
    return 0
}

# ==================== 主函数 ====================
main() {
    echo ""
    echo "========================================"
    echo "  Docker 安装脚本"
    echo "  版本: ${DOCKER_VERSION}"
    echo "  节点: dw-master3 (192.168.10.12)"
    echo "========================================"
    echo ""
    
    # 检查是否已安装
    check_docker_installed
    local check_result=$?
    
    if [ $check_result -eq 0 ]; then
        log_success "Docker ${DOCKER_VERSION} 已安装"
        verify_installation
        exit 0
    elif [ $check_result -eq 1 ]; then
        log_warn "检测到不同版本的Docker，将重新安装"
        # 停止现有服务
        systemctl stop docker 2>/dev/null || true
        systemctl stop docker.socket 2>/dev/null || true
    fi
    
    # 执行安装
    if install_docker; then
        log_success "Docker安装完成！"
        echo ""
        verify_installation
        echo ""
        log_info "安装信息:"
        echo "  - 版本: ${DOCKER_VERSION}"
        echo "  - 安装路径: ${DOCKER_BIN_PATH}"
        echo "  - 服务状态: $(systemctl is-active docker 2>/dev/null || echo '未运行')"
        echo "  - 用户 ${SYSTEM_USER} 已加入docker组"
        echo ""
        log_info "如需使用docker命令，请重新登录或执行: newgrp docker"
        exit 0
    else
        log_error "Docker安装失败！"
        exit 1
    fi
}

# 执行主函数
main "$@"
