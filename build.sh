#!/bin/bash
set -e

# 检查 Go 是否已安装
if ! command -v go &> /dev/null; then
    echo "Go not found, downloading and installing..."
    
    GO_VERSION="go1.21.13.linux-amd64"
    GO_TAR="${GO_VERSION}.tar.gz"
    GO_URL="https://studygolang.com/dl/golang/${GO_TAR}"
    
    # 下载 Go
    curl -fsSL "$GO_URL" -o "/tmp/${GO_TAR}"
    
    # 解压到 /usr/local
    tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    
    # 设置环境变量
    export PATH=$PATH:/usr/local/go/bin
    export GOPROXY=https://goproxy.cn,direct
    
    echo "Go installed successfully: $(go version)"
fi

# 安装前端依赖
pnpm install

# 构建前端
pnpm run build

# 复制构建产物到 Go 服务目录
mkdir -p server/web/dist
cp -r web/dist/* server/web/dist/

echo "Build completed successfully!"
