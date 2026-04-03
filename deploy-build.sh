#!/bin/bash
set -e

# 部署环境构建脚本（不需要 Go，使用预编译的二进制文件）

# 安装前端依赖
pnpm install

# 构建前端
pnpm run build

echo "Build completed successfully!"
echo ""
echo "部署说明："
echo "  - 确保 server-bin 或 server.exe 已存在"
echo "  - 运行：./server-bin --web"
