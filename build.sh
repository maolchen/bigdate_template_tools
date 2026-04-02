#!/bin/bash
set -e

# 安装依赖
pnpm install

# 构建前端
pnpm run build

# 复制构建产物到 Go 服务目录
mkdir -p server/web/dist
cp -r web/dist/* server/web/dist/

echo "Build completed successfully!"
