#!/bin/bash
# ============================================================
# Kafka 安装脚本
# 主机: node4 (192.168.10.13)
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
INSTALL_DIR="${INSTALL_BASE}/kafka"
LOG_DIRS="${INSTALL_BASE}/kafka/logs"
BROKER_ID="1"
BROKER_PORT="9092"
HOSTNAME="node4"
IP="192.168.10.13"

log_info "=========================================="
log_info "开始安装 Kafka"
log_info "主机: ${HOSTNAME} (${IP})"
log_info "Broker ID: ${BROKER_ID}"
log_info "=========================================="

# ============================================================
# 环境检查
# ============================================================
log_info "检查Java环境..."
if ! command -v java &> /dev/null; then
    log_error "Java未安装，请先安装JDK"
    exit 1
fi

# ============================================================
# 创建目录
# ============================================================
log_info "创建目录结构..."
mkdir -p "${LOG_DIRS}"

# ============================================================
# 复制配置文件
# ============================================================
log_info "复制配置文件..."
if [ -f "server.properties" ]; then
    cp server.properties "${INSTALL_DIR}/config/server.properties"
    log_info "server.properties 已复制到 ${INSTALL_DIR}/config/"
else
    log_error "找不到 server.properties 文件"
    exit 1
fi

# ============================================================
# 创建启动脚本
# ============================================================
log_info "创建启动脚本..."
cat > "${INSTALL_DIR}/start.sh" << START_EOF
#!/bin/bash
export KAFKA_HEAP_OPTS="-Xmx2G -Xms2G"
cd ${INSTALL_DIR}
./bin/kafka-server-start.sh -daemon config/server.properties
START_EOF
chmod +x "${INSTALL_DIR}/start.sh"

# ============================================================
# 创建停止脚本
# ============================================================
log_info "创建停止脚本..."
cat > "${INSTALL_DIR}/stop.sh" << STOP_EOF
#!/bin/bash
cd ${INSTALL_DIR}
./bin/kafka-server-stop.sh
STOP_EOF
chmod +x "${INSTALL_DIR}/stop.sh"

log_info "=========================================="
log_info "Kafka 安装完成！"
log_info "启动命令: ${INSTALL_DIR}/start.sh"
log_info "停止命令: ${INSTALL_DIR}/stop.sh"
log_info "=========================================="
