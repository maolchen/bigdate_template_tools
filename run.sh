#!/bin/bash
set -e

# 检查并安装 Go（如果未安装）
if ! command -v go &> /dev/null; then
    echo "Go not found, downloading and installing..."
    
    GO_VERSION="go1.21.13.linux-amd64"
    GO_TAR="${GO_VERSION}.tar.gz"
    GO_URL="https://studygolang.com/dl/golang/${GO_TAR}"
    
    # 下载 Go
    curl -fsSL "$GO_URL" -o "/tmp/${GO_TAR}"
    
    # 解压到 /usr/local
    tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    
    echo "Go installed successfully!"
fi

# 设置环境变量
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=https://goproxy.cn,direct

# 进入服务目录并启动
cd server
go run main.go
