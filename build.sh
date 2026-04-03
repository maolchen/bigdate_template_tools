#!/bin/bash
set -e

# 设置 Go 安装目录
GO_INSTALL_DIR="/tmp/go"
GO_BIN="${GO_INSTALL_DIR}/go/bin/go"

# 检查并安装 Go（如果未安装）
if [ ! -f "$GO_BIN" ]; then
    echo "Go not found, downloading and installing to ${GO_INSTALL_DIR}..."

    GO_VERSION="go1.21.13.linux-amd64"
    GO_TAR="${GO_VERSION}.tar.gz"
    GO_URL="https://studygolang.com/dl/golang/${GO_TAR}"

    mkdir -p "$GO_INSTALL_DIR"
    curl -fsSL "$GO_URL" -o "/tmp/${GO_TAR}"
    tar -C "$GO_INSTALL_DIR" -xzf "/tmp/${GO_TAR}"

    echo "Go installed successfully!"
fi

export PATH="${GO_INSTALL_DIR}/go/bin:$PATH"
export GOPROXY=https://goproxy.cn,direct

# 安装前端依赖
pnpm install

# 构建前端
pnpm run build

# 编译 Go 后端为静态二进制（使用根目录的 main.go）
echo "Compiling Go backend..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server-bin main.go

# 编译 Windows 二进制
echo "Compiling Windows binary..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o server.exe main.go

echo ""
echo "Build completed successfully!"
echo ""
echo "Binaries:"
echo "  - server-bin (Linux/Mac, $(ls -lh server-bin | awk '{print $5}'))"
echo "  - server.exe (Windows, $(ls -lh server.exe | awk '{print $5}'))"
