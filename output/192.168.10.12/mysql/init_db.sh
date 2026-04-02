#!/bin/bash
# ============================================================
# MySQL初始化脚本 - 设置时区
# 服务: mysql
# 节点: <no value> (<no value>)
# ============================================================

/usr/local/bin/docker exec zanalytics_mysql_v33 bash -c "ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime"
