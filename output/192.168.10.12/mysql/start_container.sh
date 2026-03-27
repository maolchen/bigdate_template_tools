#!/bin/bash
# ============================================================
# MySQL容器启动脚本
# 服务: mysql
# 节点: dw-master3 (192.168.10.12)
# ============================================================

docker run --restart=always -d --name zanalytics_mysql_v33 \
     -e MYSQL_USER=web \
     -e MYSQL_PASSWORD=zanalytics \
     -e MYSQL_ROOT_PASSWORD=zanalytics \
     -v /data/localization/mysql/my.cnf:/etc/mysql/conf.d/my.cnf \
     -v /data/localization/mysql/data:/var/lib/mysql \
     -e MYSQL_DATABASE=sdkv \
     -p 3306:3306 \
     mysql:5.7.33
