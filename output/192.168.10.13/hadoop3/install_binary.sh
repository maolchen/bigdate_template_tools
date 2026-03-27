#!/bin/bash
# Hadoop 3 二进制包安装脚本
# 使用方法: bash install_binary.sh [--force]
# 说明: 在所有Hadoop节点执行，安装Hadoop二进制包
# 参考: tmp/hadoop3/tasks/02_install_hadoop.yml

set -e

# ============================================================
# 全局变量（遵循 GLOBAL_VARS_GUIDE.md 规范）
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
SOFTWARE_DIR="/data/softwares"
PKG_PRO_DIR="/data/20251027-yuwq-cdtest"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

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

# ============================================================
# Hadoop 配置
# ============================================================
HADOOP_VERSION="3.3.6"
HADOOP_INSTALL_DIR="${INSTALL_BASE_DIR}/hadoop"
HADOOP_PKG_NAME="hadoop-3.3.6.tar.gz"
PACKAGE_SUBDIR="realtime-new-doris/realtime/hadoop3"
SOURCE_PACKAGE="${PKG_PRO_DIR}/${PACKAGE_SUBDIR}/${HADOOP_PKG_NAME}"

# ============================================================
# 参数解析
# ============================================================
FORCE_INSTALL=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE_INSTALL=true
            shift
            ;;
        *)
            log_error "未知参数: $1"
            echo "使用方法: bash install_binary.sh [--force]"
            exit 1
            ;;
    esac
done

# ============================================================
# 开始安装
# ============================================================
echo "============================================"
echo "Hadoop 3 二进制包安装脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "Hadoop版本: ${HADOOP_VERSION}"
echo "安装目录: ${HADOOP_INSTALL_DIR}"
echo "安装包: ${SOURCE_PACKAGE}"
echo "运行用户: ${RUN_USER}"
echo "强制安装: ${FORCE_INSTALL}"
echo "============================================"

# ============================================================
# 步骤1: 检查安装包
# ============================================================
log_info "步骤1: 检查安装包..."
if [ ! -f "${SOURCE_PACKAGE}" ]; then
    log_error "安装包不存在: ${SOURCE_PACKAGE}"
    exit 1
fi
log_info "安装包检查通过: ${SOURCE_PACKAGE}"

# ============================================================
# 步骤2: 检查是否已安装
# ============================================================
log_info "步骤2: 检查安装状态..."
if [ -d "${HADOOP_INSTALL_DIR}/etc" ]; then
    if [ "$FORCE_INSTALL" = "true" ]; then
        log_warn "检测到已有安装，强制模式：备份后重新安装"
        BACKUP_DIR="${HADOOP_INSTALL_DIR}.bak.$(date '+%Y%m%d%H%M%S')"
        run_as_root mv "${HADOOP_INSTALL_DIR}" "${BACKUP_DIR}"
        log_info "已备份到: ${BACKUP_DIR}"
    else
        log_info "Hadoop已安装，跳过安装步骤"
        log_info "如需重新安装，请使用 --force 参数"
        exit 0
    fi
fi

# ============================================================
# 步骤3: 创建安装目录
# ============================================================
log_info "步骤3: 创建安装目录..."
run_as_root mkdir -p "${HADOOP_INSTALL_DIR}"
log_info "安装目录创建完成"

# ============================================================
# 步骤4: 解压安装包
# ============================================================
log_info "步骤4: 解压安装包..."
run_as_root tar -zxf "${SOURCE_PACKAGE}" -C "${HADOOP_INSTALL_DIR}" --strip-components=1
log_info "安装包解压完成"

# ============================================================
# 步骤5: 配置环境变量（遵循规范4）
# ============================================================
log_info "步骤5: 配置环境变量..."

ENV_CONTENT="# HADOOP3
export HADOOP_HOME=${HADOOP_INSTALL_DIR}
export HADOOP_CONF_DIR=\$HADOOP_HOME/etc/hadoop
export LD_LIBRARY_PATH=${HADOOP_INSTALL_DIR}/lib/native
export HADOOP_CLASSPATH=\$(${HADOOP_INSTALL_DIR}/bin/hadoop classpath)
export PATH=\$PATH:\$HADOOP_HOME/bin
export PATH=\$PATH:\$HADOOP_HOME/sbin"

if [ "$RUN_USER" = "root" ]; then
    # root用户：配置到 /etc/profile
    if ! grep -q "HADOOP_HOME=${HADOOP_INSTALL_DIR}" /etc/profile 2>/dev/null; then
        echo "" >> /etc/profile
        echo "$ENV_CONTENT" >> /etc/profile
        log_info "环境变量已添加到 /etc/profile"
    else
        log_info "环境变量已存在于 /etc/profile"
    fi
else
    # 非root用户：配置到 ~/.bash_profile 和 /etc/profile
    if ! grep -q "HADOOP_HOME=${HADOOP_INSTALL_DIR}" ~/.bash_profile 2>/dev/null; then
        echo "" >> ~/.bash_profile
        echo "$ENV_CONTENT" >> ~/.bash_profile
        log_info "环境变量已添加到 ~/.bash_profile"
    else
        log_info "环境变量已存在于 ~/.bash_profile"
    fi
    
    # 同时添加到 /etc/profile
    if ! run_as_root grep -q "HADOOP_HOME=${HADOOP_INSTALL_DIR}" /etc/profile 2>/dev/null; then
        run_as_root sh -c "echo '' >> /etc/profile"
        run_as_root sh -c "echo '$ENV_CONTENT' >> /etc/profile"
        log_info "环境变量已添加到 /etc/profile"
    fi
fi

# ============================================================
# 步骤6: 设置权限（遵循规范6）
# ============================================================
log_info "步骤6: 设置目录权限..."
run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${HADOOP_INSTALL_DIR}"
run_as_root chmod -R 755 "${HADOOP_INSTALL_DIR}"
log_info "权限设置完成"

# ============================================================
# 步骤7: 验证安装
# ============================================================
log_info "步骤7: 验证安装..."
if [ -f "${HADOOP_INSTALL_DIR}/bin/hadoop" ]; then
    log_info "Hadoop二进制文件存在"
    
    export JAVA_HOME=${JAVA_HOME}
    export HADOOP_HOME=${HADOOP_INSTALL_DIR}
    VERSION_INFO=$(${HADOOP_INSTALL_DIR}/bin/hadoop version 2>/dev/null | head -1 || echo "无法获取版本")
    log_info "版本信息: ${VERSION_INFO}"
else
    log_error "Hadoop二进制文件不存在，安装可能失败"
    exit 1
fi

# ============================================================
# 安装完成
# ============================================================
echo ""
echo "============================================"
echo "Hadoop 3 二进制包安装完成！"
echo "============================================"
echo "安装目录: ${HADOOP_INSTALL_DIR}"
echo "配置目录: ${HADOOP_INSTALL_DIR}/etc/hadoop"
echo "运行用户: ${RUN_USER}"
echo ""
echo "后续步骤:"
echo "1. 部署配置文件: bash deploy_config.sh"
echo "2. 创建数据目录: bash setup_dirs.sh"
echo "3. 按正确顺序启动服务"
echo ""
echo "请执行 source /etc/profile 或重新登录使环境变量生效"
