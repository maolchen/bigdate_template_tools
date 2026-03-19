#!/bin/bash
# ============================================================
# Python3 安装脚本
# 服务: python3
# 节点: dw-master3 (192.168.10.12)
# ============================================================

set -e

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ==================== 变量定义 ====================
PYTHON3_VERSION="3.6.8"
PYTHON3_VERSION_SETUPTOOLS="40.0.0"
PYTHON3_VERSION_PIP="18.0"
PYTHON3_PACKAGE="/data/softwares/realtime-new-doris/realtime/python3/Python-3.6.8.tar.xz"
PYTHON3_TMP_DIR="/data/tmp_install_dir"
PYTHON3_SOURCE_DIR="${PYTHON3_TMP_DIR}/Python-${PYTHON3_VERSION}"

# 从版本号提取主版本号（如3.6.8 -> 3.6）
PYTHON3_MAJOR_VERSION=$(echo "$PYTHON3_VERSION" | cut -d. -f1,2)

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
check_python3_installed() {
    if command -v python3 &> /dev/null; then
        return 0
    fi
    if [ -x /usr/bin/python3 ]; then
        return 0
    fi
    if [ -x /usr/local/bin/python3 ]; then
        return 0
    fi
    return 1
}

get_python3_path() {
    if command -v python3 &> /dev/null; then
        which python3
    elif [ -x /usr/local/bin/python3 ]; then
        echo "/usr/local/bin/python3"
    elif [ -x /usr/bin/python3 ]; then
        echo "/usr/bin/python3"
    else
        echo ""
    fi
}

# ==================== 安装依赖包 ====================
install_dependencies() {
    log_info "安装Python3编译依赖包..."
    
    local deps=(
        "zlib-devel"
        "bzip2-devel"
        "openssl-devel"
        "ncurses-devel"
        "gcc"
        "make"
    )
    
    if command -v yum &> /dev/null; then
        log_info "使用yum安装依赖包..."
        yum install -y "${deps[@]}"
    elif command -v dnf &> /dev/null; then
        log_info "使用dnf安装依赖包..."
        dnf install -y "${deps[@]}"
    elif command -v apt-get &> /dev/null; then
        log_info "使用apt-get安装依赖包..."
        apt-get update
        apt-get install -y zlib1g-dev libbz2-dev libssl-dev libncurses5-dev gcc make
    else
        log_error "无法找到可用的包管理器（yum/dnf/apt-get）"
        return 1
    fi
    
    log_success "依赖包安装完成"
    return 0
}

# ==================== 解压源码包 ====================
extract_source() {
    log_info "解压Python3源码包..."
    
    # 检查源码包
    if [ ! -f "$PYTHON3_PACKAGE" ]; then
        log_error "Python3源码包不存在: $PYTHON3_PACKAGE"
        return 1
    fi
    
    # 创建临时目录
    if [ ! -d "$PYTHON3_TMP_DIR" ]; then
        mkdir -p "$PYTHON3_TMP_DIR"
    fi
    
    # 检查是否已解压
    if [ -d "$PYTHON3_SOURCE_DIR" ]; then
        log_info "源码目录已存在: $PYTHON3_SOURCE_DIR"
        return 0
    fi
    
    # 解压（支持.tar.xz和.tar.gz）
    if [[ "$PYTHON3_PACKAGE" == *.tar.xz ]]; then
        tar -xJf "$PYTHON3_PACKAGE" -C "$PYTHON3_TMP_DIR"
    elif [[ "$PYTHON3_PACKAGE" == *.tar.gz ]]; then
        tar -xzf "$PYTHON3_PACKAGE" -C "$PYTHON3_TMP_DIR"
    else
        log_error "不支持的压缩格式: $PYTHON3_PACKAGE"
        return 1
    fi
    
    log_success "源码包解压完成: $PYTHON3_SOURCE_DIR"
    return 0
}

# ==================== 编译安装 ====================
compile_install() {
    log_info "编译安装Python3..."
    
    cd "$PYTHON3_SOURCE_DIR"
    
    # 运行configure
    log_info "运行configure..."
    ./configure --enable-shared
    
    # 编译
    log_info "编译Python3 (make all)..."
    make all
    
    # 安装
    log_info "安装Python3 (make install)..."
    make install
    
    log_success "Python3编译安装完成"
    return 0
}

# ==================== 复制共享库 ====================
copy_shared_lib() {
    log_info "复制Python3共享库到系统目录..."
    
    cd "$PYTHON3_SOURCE_DIR"
    
    # 查找并复制共享库
    # 库文件名格式: libpython3.6m.so.1.0
    local lib_pattern="libpython3.*.so.*"
    local target_lib="/usr/lib64/libpython${PYTHON3_MAJOR_VERSION}m.so.1.0"
    
    # 检查目标库是否已存在
    if [ -f "$target_lib" ]; then
        log_info "共享库已存在: $target_lib"
        return 0
    fi
    
    # 复制所有匹配的库文件
    for lib in $lib_pattern; do
        if [ -f "$lib" ]; then
            log_info "复制: $lib -> /usr/lib64/"
            cp -a "$lib" /usr/lib64/
        fi
    done
    
    # 更新动态链接库缓存
    log_info "更新动态链接库缓存..."
    ldconfig
    
    log_success "共享库复制完成"
    return 0
}

# ==================== 验证安装 ====================
verify_installation() {
    log_info "验证Python3安装..."
    
    local python3_path=$(get_python3_path)
    
    if [ -z "$python3_path" ]; then
        log_error "Python3安装失败，找不到python3命令"
        return 1
    fi
    
    log_info "Python3路径: $python3_path"
    
    # 显示版本
    log_info "Python3版本信息:"
    "$python3_path" --version
    
    # 检查pip
    if command -v pip3 &> /dev/null; then
        log_info "pip3版本:"
        pip3 --version
    elif [ -x /usr/local/bin/pip3 ]; then
        log_info "pip3版本:"
        /usr/local/bin/pip3 --version
    else
        log_warn "pip3未找到，可能需要手动安装"
    fi
    
    # 检查共享库
    local lib_path="/usr/lib64/libpython${PYTHON3_MAJOR_VERSION}m.so.1.0"
    if [ -f "$lib_path" ]; then
        log_success "共享库验证通过: $lib_path"
    else
        log_warn "共享库未找到: $lib_path"
    fi
    
    log_success "Python3安装验证通过"
    return 0
}

# ==================== 清理函数 ====================
cleanup() {
    log_info "清理临时文件..."
    
    # 可选：删除源码目录以节省空间
    # 如果需要保留源码，可以注释掉这一行
    # rm -rf "$PYTHON3_SOURCE_DIR"
    
    log_info "临时文件清理完成"
}

# ==================== 主函数 ====================
main() {
    echo ""
    echo "========================================"
    echo "  Python3 安装脚本"
    echo "  节点: dw-master3 (192.168.10.12)"
    echo "  版本: ${PYTHON3_VERSION}"
    echo "========================================"
    echo ""
    
    # 检查是否已安装
    if check_python3_installed; then
        local current_path=$(get_python3_path)
        local current_version=$("$current_path" --version 2>&1 | awk '{print $2}')
        log_success "Python3已安装: $current_path"
        log_info "当前版本: $current_version"
        
        # 检查版本是否匹配
        if [ "$current_version" = "$PYTHON3_VERSION" ]; then
            log_success "版本匹配，跳过安装"
            exit 0
        else
            log_warn "当前版本 ($current_version) 与目标版本 ($PYTHON3_VERSION) 不匹配"
            log_info "继续安装目标版本..."
        fi
    fi
    
    # 安装依赖
    if ! install_dependencies; then
        log_error "依赖包安装失败！"
        exit 1
    fi
    
    # 解压源码
    if ! extract_source; then
        log_error "源码解压失败！"
        exit 1
    fi
    
    # 编译安装
    if ! compile_install; then
        log_error "Python3编译安装失败！"
        exit 1
    fi
    
    # 复制共享库
    if ! copy_shared_lib; then
        log_error "共享库复制失败！"
        exit 1
    fi
    
    # 清理
    cleanup
    
    # 验证安装
    if verify_installation; then
        log_success "Python3安装完成！"
        echo ""
        log_info "安装信息:"
        echo "  - Python版本: ${PYTHON3_VERSION}"
        echo "  - 安装路径: /usr/local/bin/python3"
        echo "  - pip版本: ${PYTHON3_VERSION_PIP}"
        echo "  - setuptools版本: ${PYTHON3_VERSION_SETUPTOOLS}"
        echo ""
        exit 0
    else
        log_error "Python3安装验证失败！"
        exit 1
    fi
}

# 执行主函数
main "$@"
