#!/bin/bash
# ============================================================
# Zookeeper 安装脚本
# 节点: dw-master3
# IP: 192.168.10.12
# myid: 3
# ============================================================

set -e

echo "=========================================="
echo "安装 Zookeeper"
echo "节点: dw-master3"
echo "IP: 192.168.10.12"
echo "myid: 3"
echo "=========================================="

# 变量定义
ZK_VERSION="3.8.3"
ZK_INSTALL_DIR="/data/localization/zookeeper"
ZK_DATA_DIR="/data/bigdata/zookeeper/data"
ZK_LOG_DIR="/var/log/bigdata/zookeeper"
ZK_USER="bigdata"
ZK_TAR="/data/tmp_install_dir/apache-zookeeper-${ZK_VERSION}-bin.tar.gz"

# 检查安装包
if [ ! -f "$ZK_TAR" ]; then
    echo "错误: 安装包不存在: $ZK_TAR"
    exit 1
fi

# 创建目录
mkdir -p "$ZK_INSTALL_DIR"
mkdir -p "$ZK_DATA_DIR"
mkdir -p "$ZK_LOG_DIR"

# 解压安装
echo "解压 Zookeeper..."
cd "/data/localization"
tar -xzf "$ZK_TAR"
mv apache-zookeeper-${ZK_VERSION}-bin zookeeper

# 复制配置文件
echo "复制配置文件..."
cp dw-master3_zoo.cfg "$ZK_INSTALL_DIR/conf/zoo.cfg"

# 创建myid文件
echo "创建 myid 文件..."
echo "3" > "$ZK_DATA_DIR/myid"

# 设置权限
chown -R $ZK_USER:$ZK_USER "$ZK_INSTALL_DIR"
chown -R $ZK_USER:$ZK_USER "$ZK_DATA_DIR"
chown -R $ZK_USER:$ZK_USER "$ZK_LOG_DIR"

# 创建启动脚本
echo "创建启动脚本..."
cat > "$ZK_INSTALL_DIR/bin/start.sh" << 'EOF'
#!/bin/bash
export JAVA_HOME=/data/jdk/jdk1.8.0_65
cd /data/localization/zookeeper
./bin/zkServer.sh start
EOF
chmod +x "$ZK_INSTALL_DIR/bin/start.sh"

# 创建停止脚本
cat > "$ZK_INSTALL_DIR/bin/stop.sh" << 'EOF'
#!/bin/bash
export JAVA_HOME=/data/jdk/jdk1.8.0_65
cd /data/localization/zookeeper
./bin/zkServer.sh stop
EOF
chmod +x "$ZK_INSTALL_DIR/bin/stop.sh"

echo "=========================================="
echo "Zookeeper 安装完成"
echo "安装目录: $ZK_INSTALL_DIR"
echo "数据目录: $ZK_DATA_DIR"
echo "myid: 3"
echo "=========================================="
