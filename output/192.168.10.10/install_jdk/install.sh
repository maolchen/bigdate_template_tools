#!/bin/bash
# ============================================================
# JDK 安装脚本
# 服务: install_jdk
# 节点: dw-master1 (192.168.10.10)
# ============================================================

set -e

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ==================== 变量定义 ====================
JDK_PACKAGE="/data/softwares/realtime-new-doris/realtime/kafka/jdk-8u202-linux-x64.tar.gz"
JDK_INSTALL_PATH="/data/jdk"
SYSTEM_USER="bigdata"

# 从安装包名称提取JDK版本目录名
# 例如: jdk-8u202-linux-x64.tar.gz -> jdk1.8.0_202
JDK_PKG_NAME="jdk-8u202-linux-x64.tar.gz"
JDK_VERSION_DIR=$(echo "$JDK_PKG_NAME" | sed 's/-linux-x64.tar.gz//' | sed 's/jdk-/jdk1./' | sed 's/u/0_/' | sed 's/\([0-9]\+\)0_/\1_/')

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
check_java_installed() {
    if command -v java &> /dev/null; then
        return 0
    fi
    return 1
}

check_jdk_installed() {
    if [ -f "${JDK_INSTALL_PATH}/bin/java" ]; then
        return 0
    fi
    return 1
}

get_current_java_path() {
    which java 2>/dev/null || echo ""
}

# ==================== 备份系统Java ====================
backup_system_java() {
    log_info "检查系统已安装的Java..."
    
    if check_java_installed; then
        local java_path=$(get_current_java_path)
        log_warn "检测到系统已安装Java: $java_path"
        
        # 查找所有可能的java命令位置
        local java_dirs=("/usr/bin" "/bin" "/usr/local/bin")
        
        for dir in "${java_dirs[@]}"; do
            if [ -f "$dir/java" ]; then
                # 检查是否是我们安装的JDK
                if [[ "$dir/java" == "${JDK_INSTALL_PATH}/bin/java" ]]; then
                    continue
                fi
                
                # 备份系统java命令
                if [ ! -f "${dir}/java.bak" ]; then
                    log_info "备份系统Java: ${dir}/java -> ${dir}/java.bak"
                    run_as_root mv -f "${dir}/java" "${dir}/java.bak"
                else
                    log_info "已存在备份文件: ${dir}/java.bak"
                fi
            fi
        done
        
        # 查找javac, jar等命令并备份
        for cmd in javac jar javaws; do
            for dir in "${java_dirs[@]}"; do
                if [ -f "$dir/$cmd" ] && [ ! -f "${dir}/${cmd}.bak" ]; then
                    if [[ "$dir/$cmd" != "${JDK_INSTALL_PATH}/bin/$cmd" ]]; then
                        log_info "备份: ${dir}/${cmd} -> ${dir}/${cmd}.bak"
                        run_as_root mv -f "${dir}/${cmd}" "${dir}/${cmd}.bak"
                    fi
                fi
            done
        done
        
        log_success "系统Java备份完成"
    else
        log_info "未检测到系统已安装的Java"
    fi
}

# ==================== 安装函数 ====================
install_jdk() {
    log_info "开始安装JDK..."
    
    # 检查是否已安装
    if check_jdk_installed; then
        log_success "JDK已安装在 ${JDK_INSTALL_PATH}"
        return 0
    fi
    
    # 检查安装包
    if [ ! -f "$JDK_PACKAGE" ]; then
        log_error "JDK安装包不存在: $JDK_PACKAGE"
        return 1
    fi
    log_info "安装包: $JDK_PACKAGE"
    
    # 创建安装目录的父目录
    local parent_dir=$(dirname "$JDK_INSTALL_PATH")
    if [ ! -d "$parent_dir" ]; then
        log_info "创建父目录: $parent_dir"
        run_as_root mkdir -p "$parent_dir"
    fi
    
    # 解压安装包
    log_info "解压JDK安装包..."
    local tmp_dir="/tmp/jdk_install_$$"
    mkdir -p "$tmp_dir"
    tar -xzf "$JDK_PACKAGE" -C "$tmp_dir"
    
    # 查找解压后的目录（可能是jdk1.8.0_202这样的名称）
    local extracted_dir=$(ls -d "$tmp_dir"/jdk* 2>/dev/null | head -1)
    if [ -z "$extracted_dir" ]; then
        log_error "无法找到解压后的JDK目录"
        rm -rf "$tmp_dir"
        return 1
    fi
    
    log_info "解压目录: $extracted_dir"
    
    # 移动到目标目录
    # 注意：用户要求jdk_path是/data/jdk，bin就是/data/jdk/bin
    # 所以直接将解压的内容移动到目标目录，而不是移动目录本身
    log_info "安装JDK到 ${JDK_INSTALL_PATH}..."
    
    if [ -d "$JDK_INSTALL_PATH" ]; then
        log_warn "目标目录已存在，备份..."
        run_as_root mv -f "$JDK_INSTALL_PATH" "${JDK_INSTALL_PATH}.bak.$(date +%Y%m%d%H%M%S)"
    fi
    
    # 创建目标目录并移动内容
    run_as_root mkdir -p "$JDK_INSTALL_PATH"
    run_as_root cp -r "$extracted_dir"/* "$JDK_INSTALL_PATH"/
    
    # 清理临时文件
    rm -rf "$tmp_dir"
    
    # 设置权限
    log_info "设置JDK目录权限..."
    if [ -n "$SYSTEM_USER" ]; then
        run_as_root chown -R ${SYSTEM_USER}:${SYSTEM_USER} "$JDK_INSTALL_PATH"
    fi
    
    log_success "JDK安装完成: ${JDK_INSTALL_PATH}"
    return 0
}

# ==================== 环境变量配置函数 ====================
configure_environment() {
    log_info "配置JDK环境变量..."
    
    # 环境变量内容
    local env_content="
# JDK Environment Variables
export JAVA_HOME=${JDK_INSTALL_PATH}
export JAVA_BIN=\${JAVA_HOME}/bin
export PATH=\$PATH:\${JAVA_HOME}/bin
export CLASSPATH=.:\${JAVA_HOME}/lib/dt.jar:\${JAVA_HOME}/lib/tools.jar
export JAVA_HOME JAVA_BIN PATH CLASSPATH
"
    
    # 判断当前用户
    local current_user=$(whoami)
    
    if [ "$current_user" = "root" ]; then
        # root用户：直接配置到/etc/profile
        log_info "当前用户为root，配置环境变量到 /etc/profile"
        
        # 检查是否已配置
        if grep -q "JAVA_HOME=${JDK_INSTALL_PATH}" /etc/profile 2>/dev/null; then
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
        
        if grep -q "JAVA_HOME=${JDK_INSTALL_PATH}" "$bash_profile" 2>/dev/null; then
            log_info "环境变量已在 $bash_profile 中配置，跳过"
        else
            echo "$env_content" >> "$bash_profile"
            log_success "环境变量已添加到 $bash_profile"
        fi
        
        # 如果指定了系统用户，使用sudo添加到/etc/profile
        if [ -n "$SYSTEM_USER" ]; then
            log_info "使用sudo添加环境变量到 /etc/profile"
            if grep -q "JAVA_HOME=${JDK_INSTALL_PATH}" /etc/profile 2>/dev/null; then
                log_info "环境变量已在/etc/profile中配置，跳过"
            else
                run_as_root bash -c "echo '$env_content' >> /etc/profile"
                log_success "环境变量已添加到 /etc/profile"
            fi
        fi
    fi
    
    # 使环境变量生效
    log_info "使环境变量生效..."
    export JAVA_HOME="${JDK_INSTALL_PATH}"
    export JAVA_BIN="${JDK_INSTALL_PATH}/bin"
    export PATH=$PATH:"${JDK_INSTALL_PATH}/bin"
    export CLASSPATH=".:${JDK_INSTALL_PATH}/lib/dt.jar:${JDK_INSTALL_PATH}/lib/tools.jar"
    
    log_success "环境变量配置完成"
    return 0
}

# ==================== 创建软链接 ====================
create_symlinks() {
    log_info "创建Java命令软链接..."
    
    local java_bin="${JDK_INSTALL_PATH}/bin"
    local bin_dirs=("/usr/local/bin" "/usr/bin")
    
    for cmd in java javac jar javah javap javadoc; do
        if [ -f "${java_bin}/${cmd}" ]; then
            for bin_dir in "${bin_dirs[@]}"; do
                # 跳过已备份的文件
                if [ -f "${bin_dir}/${cmd}.bak" ]; then
                    log_info "跳过 ${bin_dir}/${cmd}（已有备份）"
                    continue
                fi
                
                # 创建软链接
                run_as_root ln -sf "${java_bin}/${cmd}" "${bin_dir}/${cmd}"
            done
        fi
    done
    
    log_success "Java命令软链接创建完成"
    return 0
}

# ==================== 验证函数 ====================
verify_installation() {
    log_info "验证JDK安装..."
    
    # 检查java命令
    local java_cmd="${JDK_INSTALL_PATH}/bin/java"
    if [ ! -f "$java_cmd" ]; then
        log_error "Java命令不存在: $java_cmd"
        return 1
    fi
    
    # 检查版本
    log_info "JDK版本信息:"
    "$java_cmd" -version
    
    # 检查环境变量
    if [ -n "$JAVA_HOME" ]; then
        log_info "JAVA_HOME: $JAVA_HOME"
    else
        log_warn "JAVA_HOME未设置，请重新登录或执行: source /etc/profile"
    fi
    
    # 验证javac
    if [ -f "${JDK_INSTALL_PATH}/bin/javac" ]; then
        log_success "JDK安装验证通过（包含编译工具）"
    else
        log_success "JRE安装验证通过"
    fi
    
    return 0
}

# ==================== 主函数 ====================
main() {
    echo ""
    echo "========================================"
    echo "  JDK 安装脚本"
    echo "  节点: dw-master1 (192.168.10.10)"
    echo "  安装路径: ${JDK_INSTALL_PATH}"
    echo "  系统用户: ${SYSTEM_USER}"
    echo "========================================"
    echo ""
    
    # 备份系统Java
    backup_system_java
    
    # 安装JDK
    if ! install_jdk; then
        log_error "JDK安装失败！"
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
        log_success "JDK安装完成！"
        echo ""
        log_info "安装信息:"
        echo "  - 安装路径: ${JDK_INSTALL_PATH}"
        echo "  - Java版本: $("${JDK_INSTALL_PATH}/bin/java" -version 2>&1 | head -1)"
        echo "  - JAVA_HOME: ${JDK_INSTALL_PATH}"
        echo ""
        log_info "请执行以下命令使环境变量生效:"
        echo "  source /etc/profile"
        echo "  或者重新登录系统"
        echo ""
        exit 0
    else
        log_error "JDK安装验证失败！"
        exit 1
    fi
}

# 执行主函数
main "$@"
