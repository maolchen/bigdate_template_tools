#!/bin/bash
# ============================================================
# Scala 安装脚本
# 服务: scala
# 节点: dw-master2 (192.168.10.11)
# ============================================================

set -e

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ==================== 变量定义 ====================
SCALA_VERSION="2.11.6"
SCALA_INSTALL_DIR="/data/scala"
SCALA_PACKAGE="/data/softwares/realtime-new-doris/realtime/kafka/scala-${SCALA_VERSION}.tgz"
SYSTEM_USER="bigdata"

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
check_scala_installed() {
    if [ -f "${SCALA_INSTALL_DIR}/bin/scala" ]; then
        return 0
    fi
    return 1
}

check_java_installed() {
    if command -v java &> /dev/null || [ -n "$JAVA_HOME" ]; then
        return 0
    fi
    return 1
}

# ==================== 安装函数 ====================
install_scala() {
    log_info "开始安装Scala ${SCALA_VERSION}..."
    
    # 检查Java是否已安装
    if ! check_java_installed; then
        log_error "Java未安装，Scala需要Java环境，请先安装JDK"
        return 1
    fi
    log_info "Java环境检查通过"
    
    # 检查是否已安装
    if check_scala_installed; then
        log_success "Scala已安装在 ${SCALA_INSTALL_DIR}"
        return 0
    fi
    
    # 检查安装包
    if [ ! -f "$SCALA_PACKAGE" ]; then
        log_error "Scala安装包不存在: $SCALA_PACKAGE"
        return 1
    fi
    log_info "安装包: $SCALA_PACKAGE"
    
    # 创建安装目录的父目录
    local parent_dir=$(dirname "$SCALA_INSTALL_DIR")
    if [ ! -d "$parent_dir" ]; then
        log_info "创建父目录: $parent_dir"
        run_as_root mkdir -p "$parent_dir"
    fi
    
    # 解压安装包
    log_info "解压Scala安装包..."
    local tmp_dir="/tmp/scala_install_$$"
    mkdir -p "$tmp_dir"
    tar -xzf "$SCALA_PACKAGE" -C "$tmp_dir"
    
    # 查找解压后的目录（可能是scala-2.11.6这样的名称）
    local extracted_dir=$(ls -d "$tmp_dir"/scala-* 2>/dev/null | head -1)
    if [ -z "$extracted_dir" ]; then
        log_error "无法找到解压后的Scala目录"
        rm -rf "$tmp_dir"
        return 1
    fi
    
    log_info "解压目录: $extracted_dir"
    
    # 移动到目标目录
    log_info "安装Scala到 ${SCALA_INSTALL_DIR}..."
    
    if [ -d "$SCALA_INSTALL_DIR" ]; then
        log_warn "目标目录已存在，备份..."
        run_as_root mv -f "$SCALA_INSTALL_DIR" "${SCALA_INSTALL_DIR}.bak.$(date +%Y%m%d%H%M%S)"
    fi
    
    # 创建目标目录并移动内容
    run_as_root mkdir -p "$SCALA_INSTALL_DIR"
    run_as_root cp -r "$extracted_dir"/* "$SCALA_INSTALL_DIR"/
    
    # 清理临时文件
    rm -rf "$tmp_dir"
    
    # 设置权限
    log_info "设置Scala目录权限..."
    if [ -n "$SYSTEM_USER" ]; then
        run_as_root chown -R ${SYSTEM_USER}:${SYSTEM_USER} "$SCALA_INSTALL_DIR"
    fi
    
    log_success "Scala安装完成: ${SCALA_INSTALL_DIR}"
    return 0
}

# ==================== 环境变量配置函数 ====================
configure_environment() {
    log_info "配置Scala环境变量..."
    
    # 环境变量内容
    local env_content="
# Scala Environment Variables
export SCALA_HOME=${SCALA_INSTALL_DIR}
export PATH=\$PATH:\${SCALA_HOME}/bin
export SCALA_HOME PATH
"
    
    # 判断当前用户
    local current_user=$(whoami)
    
    if [ "$current_user" = "root" ]; then
        # root用户：直接配置到/etc/profile
        log_info "当前用户为root，配置环境变量到 /etc/profile"
        
        # 检查是否已配置
        if grep -q "SCALA_HOME=${SCALA_INSTALL_DIR}" /etc/profile 2>/dev/null; then
            log_info "环境变量已在/etc/profile中配置，跳过"
        else
            run_as_root bash -c "echo '$env_content' >> /etc/profile"
            log_success "环境变量已添加到 /etc/profile"
        fi
    else
        # 非root用户
        log_info "当前用户为 ${current_user}"
        
        # 配置到用户的 ~/.bash_profile
        local bash_profile="$HOME/.bash_profile"
        if [ ! -f "$bash_profile" ]; then
            bash_profile="$HOME/.bashrc"
        fi
        
        if grep -q "SCALA_HOME=${SCALA_INSTALL_DIR}" "$bash_profile" 2>/dev/null; then
            log_info "环境变量已在 $bash_profile 中配置，跳过"
        else
            echo "$env_content" >> "$bash_profile"
            log_success "环境变量已添加到 $bash_profile"
        fi
        
        # 如果指定了系统用户，使用sudo添加到/etc/profile
        if [ -n "$SYSTEM_USER" ]; then
            log_info "使用sudo添加环境变量到 /etc/profile"
            if grep -q "SCALA_HOME=${SCALA_INSTALL_DIR}" /etc/profile 2>/dev/null; then
                log_info "环境变量已在/etc/profile中配置，跳过"
            else
                run_as_root bash -c "echo '$env_content' >> /etc/profile"
                log_success "环境变量已添加到 /etc/profile"
            fi
        fi
    fi
    
    # 使环境变量生效
    log_info "使环境变量生效..."
    export SCALA_HOME="${SCALA_INSTALL_DIR}"
    export PATH=$PATH:"${SCALA_INSTALL_DIR}/bin"
    
    log_success "环境变量配置完成"
    return 0
}

# ==================== 创建软链接 ====================
create_symlinks() {
    log_info "创建Scala命令软链接..."
    
    local scala_bin="${SCALA_INSTALL_DIR}/bin"
    local bin_dirs=("/usr/local/bin" "/usr/bin")
    
    for cmd in scala scalac scaladoc scalap; do
        if [ -f "${scala_bin}/${cmd}" ]; then
            for bin_dir in "${bin_dirs[@]}"; do
                # 创建软链接
                run_as_root ln -sf "${scala_bin}/${cmd}" "${bin_dir}/${cmd}" 2>/dev/null || true
            done
        fi
    done
    
    log_success "Scala命令软链接创建完成"
    return 0
}

# ==================== 验证函数 ====================
verify_installation() {
    log_info "验证Scala安装..."
    
    # 检查scala命令
    local scala_cmd="${SCALA_INSTALL_DIR}/bin/scala"
    if [ ! -f "$scala_cmd" ]; then
        log_error "Scala命令不存在: $scala_cmd"
        return 1
    fi
    
    # 检查版本
    log_info "Scala版本信息:"
    "$scala_cmd" -version
    
    # 检查环境变量
    if [ -n "$SCALA_HOME" ]; then
        log_info "SCALA_HOME: $SCALA_HOME"
    else
        log_warn "SCALA_HOME未设置，请重新登录或执行: source /etc/profile"
    fi
    
    # 验证scalac编译器
    if [ -f "${SCALA_INSTALL_DIR}/bin/scalac" ]; then
        log_success "Scala安装验证通过（包含编译工具）"
    else
        log_success "Scala安装验证通过"
    fi
    
    return 0
}

# ==================== 主函数 ====================
main() {
    echo ""
    echo "========================================"
    echo "  Scala 安装脚本"
    echo "  节点: dw-master2 (192.168.10.11)"
    echo "  版本: ${SCALA_VERSION}"
    echo "  安装路径: ${SCALA_INSTALL_DIR}"
    echo "  系统用户: ${SYSTEM_USER}"
    echo "========================================"
    echo ""
    
    # 安装Scala
    if ! install_scala; then
        log_error "Scala安装失败！"
        exit 1
    fi
    
    # 配置环境变量
    if ! configure_environment; then
        log_error "环境变量配置失败！"
        exit 1
    fi
    
    # 创建软链接
    create_symlinks
    
    # 验证安装
    if verify_installation; then
        log_success "Scala安装完成！"
        echo ""
        log_info "安装信息:"
        echo "  - 安装路径: ${SCALA_INSTALL_DIR}"
        echo "  - Scala版本: ${SCALA_VERSION}"
        echo "  - SCALA_HOME: ${SCALA_INSTALL_DIR}"
        echo ""
        log_info "请执行以下命令使环境变量生效:"
        echo "  source /etc/profile"
        echo "  或者重新登录系统"
        echo ""
        exit 0
    else
        log_error "Scala安装验证失败！"
        exit 1
    fi
}

# 执行主函数
main "$@"
