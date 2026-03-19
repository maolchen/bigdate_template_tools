#!/bin/bash
# ============================================================
# Zookeeper 安装脚本
# 主机: node1 (192.168.10.10)
# ============================================================

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ============================================================
# 配置变量
# ============================================================
INSTALL_BASE="/data/localization"
INSTALL_DIR="${INSTALL_BASE}/zookeeper"
DATA_DIR="${INSTALL_BASE}/zookeeper/data"
LOG_DIR="${INSTALL_BASE}/zookeeper/logs"
MYID="1"
HOSTNAME="node1"
IP="192.168.10.10"

log_info "=========================================="
log_info "开始安装 Zookeeper"
log_info "主机: ${HOSTNAME} (${IP})"
log_info "MyID: ${MYID}"
log_info "=========================================="

# ============================================================
# 环境检查
# ============================================================
log_info "检查Java环境..."
if ! command -v java &> /dev/null; then
    log_warn "Java未安装，请先安装JDK"
    log_warn "JDK路径: /data/jdk"
fi

# ============================================================
# 创建目录
# ============================================================
log_info "创建目录结构..."
mkdir -p "${DATA_DIR}"
mkdir -p "${LOG_DIR}"

# ============================================================
# 写入myid
# ============================================================
log_info "配置 myid..."
echo "${MYID}" > "${DATA_DIR}/myid"

# ============================================================
# 复制配置文件
# ============================================================
log_info "复制配置文件..."
if [ -f "zoo.cfg" ]; then
    cp zoo.cfg "${INSTALL_DIR}/conf/zoo.cfg"
    log_info "zoo.cfg 已复制到 ${INSTALL_DIR}/conf/"
else
    log_error "找不到 zoo.cfg 文件"
    exit 1
fi

# ============================================================
# 创建启动脚本
# ============================================================
log_info "创建启动脚本..."
cat > "${INSTALL_DIR}/start.sh" << START_EOF
#!/bin/bash
cd ${INSTALL_DIR}
./bin/zkServer.sh start
START_EOF
chmod +x "${INSTALL_DIR}/start.sh"

# ============================================================
# 创建停止脚本
# ============================================================
log_info "创建停止脚本..."
cat > "${INSTALL_DIR}/stop.sh" << STOP_EOF
#!/bin/bash
cd ${INSTALL_DIR}
./bin/zkServer.sh stop
STOP_EOF
chmod +x "${INSTALL_DIR}/stop.sh"

log_info "=========================================="
log_info "Zookeeper 安装完成！"
log_info "启动命令: ${INSTALL_DIR}/start.sh"
log_info "停止命令: ${INSTALL_DIR}/stop.sh"
log_info "=========================================="
