#!/bin/bash
set -e

# 设置 Go 安装目录（使用 /tmp 避免权限问题）
GO_INSTALL_DIR="/tmp/go"
GO_BIN="${GO_INSTALL_DIR}/go/bin/go"

# 检查 Go 是否已安装
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
    
    echo "Go installed successfully: $($GO_BIN version)"
fi

# 设置环境变量
export PATH="${GO_INSTALL_DIR}/go/bin:$PATH"
export GOPROXY=https://goproxy.cn,direct

# 安装前端依赖
pnpm install

# 构建前端
pnpm run build

# 复制构建产物到 Go 服务目录
mkdir -p server/web/dist
cp -r web/dist/* server/web/dist/

echo "Build completed successfully!"
