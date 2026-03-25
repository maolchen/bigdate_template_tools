#!/bin/bash
# ============================================================
# Kafka 安装脚本 (包含内置 Zookeeper)
# 服务: kafka
# 节点: realtime-kafka2 (192.168.10.14)
# Broker ID: 2
# ============================================================

set -e

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ==================== 变量定义 ====================
KAFKA_VERSION="2.11-0.11.0.2"
SCALA_VERSION="2.11"
INSTALL_DIR="/data/localization/kafka"
SOURCE_PACKAGE="/data/softwares/realtime-new-doris/realtime/kafka/kafka_${KAFKA_VERSION}.tgz"
TEMP_DIR="/data/tmp_install_dir"
BROKER_ID="2"
ZK_MYID="2"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

# 全局目录
DATA_BASE_DIR="/data"
INSTALL_BASE_DIR="/data/localization"

# Zookeeper 配置
ZK_DATA_DIR="${DATA_BASE_DIR}/zk-kafka-data"
ZK_CLIENT_PORT="2182"
ZK_PEER_PORT="2889"
ZK_ELECTION_PORT="3889"

# Kafka Server 配置
KAFKA_PORT="9092"
KAFKA_LOG_DIRS="${DATA_BASE_DIR}/kafka-data"

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
        log_error "Kafka安装包不存在: $SOURCE_PACKAGE"
        return 1
    fi
    log_success "安装包检查通过"
    return 0
}

# ==================== 停止服务 ====================
stop_services() {
    log_info "停止 Kafka 和 Zookeeper 服务..."
    
    # 停止 Kafka
    if systemctl is-active kafka > /dev/null 2>&1; then
        log_info "停止 Kafka 服务..."
        systemctl stop kafka || true
    fi
    
    # 停止 Zookeeper
    if systemctl is-active zookeeper > /dev/null 2>&1; then
        log_info "停止 Zookeeper 服务..."
        systemctl stop zookeeper || true
    fi
    
    # 强制终止残留进程
    if pgrep -f "kafka.Kafka" > /dev/null 2>&1; then
        log_warn "强制终止 Kafka 进程..."
        pkill -9 -f "kafka.Kafka" || true
    fi
    
    if pgrep -f "org.apache.zookeeper.server.quorum.QuorumPeerMain" > /dev/null 2>&1; then
        log_warn "强制终止 Zookeeper 进程..."
        pkill -9 -f "org.apache.zookeeper.server.quorum.QuorumPeerMain" || true
    fi
    
    sleep 3
    log_success "服务已停止"
}

# ==================== 清理旧安装 ====================
clean_old_install() {
    log_info "清理旧安装目录..."
    
    if [ -d "$INSTALL_DIR" ]; then
        log_warn "删除旧安装目录: $INSTALL_DIR"
        rm -rf "$INSTALL_DIR"
    fi
    
    if [ -d "$ZK_DATA_DIR" ]; then
        log_warn "删除旧ZK数据目录: $ZK_DATA_DIR"
        rm -rf "$ZK_DATA_DIR"
    fi
    
    if [ -d "$KAFKA_LOG_DIRS" ]; then
        log_warn "删除旧Kafka日志目录: $KAFKA_LOG_DIRS"
        rm -rf "$KAFKA_LOG_DIRS"
    fi
    
    log_success "清理完成"
}

# ==================== 创建目录 ====================
create_directories() {
    log_info "创建所需目录..."
    
    mkdir -p "$ZK_DATA_DIR"
    mkdir -p "$KAFKA_LOG_DIRS"
    mkdir -p "$TEMP_DIR"
    mkdir -p "$(dirname $INSTALL_DIR)"
    
    log_success "目录创建完成"
}

# ==================== 解压安装包 ====================
extract_package() {
    log_info "解压Kafka安装包..."
    
    tar -xzf "$SOURCE_PACKAGE" -C "$TEMP_DIR"
    
    # 移动到目标位置
    mv "$TEMP_DIR/kafka_${SCALA_VERSION}-${KAFKA_VERSION#*-}" "$INSTALL_DIR"
    
    log_success "解压完成: $INSTALL_DIR"
}

# ==================== 设置 broker.id 和 myid ====================
set_ids() {
    log_info "设置 Broker ID 和 ZK MyID..."
    
    # 设置 broker.id
    sed -i "s/^broker.id=.*/broker.id=${BROKER_ID}/" "${INSTALL_DIR}/config/server.properties"
    
    # 设置 myid
    echo "$ZK_MYID" > "${ZK_DATA_DIR}/myid"
    
    log_success "Broker ID: ${BROKER_ID}, MyID: ${ZK_MYID}"
}

# ==================== 生成 Zookeeper 配置 ====================
generate_zookeeper_config() {
    log_info "生成 Zookeeper 配置文件..."
    
    cat > "${INSTALL_DIR}/config/zookeeper.properties" << 'ZK_CONF_EOF'
# Zookeeper 配置文件
# 节点: realtime-kafka2 (192.168.10.14)

# 数据目录
dataDir=/data/zk-kafka-data

# 客户端端口
clientPort=2182

# 连接限制
maxClientCnxns=0

# 集群配置
initLimit=5
syncLimit=2

# 集群节点列表
server.1=192.168.10.13:2889:3889
server.2=192.168.10.14:2889:3889
server.3=192.168.10.15:2889:3889
ZK_CONF_EOF
    
    log_success "Zookeeper 配置已生成"
}

# ==================== 生成 Kafka Server 配置 ====================
generate_server_config() {
    log_info "生成 Kafka Server 配置文件..."
    
    # 获取所有 Kafka 节点的 ZK 连接串
    ZK_CONNECT=""
    
    cat > "${INSTALL_DIR}/config/server.properties" << 'SERVER_CONF_EOF'
# Kafka Server 配置文件
# 节点: realtime-kafka2 (192.168.10.14)
# Broker ID: 2

############################# Server Basics #############################
broker.id=2
delete.topic.enable=true

############################# Socket Server Settings #############################
listeners=PLAINTEXT://192.168.10.14:9092
advertised.listeners=PLAINTEXT://192.168.10.14:9092
num.network.threads=3
num.io.threads=8
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600

############################# Log Basics #############################
log.dirs=/data/kafka-data
num.partitions=5
num.recovery.threads.per.data.dir=1

############################# Internal Topic Settings #############################
offsets.topic.replication.factor=1
transaction.state.log.replication.factor=1
transaction.state.log.min.isr=1

############################# Log Retention Policy #############################
log.retention.hours=168
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000

############################# Zookeeper #############################
zookeeper.connect=
zookeeper.connection.timeout.ms=6000

############################# Group Coordinator Settings #############################
group.initial.rebalance.delay.ms=0
SERVER_CONF_EOF
    
    log_success "Kafka Server 配置已生成"
}

# ==================== 生成 Consumer 配置 ====================
generate_consumer_config() {
    log_info "生成 Consumer 配置文件..."
    
    cat > "${INSTALL_DIR}/config/consumer.properties" << 'CONSUMER_CONF_EOF'
# Consumer 配置文件
zookeeper.connect=
zookeeper.connection.timeout.ms=6000
group.id=test-consumer-group
CONSUMER_CONF_EOF
    
    log_success "Consumer 配置已生成"
}

# ==================== 生成 Producer 配置 ====================
generate_producer_config() {
    log_info "生成 Producer 配置文件..."
    
    cat > "${INSTALL_DIR}/config/producer.properties" << 'PRODUCER_CONF_EOF'
# Producer 配置文件
acks=all
retries=3
batch.size=16384
buffer.memory=33554432
bootstrap.servers=
PRODUCER_CONF_EOF
    
    log_success "Producer 配置已生成"
}

# ==================== 生成启动/停止脚本 ====================
generate_scripts() {
    log_info "生成启动和停止脚本..."
    
    # Zookeeper 启动脚本
    cat > "${INSTALL_DIR}/start_zk.sh" << 'START_ZK_EOF'
#!/bin/bash
cd <no value>
nohup bin/zookeeper-server-start.sh config/zookeeper.properties > zk.log &
START_ZK_EOF
    chmod +x "${INSTALL_DIR}/start_zk.sh"
    
    # Zookeeper 停止脚本
    cat > "${INSTALL_DIR}/stop_zk.sh" << 'STOP_ZK_EOF'
#!/bin/bash
ps aux | grep -v grep | grep "zookeeper.properties" | awk '{print $2}' | xargs kill -9 2>/dev/null || true
STOP_ZK_EOF
    chmod +x "${INSTALL_DIR}/stop_zk.sh"
    
    # Kafka 启动脚本
    cat > "${INSTALL_DIR}/start_kafka.sh" << 'START_KAFKA_EOF'
#!/bin/bash
cd <no value>
nohup bin/kafka-server-start.sh config/server.properties > kafka.log &
START_KAFKA_EOF
    chmod +x "${INSTALL_DIR}/start_kafka.sh"
    
    # Kafka 停止脚本
    cat > "${INSTALL_DIR}/stop_kafka.sh" << 'STOP_KAFKA_EOF'
#!/bin/bash
ps aux | grep -v grep | grep "server.properties" | awk '{print $2}' | xargs kill -9 2>/dev/null || true
STOP_KAFKA_EOF
    chmod +x "${INSTALL_DIR}/stop_kafka.sh"
    
    # 状态检查脚本
    cat > "${INSTALL_DIR}/get_check.sh" << 'GET_CHECK_EOF'
#!/bin/bash
echo "=== Zookeeper Status ==="
if pgrep -f "zookeeper.properties" > /dev/null; then
    echo "Zookeeper: Running"
    netstat -tuln | grep 2182 || true
else
    echo "Zookeeper: Stopped"
fi

echo ""
echo "=== Kafka Status ==="
if pgrep -f "kafka.Kafka" > /dev/null; then
    echo "Kafka: Running"
    netstat -tuln | grep 9092 || true
else
    echo "Kafka: Stopped"
fi
GET_CHECK_EOF
    chmod +x "${INSTALL_DIR}/get_check.sh"
    
    log_success "脚本生成完成"
}

# ==================== 安装 Systemd 服务 ====================
install_systemd_services() {
    log_info "安装 Systemd 服务文件..."
    
    # Zookeeper Service
    cat > /etc/systemd/system/zookeeper.service << 'ZK_SERVICE_EOF'
[Unit]
Description=Apache Zookeeper server (Kafka embedded)
Documentation=http://zookeeper.apache.org
Requires=network.target remote-fs.target
After=network.target remote-fs.target

[Service]
Type=simple
Environment="JAVA_HOME=<no value>"
Environment="PATH=<no value>/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
User=<no value>
Group=<no value>
ExecStart=<no value>/bin/zookeeper-server-start.sh <no value>/config/zookeeper.properties
ExecStop=<no value>/bin/zookeeper-server-stop.sh
TimeoutSec=30
Restart=on-failure

[Install]
WantedBy=multi-user.target
ZK_SERVICE_EOF
    
    # Kafka Service
    cat > /etc/systemd/system/kafka.service << 'KAFKA_SERVICE_EOF'
[Unit]
Description=Apache Kafka Server (broker)
Documentation=http://kafka.apache.org/documentation.html
Requires=zookeeper.service
After=zookeeper.service

[Service]
Type=simple
Environment="JAVA_HOME=<no value>"
Environment="PATH=<no value>/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
User=<no value>
Group=<no value>
ExecStart=<no value>/bin/kafka-server-start.sh <no value>/config/server.properties
ExecStop=<no value>/bin/kafka-server-stop.sh
Restart=on-failure

[Install]
WantedBy=multi-user.target
KAFKA_SERVICE_EOF
    
    # 重载 systemd
    systemctl daemon-reload
    
    log_success "Systemd 服务安装完成"
}

# ==================== 设置权限 ====================
set_permissions() {
    log_info "设置目录权限..."
    
    chown -R ${RUN_USER}:${RUN_GROUP} "$INSTALL_DIR"
    chown -R ${RUN_USER}:${RUN_GROUP} "$ZK_DATA_DIR"
    chown -R ${RUN_USER}:${RUN_GROUP} "$KAFKA_LOG_DIRS"
    
    log_success "权限设置完成"
}

# ==================== 启动服务 ====================
start_services() {
    log_info "启动服务..."
    
    # 启动 Zookeeper
    log_info "启动 Zookeeper..."
    systemctl enable zookeeper
    systemctl start zookeeper
    
    # 等待 Zookeeper 启动
    sleep 5
    if ! systemctl is-active zookeeper > /dev/null; then
        log_error "Zookeeper 启动失败"
        journalctl -u zookeeper -n 20
        exit 1
    fi
    log_success "Zookeeper 启动成功"
    
    # 等待 Zookeeper 端口
    for i in {1..30}; do
        if netstat -tuln 2>/dev/null | grep -q ":${ZK_CLIENT_PORT} "; then
            break
        fi
        sleep 1
    done
    
    # 启动 Kafka
    log_info "启动 Kafka..."
    systemctl enable kafka
    systemctl start kafka
    
    # 等待 Kafka 启动
    sleep 5
    if ! systemctl is-active kafka > /dev/null; then
        log_error "Kafka 启动失败"
        journalctl -u kafka -n 20
        exit 1
    fi
    log_success "Kafka 启动成功"
    
    # 等待 Kafka 端口
    for i in {1..30}; do
        if netstat -tuln 2>/dev/null | grep -q ":${KAFKA_PORT} "; then
            break
        fi
        sleep 1
    done
}

# ==================== 显示状态 ====================
show_status() {
    echo ""
    echo "============================================"
    echo "Kafka 安装完成"
    echo "============================================"
    echo "版本: kafka_${SCALA_VERSION}-${KAFKA_VERSION#*-}"
    echo "安装目录: ${INSTALL_DIR}"
    echo "Broker ID: ${BROKER_ID}"
    echo "ZK MyID: ${ZK_MYID}"
    echo ""
    echo "服务状态:"
    systemctl status zookeeper --no-pager | head -3
    systemctl status kafka --no-pager | head -3
    echo ""
    echo "端口:"
    echo "  Zookeeper: ${ZK_CLIENT_PORT}"
    echo "  Kafka: ${KAFKA_PORT}"
    echo ""
    echo "管理命令:"
    echo "  启动ZK: systemctl start zookeeper"
    echo "  停止ZK: systemctl stop zookeeper"
    echo "  启动Kafka: systemctl start kafka"
    echo "  停止Kafka: systemctl stop kafka"
    echo "  查看Topic: ${INSTALL_DIR}/bin/kafka-topics.sh --zookeeper localhost:${ZK_CLIENT_PORT} --list"
    echo "============================================"
}

# ==================== 主函数 ====================
main() {
    log_info "开始安装 Kafka..."
    echo ""
    
    # 检查安装包
    check_source_package || exit 1
    
    # 停止服务
    stop_services
    
    # 清理旧安装
    clean_old_install
    
    # 创建目录
    create_directories
    
    # 解压
    extract_package
    
    # 生成配置
    generate_zookeeper_config
    generate_server_config
    generate_consumer_config
    generate_producer_config
    
    # 设置 ID
    set_ids
    
    # 生成脚本
    generate_scripts
    
    # 安装 systemd 服务
    install_systemd_services
    
    # 设置权限
    set_permissions
    
    # 启动服务
    start_services
    
    # 显示状态
    show_status
    
    log_success "Kafka 安装完成！"
}

# 执行主函数
main "$@"
