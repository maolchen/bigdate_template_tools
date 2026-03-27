#!/bin/bash
# ============================================================
# MySQL用户初始化脚本
# 服务: mysql
# 节点: dw-master3 (192.168.10.12)
# ============================================================

set -e

# ==================== 变量定义 ====================
MYSQL_HOST="realtime-db"
MYSQL_PORT="3306"
MYSQL_ROOT_PASSWORD="zanalytics"
MYSQL_USER="web"
MYSQL_PASSWORD="zanalytics"
DB_0="sdkv"
DB_1="audit_log"
DB_2="cep"

# ==================== 等待MySQL启动 ====================
echo "=> Waiting for MySQL service to start..."
RET=1
MAX_RETRIES=30
RETRY_COUNT=0

while [[ RET -ne 0 ]]; do
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -gt $MAX_RETRIES ]; then
        echo "=> ERROR: MySQL did not start within expected time"
        exit 1
    fi
    
    echo "=> Waiting for confirmation of MySQL service startup (attempt $RETRY_COUNT/$MAX_RETRIES)"
    sleep 5
    /usr/bin/mysql -h "$MYSQL_HOST" -P"$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" -e "status" > /dev/null 2>&1
    RET=$?
done

echo "=> MySQL service is up and running"

# ==================== 显示密码信息 ====================
echo ""
echo "============================MYSQL_PASS_INFO==============================="
echo ""
echo "=> MySQL root password: $MYSQL_ROOT_PASSWORD"
echo "=> MySQL web user password: $MYSQL_PASSWORD"
echo ""

# ==================== 创建用户和数据库 ====================
echo "=> Creating MySQL user '$MYSQL_USER'..."

/usr/bin/mysql -h "$MYSQL_HOST" -P"$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" -e "grant all privileges on *.* to '${MYSQL_USER}'@'%' identified by '${MYSQL_PASSWORD}'"

/usr/bin/mysql -h "$MYSQL_HOST" -P"$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" -e 'SET PASSWORD FOR "root" = PASSWORD("'"${MYSQL_ROOT_PASSWORD}"'")'

/usr/bin/mysql -h "$MYSQL_HOST" -P"$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" -e "flush privileges"

# ==================== 创建数据库 ====================
echo "=> Creating databases..."
echo "=> Creating database: sdkv"
/usr/bin/mysql -h "$MYSQL_HOST" -P"$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS sdkv DEFAULT CHARSET utf8 COLLATE utf8_general_ci"
echo "=> Creating database: audit_log"
/usr/bin/mysql -h "$MYSQL_HOST" -P"$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS audit_log DEFAULT CHARSET utf8 COLLATE utf8_general_ci"
echo "=> Creating database: cep"
/usr/bin/mysql -h "$MYSQL_HOST" -P"$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS cep DEFAULT CHARSET utf8 COLLATE utf8_general_ci"

echo ""
echo "=> MySQL initialization completed successfully!"
