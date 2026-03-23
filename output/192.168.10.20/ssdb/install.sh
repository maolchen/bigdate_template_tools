#!/bin/bash
# ============================================================
# SSDB 安装脚本 (编译安装，多实例)
# 服务: ssdb
# 节点: etl-ssdb (192.168.10.20)
# 注意: SSDB 编译安装目录固定为 /usr/local/ssdb，不能修改
# ============================================================

set -e

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ==================== 变量定义 ====================
SSDB_VERSION="ssdb_master"
INSTALL_DIR="/usr/local/ssdb"
SOURCE_PACKAGE="/data/softwares/realtime-new-doris/realtime/web_all/etl-ssdb/${SSDB_VERSION}.zip"
TEMP_DIR="/data/tmp_install_dir"
RUN_USER="bigdata"
RUN_GROUP="bigdata"

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
check_source_package() {
    if [ ! -f "$SOURCE_PACKAGE" ]; then
        log_error "SSDB安装包不存在: $SOURCE_PACKAGE"
        return 1
    fi
    log_success "安装包检查通过"
    return 0
}

# ==================== 停止所有实例 ====================
stop_all_instances() {
    log_info "停止所有SSDB实例..."
    
    # 检查是否有正在运行的进程
    SSDB_PROC_COUNT=$(pgrep -f "ssdb-server.*conf" | wc -l 2>/dev/null || echo "0")
    
    if [ "$SSDB_PROC_COUNT" -gt 0 ]; then
        log_info "发现 $SSDB_PROC_COUNT 个运行中的SSDB进程，正在停止..."
        
        # 逐个停止实例
        if pgrep -f "ssdb-server.*8892" > /dev/null 2>&1; then
            log_info "停止 SSDB 实例 (端口: 8892)..."
            pkill -f "ssdb-server.*8892" || true
        fi
        if pgrep -f "ssdb-server.*8893" > /dev/null 2>&1; then
            log_info "停止 SSDB 实例 (端口: 8893)..."
            pkill -f "ssdb-server.*8893" || true
        fi
        if pgrep -f "ssdb-server.*8894" > /dev/null 2>&1; then
            log_info "停止 SSDB 实例 (端口: 8894)..."
            pkill -f "ssdb-server.*8894" || true
        fi
        if pgrep -f "ssdb-server.*8895" > /dev/null 2>&1; then
            log_info "停止 SSDB 实例 (端口: 8895)..."
            pkill -f "ssdb-server.*8895" || true
        fi
        if pgrep -f "ssdb-server.*8897" > /dev/null 2>&1; then
            log_info "停止 SSDB 实例 (端口: 8897)..."
            pkill -f "ssdb-server.*8897" || true
        fi
        if pgrep -f "ssdb-server.*8898" > /dev/null 2>&1; then
            log_info "停止 SSDB 实例 (端口: 8898)..."
            pkill -f "ssdb-server.*8898" || true
        fi
        
        sleep 3
        
        # 强制终止残留进程
        if pgrep -f "ssdb-server" > /dev/null 2>&1; then
            log_warn "强制终止残留进程..."
            pkill -9 -f "ssdb-server" || true
            sleep 2
        fi
    fi
    
    log_success "所有实例已停止"
}

# ==================== 创建数据目录 ====================
create_data_dirs() {
    log_info "创建数据目录..."
    mkdir -p "/data/data2"
    log_info "创建目录: /data/data2"
    mkdir -p "/data/data3"
    log_info "创建目录: /data/data3"
    mkdir -p "/data/data4"
    log_info "创建目录: /data/data4"
    mkdir -p "/data/data5"
    log_info "创建目录: /data/data5"
    mkdir -p "/data/data7"
    log_info "创建目录: /data/data7"
    mkdir -p "/data/data8"
    log_info "创建目录: /data/data8"
    
    log_success "数据目录创建完成"
}

# ==================== 解压安装包 ====================
extract_package() {
    log_info "解压SSDB安装包..."
    
    # 创建临时目录
    mkdir -p "$TEMP_DIR"
    
    # 解压 zip 文件
    if command -v unzip &> /dev/null; then
        unzip -o "$SOURCE_PACKAGE" -d "$TEMP_DIR"
    else
        log_error "unzip 命令不存在，请先安装 unzip"
        exit 1
    fi
    
    log_success "解压完成: $TEMP_DIR/${SSDB_VERSION}"
}

# ==================== 编译安装 ====================
compile_install() {
    log_info "检查是否已安装SSDB..."
    
    # 检查是否已安装
    if [ -f "${INSTALL_DIR}/ssdb-server" ]; then
        log_warn "SSDB已安装，跳过编译步骤"
        return 0
    fi
    
    log_info "编译安装SSDB..."
    
    cd "$TEMP_DIR/${SSDB_VERSION}"
    
    # 编译
    log_info "执行 make..."
    make -j$(nproc)
    
    # 安装（安装到默认目录 /usr/local/ssdb）
    log_info "执行 make install..."
    make install
    
    if [ ! -f "${INSTALL_DIR}/ssdb-server" ]; then
        log_error "SSDB 安装失败，找不到 ssdb-server"
        exit 1
    fi
    
    log_success "编译安装完成: $INSTALL_DIR"
}

# ==================== 生成配置文件 ====================
generate_configs() {
    log_info "生成SSDB配置文件..."
    # ==================== 实例 8892 ====================
    cat > "${INSTALL_DIR}/ssdb_8892.conf" << 'SSDB_CONF_EOF'
# SSDB配置文件 - 端口 8892
# 节点: etl-ssdb (192.168.10.20)

# 监听地址
bind 0.0.0.0
port 8892

# 工作目录
work_dir = /data/data2
pidfile = /data/data2/ssdb.pid

# 性能参数
work_rps = 1000
work_threads = 8

# 缓存配置
cache_size = 1024
block_size = 32

# 压缩与写入
compaction_speed = 1000
write_buffer_size = 64
compression = no

# 日志
logger {
    level = debug
    output = /data/data2/ssdb.log
    rotate {
        size = 1000000000
        num = 10
    }
}

# leveldb 配置
leveldb {
    cache_size = 1024
    block_size = 32
    write_buffer_size = 64
    compaction_speed = 1000
    compression = no
}
SSDB_CONF_EOF
    log_info "配置文件已生成: ${INSTALL_DIR}/ssdb_8892.conf"
    # ==================== 实例 8893 ====================
    cat > "${INSTALL_DIR}/ssdb_8893.conf" << 'SSDB_CONF_EOF'
# SSDB配置文件 - 端口 8893
# 节点: etl-ssdb (192.168.10.20)

# 监听地址
bind 0.0.0.0
port 8893

# 工作目录
work_dir = /data/data3
pidfile = /data/data3/ssdb.pid

# 性能参数
work_rps = 1000
work_threads = 8

# 缓存配置
cache_size = 1024
block_size = 32

# 压缩与写入
compaction_speed = 1000
write_buffer_size = 64
compression = no

# 日志
logger {
    level = debug
    output = /data/data3/ssdb.log
    rotate {
        size = 1000000000
        num = 10
    }
}

# leveldb 配置
leveldb {
    cache_size = 1024
    block_size = 32
    write_buffer_size = 64
    compaction_speed = 1000
    compression = no
}
SSDB_CONF_EOF
    log_info "配置文件已生成: ${INSTALL_DIR}/ssdb_8893.conf"
    # ==================== 实例 8894 ====================
    cat > "${INSTALL_DIR}/ssdb_8894.conf" << 'SSDB_CONF_EOF'
# SSDB配置文件 - 端口 8894
# 节点: etl-ssdb (192.168.10.20)

# 监听地址
bind 0.0.0.0
port 8894

# 工作目录
work_dir = /data/data4
pidfile = /data/data4/ssdb.pid

# 性能参数
work_rps = 1000
work_threads = 8

# 缓存配置
cache_size = 1024
block_size = 32

# 压缩与写入
compaction_speed = 1000
write_buffer_size = 64
compression = no

# 日志
logger {
    level = debug
    output = /data/data4/ssdb.log
    rotate {
        size = 1000000000
        num = 10
    }
}

# leveldb 配置
leveldb {
    cache_size = 1024
    block_size = 32
    write_buffer_size = 64
    compaction_speed = 1000
    compression = no
}
SSDB_CONF_EOF
    log_info "配置文件已生成: ${INSTALL_DIR}/ssdb_8894.conf"
    # ==================== 实例 8895 ====================
    cat > "${INSTALL_DIR}/ssdb_8895.conf" << 'SSDB_CONF_EOF'
# SSDB配置文件 - 端口 8895
# 节点: etl-ssdb (192.168.10.20)

# 监听地址
bind 0.0.0.0
port 8895

# 工作目录
work_dir = /data/data5
pidfile = /data/data5/ssdb.pid

# 性能参数
work_rps = 1000
work_threads = 8

# 缓存配置
cache_size = 1024
block_size = 32

# 压缩与写入
compaction_speed = 1000
write_buffer_size = 64
compression = no

# 日志
logger {
    level = debug
    output = /data/data5/ssdb.log
    rotate {
        size = 1000000000
        num = 10
    }
}

# leveldb 配置
leveldb {
    cache_size = 1024
    block_size = 32
    write_buffer_size = 64
    compaction_speed = 1000
    compression = no
}
SSDB_CONF_EOF
    log_info "配置文件已生成: ${INSTALL_DIR}/ssdb_8895.conf"
    # ==================== 实例 8897 ====================
    cat > "${INSTALL_DIR}/ssdb_8897.conf" << 'SSDB_CONF_EOF'
# SSDB配置文件 - 端口 8897
# 节点: etl-ssdb (192.168.10.20)

# 监听地址
bind 0.0.0.0
port 8897

# 工作目录
work_dir = /data/data7
pidfile = /data/data7/ssdb.pid

# 性能参数
work_rps = 1000
work_threads = 8

# 缓存配置
cache_size = 1024
block_size = 32

# 压缩与写入
compaction_speed = 1000
write_buffer_size = 64
compression = no

# 日志
logger {
    level = debug
    output = /data/data7/ssdb.log
    rotate {
        size = 1000000000
        num = 10
    }
}

# leveldb 配置
leveldb {
    cache_size = 1024
    block_size = 32
    write_buffer_size = 64
    compaction_speed = 1000
    compression = no
}
SSDB_CONF_EOF
    log_info "配置文件已生成: ${INSTALL_DIR}/ssdb_8897.conf"
    # ==================== 实例 8898 ====================
    cat > "${INSTALL_DIR}/ssdb_8898.conf" << 'SSDB_CONF_EOF'
# SSDB配置文件 - 端口 8898
# 节点: etl-ssdb (192.168.10.20)

# 监听地址
bind 0.0.0.0
port 8898

# 工作目录
work_dir = /data/data8
pidfile = /data/data8/ssdb.pid

# 性能参数
work_rps = 1000
work_threads = 8

# 缓存配置
cache_size = 1024
block_size = 32

# 压缩与写入
compaction_speed = 1000
write_buffer_size = 64
compression = no

# 日志
logger {
    level = debug
    output = /data/data8/ssdb.log
    rotate {
        size = 1000000000
        num = 10
    }
}

# leveldb 配置
leveldb {
    cache_size = 1024
    block_size = 32
    write_buffer_size = 64
    compaction_speed = 1000
    compression = no
}
SSDB_CONF_EOF
    log_info "配置文件已生成: ${INSTALL_DIR}/ssdb_8898.conf"
    
    log_success "所有配置文件生成完成"
}

# ==================== 生成启动/停止脚本 ====================
generate_scripts() {
    log_info "生成启动和停止脚本..."
    
    # 启动脚本
    cat > "${INSTALL_DIR}/start_ssdb_all.sh" << 'START_SCRIPT_EOF'
#!/bin/bash
# SSDB 启动所有实例脚本

INSTALL_DIR="/usr/local/ssdb"
SSDB_SERVER="${INSTALL_DIR}/ssdb-server"

echo "启动所有SSDB实例..."
if pgrep -f "ssdb-server.*8892" > /dev/null 2>&1; then
    echo "实例 8892 已在运行"
else
    echo "启动实例 8892..."
    ${SSDB_SERVER} ${INSTALL_DIR}/ssdb_8892.conf -d
    sleep 1
fi
if pgrep -f "ssdb-server.*8893" > /dev/null 2>&1; then
    echo "实例 8893 已在运行"
else
    echo "启动实例 8893..."
    ${SSDB_SERVER} ${INSTALL_DIR}/ssdb_8893.conf -d
    sleep 1
fi
if pgrep -f "ssdb-server.*8894" > /dev/null 2>&1; then
    echo "实例 8894 已在运行"
else
    echo "启动实例 8894..."
    ${SSDB_SERVER} ${INSTALL_DIR}/ssdb_8894.conf -d
    sleep 1
fi
if pgrep -f "ssdb-server.*8895" > /dev/null 2>&1; then
    echo "实例 8895 已在运行"
else
    echo "启动实例 8895..."
    ${SSDB_SERVER} ${INSTALL_DIR}/ssdb_8895.conf -d
    sleep 1
fi
if pgrep -f "ssdb-server.*8897" > /dev/null 2>&1; then
    echo "实例 8897 已在运行"
else
    echo "启动实例 8897..."
    ${SSDB_SERVER} ${INSTALL_DIR}/ssdb_8897.conf -d
    sleep 1
fi
if pgrep -f "ssdb-server.*8898" > /dev/null 2>&1; then
    echo "实例 8898 已在运行"
else
    echo "启动实例 8898..."
    ${SSDB_SERVER} ${INSTALL_DIR}/ssdb_8898.conf -d
    sleep 1
fi

echo "所有实例启动完成"
START_SCRIPT_EOF
    chmod +x "${INSTALL_DIR}/start_ssdb_all.sh"
    log_info "启动脚本已生成: ${INSTALL_DIR}/start_ssdb_all.sh"
    
    # 停止脚本
    cat > "${INSTALL_DIR}/stop_ssdb_all.sh" << 'STOP_SCRIPT_EOF'
#!/bin/bash
# SSDB 停止所有实例脚本

echo "停止所有SSDB实例..."
if pgrep -f "ssdb-server.*8892" > /dev/null 2>&1; then
    echo "停止实例 8892..."
    pkill -f "ssdb-server.*8892"
fi
if pgrep -f "ssdb-server.*8893" > /dev/null 2>&1; then
    echo "停止实例 8893..."
    pkill -f "ssdb-server.*8893"
fi
if pgrep -f "ssdb-server.*8894" > /dev/null 2>&1; then
    echo "停止实例 8894..."
    pkill -f "ssdb-server.*8894"
fi
if pgrep -f "ssdb-server.*8895" > /dev/null 2>&1; then
    echo "停止实例 8895..."
    pkill -f "ssdb-server.*8895"
fi
if pgrep -f "ssdb-server.*8897" > /dev/null 2>&1; then
    echo "停止实例 8897..."
    pkill -f "ssdb-server.*8897"
fi
if pgrep -f "ssdb-server.*8898" > /dev/null 2>&1; then
    echo "停止实例 8898..."
    pkill -f "ssdb-server.*8898"
fi

sleep 2

# 强制终止残留进程
if pgrep -f "ssdb-server" > /dev/null 2>&1; then
    echo "强制终止残留进程..."
    pkill -9 -f "ssdb-server"
fi

echo "所有实例已停止"
STOP_SCRIPT_EOF
    chmod +x "${INSTALL_DIR}/stop_ssdb_all.sh"
    log_info "停止脚本已生成: ${INSTALL_DIR}/stop_ssdb_all.sh"
    
    log_success "脚本生成完成"
}

# ==================== 设置权限 ====================
set_permissions() {
    log_info "设置目录权限..."
    
    chown -R ${RUN_USER}:${RUN_GROUP} "$INSTALL_DIR"
    chown -R ${RUN_USER}:${RUN_GROUP} "/data/data2"
    chown -R ${RUN_USER}:${RUN_GROUP} "/data/data3"
    chown -R ${RUN_USER}:${RUN_GROUP} "/data/data4"
    chown -R ${RUN_USER}:${RUN_GROUP} "/data/data5"
    chown -R ${RUN_USER}:${RUN_GROUP} "/data/data7"
    chown -R ${RUN_USER}:${RUN_GROUP} "/data/data8"
    
    log_success "权限设置完成"
}

# ==================== 启动所有实例 ====================
start_all_instances() {
    log_info "启动所有SSDB实例..."
    
    # 检查是否有运行中的进程
    SSDB_PROC_COUNT=$(pgrep -f "ssdb-server.*conf" | wc -l 2>/dev/null || echo "0")
    EXPECTED_COUNT=6
    
    if [ "$SSDB_PROC_COUNT" -ge "$EXPECTED_COUNT" ]; then
        log_warn "所有实例已在运行中"
        return 0
    fi
    
    # 使用启动脚本
    sh "${INSTALL_DIR}/start_ssdb_all.sh"
    
    # 等待启动
    sleep 3
    
    # 验证端口
    if netstat -tuln 2>/dev/null | grep -q ":8892 "; then
        log_success "SSDB 实例 8892 启动成功"
    else
        log_error "SSDB 实例 8892 启动失败"
        exit 1
    fi
    if netstat -tuln 2>/dev/null | grep -q ":8893 "; then
        log_success "SSDB 实例 8893 启动成功"
    else
        log_error "SSDB 实例 8893 启动失败"
        exit 1
    fi
    if netstat -tuln 2>/dev/null | grep -q ":8894 "; then
        log_success "SSDB 实例 8894 启动成功"
    else
        log_error "SSDB 实例 8894 启动失败"
        exit 1
    fi
    if netstat -tuln 2>/dev/null | grep -q ":8895 "; then
        log_success "SSDB 实例 8895 启动成功"
    else
        log_error "SSDB 实例 8895 启动失败"
        exit 1
    fi
    if netstat -tuln 2>/dev/null | grep -q ":8897 "; then
        log_success "SSDB 实例 8897 启动成功"
    else
        log_error "SSDB 实例 8897 启动失败"
        exit 1
    fi
    if netstat -tuln 2>/dev/null | grep -q ":8898 "; then
        log_success "SSDB 实例 8898 启动成功"
    else
        log_error "SSDB 实例 8898 启动失败"
        exit 1
    fi
    
    log_success "所有实例启动完成"
}

# ==================== 添加开机启动 ====================
add_startup_scripts() {
    log_info "配置开机启动..."
    
    # 确保 rc.local 可执行
    if [ -f /etc/rc.local ]; then
        chmod +x /etc/rc.local
    fi
    
    START_CMD="su - ${RUN_USER} -c 'sh ${INSTALL_DIR}/start_ssdb_all.sh'"
    
    if ! grep -q "start_ssdb_all.sh" /etc/rc.local 2>/dev/null; then
        if grep -q "^exit 0" /etc/rc.local 2>/dev/null; then
            sed -i "/^exit 0/i ${START_CMD}" /etc/rc.local
        else
            echo "${START_CMD}" >> /etc/rc.local
        fi
        log_info "已添加到开机启动"
    fi
    
    log_success "开机启动配置完成"
}

# ==================== 显示状态 ====================
show_status() {
    echo ""
    echo "============================================"
    echo "SSDB 安装完成"
    echo "============================================"
    echo "版本: ${SSDB_VERSION}"
    echo "安装目录: ${INSTALL_DIR}"
    echo ""
    echo "实例列表:"
    echo "  - 端口: 8892"
    echo "    配置: ${INSTALL_DIR}/ssdb_8892.conf"
    echo "    数据: /data/data2"
    echo ""
    echo "  - 端口: 8893"
    echo "    配置: ${INSTALL_DIR}/ssdb_8893.conf"
    echo "    数据: /data/data3"
    echo ""
    echo "  - 端口: 8894"
    echo "    配置: ${INSTALL_DIR}/ssdb_8894.conf"
    echo "    数据: /data/data4"
    echo ""
    echo "  - 端口: 8895"
    echo "    配置: ${INSTALL_DIR}/ssdb_8895.conf"
    echo "    数据: /data/data5"
    echo ""
    echo "  - 端口: 8897"
    echo "    配置: ${INSTALL_DIR}/ssdb_8897.conf"
    echo "    数据: /data/data7"
    echo ""
    echo "  - 端口: 8898"
    echo "    配置: ${INSTALL_DIR}/ssdb_8898.conf"
    echo "    数据: /data/data8"
    echo ""
    echo "管理命令:"
    echo "  启动所有: sh ${INSTALL_DIR}/start_ssdb_all.sh"
    echo "  停止所有: sh ${INSTALL_DIR}/stop_ssdb_all.sh"
    echo "  单实例: ${INSTALL_DIR}/ssdb-server <config_file> -d"
    echo "  客户端: ${INSTALL_DIR}/ssdb-cli -p <port>"
    echo "============================================"
}

# ==================== 主函数 ====================
main() {
    log_info "开始安装 SSDB ${SSDB_VERSION}..."
    echo ""
    
    # 检查安装包
    check_source_package || exit 1
    
    # 停止旧实例
    stop_all_instances
    
    # 创建数据目录
    create_data_dirs
    
    # 解压
    extract_package
    
    # 编译安装
    compile_install
    
    # 生成配置
    generate_configs
    
    # 生成脚本
    generate_scripts
    
    # 设置权限
    set_permissions
    
    # 启动实例
    start_all_instances
    
    # 添加开机启动
    add_startup_scripts
    
    # 显示状态
    show_status
    
    log_success "SSDB 安装完成！"
}

# 执行主函数
main "$@"
