#!/bin/bash
# 项目清理脚本 - 删除测试文件、输出文件和临时文件

set -e

echo "开始清理项目..."

# 删除临时输出目录
echo "删除 output/..."
rm -rf output

# 删除临时目录
echo "删除 tmp/..."
rm -rf tmp

# 删除旧的编译二进制文件
echo "删除旧的二进制文件..."
rm -f config-generator
rm -f server-bin
rm -f server.exe

# 删除运行日志
echo "删除运行日志..."
rm -f run.log

# 删除旧的 dist 目录
echo "删除 web/dist/..."
rm -rf web/dist

echo "清理完成！"
echo ""
echo "保留的文件和目录："
echo "  - main.go          # 源代码"
echo "  - config.yaml      # 配置文件"
echo "  - templates/       # 模板文件"
echo "  - web/             # 前端源码"
echo "  - *.sh             # 构建脚本"
echo "  - *.json           # 配置文件"
echo "  - *.md             # 文档"
echo "  - go.mod/go.sum    # Go 依赖"
