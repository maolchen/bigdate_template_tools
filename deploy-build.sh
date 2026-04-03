#!/bin/bash
set -e

# 部署环境构建脚本（不需要 Go，使用预编译的二进制文件）

# 安装前端依赖
pnpm install

# 构建前端
pnpm run build

# 复制前端构建产物
mkdir -p web/dist
cp -r server/web/dist/* web/dist/ 2>/dev/null || echo "server/web/dist 不存在，跳过复制"

echo "Build completed successfully!"
