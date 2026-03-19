#!/bin/bash
# ============================================================
# JDK安装脚本 - 全局服务示例
# 此脚本将在所有节点执行
# ============================================================

set -e

echo "=========================================="
echo "安装JDK: dw-worker2"
echo "IP: 192.168.10.14"
echo "=========================================="

# JDK安装目录
JDK_PATH="/data/jdk"
JAVA_HOME="/data/jdk/jdk1.8.0_65"
JDK_VERSION="1.8.0_65"

# 检查JDK是否已安装
if [ -d "$JAVA_HOME" ]; then
    echo "JDK已安装: $JAVA_HOME"
    exit 0
fi

# 创建安装目录
mkdir -p "$JDK_PATH"

# 检查安装包是否存在（假设安装包已上传到临时目录）
JDK_TAR="/data/tmp_install_dir/jdk-${JDK_VERSION}-linux-x64.tar.gz"

if [ ! -f "$JDK_TAR" ]; then
    echo "警告: JDK安装包不存在: $JDK_TAR"
    echo "请确保安装包已上传到临时目录"
    exit 1
fi

# 解压安装
echo "解压JDK安装包..."
cd "$JDK_PATH"
tar -xzf "$JDK_TAR"

# 配置环境变量
echo "配置环境变量..."
cat >> /etc/profile << 'EOF'

# Java Environment
export JAVA_HOME=/data/jdk/jdk1.8.0_65
export PATH=$JAVA_HOME/bin:$PATH
EOF

# 验证安装
echo "验证安装..."
export JAVA_HOME="$JAVA_HOME"
export PATH=$JAVA_HOME/bin:$PATH
java -version

echo "=========================================="
echo "JDK安装完成: dw-worker2"
echo "JAVA_HOME: $JAVA_HOME"
echo "=========================================="
