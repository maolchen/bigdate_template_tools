#!/bin/bash
set -e

# 部署环境构建脚本（不需要 Go，使用预编译的二进制文件）

# 安装前端依赖
pnpm install

# 构建前端
pnpm run build

# 复制前端构建产物
mkdir -p server/web/dist
cp -r web/dist/* server/web/dist/

echo "Build completed successfully!"
