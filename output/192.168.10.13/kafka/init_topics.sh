#!/bin/bash
# ============================================================
# Kafka Topic 初始化脚本
# 服务: kafka
# 节点: realtime-kafka1 (192.168.10.13)
# ============================================================

set -e

# ==================== 变量定义 ====================
INSTALL_DIR="/data/localization/kafka"
KAFKA_VERSION="2.11-0.11.0.2"
ZK_PORT="2182"
KAFKA_PORT="9092"
JAVA_HOME="/data/jdk/"

export JAVA_HOME=$JAVA_HOME
export PATH=$JAVA_HOME/bin:$PATH

# ==================== 检测 Kafka 版本 ====================
# 3.x 及以上版本使用 --bootstrap-server
# 旧版本使用 --zookeeper
USE_BOOTSTRAP_SERVER=false
if [[ "$KAFKA_VERSION" == *"3."* ]]; then
    USE_BOOTSTRAP_SERVER=true
fi

# ==================== 获取 Topic 列表 ====================
get_topic_list() {
    if [ "$USE_BOOTSTRAP_SERVER" = true ]; then
        ${INSTALL_DIR}/bin/kafka-topics.sh --bootstrap-server localhost:${KAFKA_PORT} --list 2>/dev/null
    else
        ${INSTALL_DIR}/bin/kafka-topics.sh --zookeeper localhost:${ZK_PORT} --list 2>/dev/null
    fi
}

# ==================== 创建 Topic ====================
create_topic() {
    local topic_name=$1
    local partitions=$2
    local replication_factor=$3
    
    echo "创建 Topic: ${topic_name} (partitions=${partitions}, replication=${replication_factor})"
    
    if [ "$USE_BOOTSTRAP_SERVER" = true ]; then
        ${INSTALL_DIR}/bin/kafka-topics.sh --bootstrap-server localhost:${KAFKA_PORT} \
            --create --topic ${topic_name} \
            --partitions ${partitions} \
            --replication-factor ${replication_factor} 2>/dev/null || echo "Topic ${topic_name} 已存在"
    else
        ${INSTALL_DIR}/bin/kafka-topics.sh --zookeeper localhost:${ZK_PORT} \
            --create --topic ${topic_name} \
            --partitions ${partitions} \
            --replication-factor ${replication_factor} 2>/dev/null || echo "Topic ${topic_name} 已存在"
    fi
}

# ==================== 设置 Consumer Offset ====================
set_consumer_offset() {
    local consumer_config=$1
    local topic=$2
    local offset=${3:-earliest}
    
    echo "设置 Consumer Offset: ${consumer_config} -> ${topic} (${offset})"
    
    ${INSTALL_DIR}/bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK ${offset} ${consumer_config} ${topic} 2>/dev/null || true
}

# ==================== 主函数 ====================
main() {
    echo "============================================"
    echo "Kafka Topic 初始化"
    echo "============================================"
    echo "Kafka 版本: ${KAFKA_VERSION}"
    echo "使用 Bootstrap Server: ${USE_BOOTSTRAP_SERVER}"
    echo ""
    
    cd ${INSTALL_DIR}
    
    # 检查当前 Topic 列表
    echo ">>> 检查现有 Topic..."
    CURRENT_TOPICS=$(get_topic_list)
    TOPIC_COUNT=$(echo "$CURRENT_TOPICS" | grep -v "^$" | wc -l)
    
    echo "现有 Topic 数量: ${TOPIC_COUNT}"
    
    if [ "$TOPIC_COUNT" -gt 0 ]; then
        echo "现有 Topic 列表:"
        echo "$CURRENT_TOPICS"
    fi
    
    echo ""
    echo ">>> 开始创建 Topic..."
    create_topic "sdklua_online" 5 2
    create_topic "debug" 5 2
    create_topic "pay_statisv2" 5 2
    create_topic "pay_zg_total" 5 2
    create_topic "pay_zg_total_random" 5 2
    create_topic "tag-evaluate-new" 1 1
    create_topic "file-task" 1 1
    create_topic "group-user-load" 1 1
    
    echo ""
    echo ">>> 验证 Topic 创建..."
    
    FINAL_TOPICS=$(get_topic_list)
    echo "当前 Topic 列表:"
    echo "$FINAL_TOPICS"
    
    echo ""
    echo "============================================"
    echo "Topic 初始化完成"
    echo "============================================"
}

# 执行主函数
main "$@"
