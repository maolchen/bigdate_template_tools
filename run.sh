#!/bin/bash
set -e

# 设置 Go 环境变量
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=https://goproxy.cn,direct

# 进入服务目录并启动
cd server
go run main.go
