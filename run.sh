#!/bin/bash
set -e

# 设置 Go 安装目录（使用 /tmp 避免权限问题）
GO_INSTALL_DIR="/tmp/go"
GO_BIN="${GO_INSTALL_DIR}/go/bin/go"

# 检查并安装 Go（如果未安装）
if [ ! -f "$GO_BIN" ]; then
    echo "Go not found, downloading and installing to ${GO_INSTALL_DIR}..."
    
    GO_VERSION="go1.21.13.linux-amd64"
    GO_TAR="${GO_VERSION}.tar.gz"
    GO_URL="https://studygolang.com/dl/golang/${GO_TAR}"
    
    # 创建安装目录
    mkdir -p "$GO_INSTALL_DIR"
    
    # 下载 Go
    curl -fsSL "$GO_URL" -o "/tmp/${GO_TAR}"
    
    # 解压到 /tmp/go
    tar -C "$GO_INSTALL_DIR" -xzf "/tmp/${GO_TAR}"
    
    echo "Go installed successfully!"
fi

# 设置环境变量
export PATH="${GO_INSTALL_DIR}/go/bin:$PATH"
export GOPROXY=https://goproxy.cn,direct

# 进入服务目录并启动
cd server
go run main.go
