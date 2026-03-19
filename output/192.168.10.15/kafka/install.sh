#!/bin/bash
# ============================================================
# Kafka 安装脚本
# 节点: realtime-kafka3
# IP: 192.168.10.15
# broker.id: 3
# ============================================================

set -e

echo "=========================================="
echo "安装 Kafka"
echo "节点: realtime-kafka3"
echo "IP: 192.168.10.15"
echo "broker.id: 3"
echo "=========================================="

# 变量定义
KAFKA_VERSION="3.5.1"
SCALA_VERSION="2.13"
KAFKA_INSTALL_DIR="/data/localization/kafka"
KAFKA_DATA_DIR="/data/bigdata/kafka/logs"
KAFKA_LOG_DIR="/var/log/bigdata/kafka"
KAFKA_USER="bigdata"
KAFKA_TAR="/data/tmp_install_dir/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz"

# 检查安装包
if [ ! -f "$KAFKA_TAR" ]; then
    echo "错误: 安装包不存在: $KAFKA_TAR"
    exit 1
fi

# 创建目录
mkdir -p "$KAFKA_INSTALL_DIR"
mkdir -p "$KAFKA_DATA_DIR"
mkdir -p "$KAFKA_LOG_DIR"

# 解压安装
echo "解压 Kafka..."
cd "/data/localization"
tar -xzf "$KAFKA_TAR"
mv kafka_${SCALA_VERSION}-${KAFKA_VERSION} kafka

# 复制配置文件
echo "复制配置文件..."
cp realtime-kafka3_server.properties "$KAFKA_INSTALL_DIR/config/server.properties"

# 设置权限
chown -R $KAFKA_USER:$KAFKA_USER "$KAFKA_INSTALL_DIR"
chown -R $KAFKA_USER:$KAFKA_USER "$KAFKA_DATA_DIR"
chown -R $KAFKA_USER:$KAFKA_USER "$KAFKA_LOG_DIR"

# 创建启动脚本
echo "创建启动脚本..."
cat > "$KAFKA_INSTALL_DIR/bin/start.sh" << 'EOF'
#!/bin/bash
export JAVA_HOME=/data/jdk/jdk1.8.0_65
export KAFKA_HEAP_OPTS="-Xmx2G -Xms2G"
export KAFKA_LOG_DIR=/var/log/bigdata/kafka
cd /data/localization/kafka
./bin/kafka-server-start.sh -daemon config/server.properties
EOF
chmod +x "$KAFKA_INSTALL_DIR/bin/start.sh"

# 创建停止脚本
cat > "$KAFKA_INSTALL_DIR/bin/stop.sh" << 'EOF'
#!/bin/bash
cd /data/localization/kafka
./bin/kafka-server-stop.sh
EOF
chmod +x "$KAFKA_INSTALL_DIR/bin/stop.sh"

echo "=========================================="
echo "Kafka 安装完成"
echo "安装目录: $KAFKA_INSTALL_DIR"
echo "数据目录: $KAFKA_DATA_DIR"
echo "broker.id: 3"
echo "=========================================="
