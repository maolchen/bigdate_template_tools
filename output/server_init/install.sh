#!/bin/bash
# ============================================================
# 服务器初始化脚本 - 全局服务示例
# 此脚本将在所有节点执行
# ============================================================

set -e

echo "=========================================="
echo "服务器初始化: realtime-es1"
echo "IP: 192.168.10.16"
echo "Hostname: realtime-es1"
echo "=========================================="

# 时区设置
echo "设置时区..."
ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
timedatectl set-timezone Asia/Shanghai

# 创建用户
echo "创建用户和组..."
groupadd -f bigdata
id -u bigdata &>/dev/null || useradd -g bigdata bigdata

# 创建目录
echo "创建基础目录..."
mkdir -p /data/localization
mkdir -p /data/bigdata
mkdir -p /var/log/bigdata
mkdir -p /data/tmp_install_dir
mkdir -p /data/softwares

# 设置权限
chown -R bigdata:bigdata /data/localization
chown -R bigdata:bigdata /data/bigdata
chown -R bigdata:bigdata /var/log/bigdata
chown -R bigdata:bigdata /data/tmp_install_dir
chown -R bigdata:bigdata /data/softwares

# 系统参数优化
echo "优化系统参数..."
cat >> /etc/sysctl.conf << EOF
# 大数据平台优化
vm.swappiness = 10
vm.dirty_ratio = 80
vm.dirty_background_ratio = 5
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.core.netdev_max_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 1200
net.ipv4.tcp_keepalive_probes = 5
net.ipv4.tcp_keepalive_intvl = 30
EOF
sysctl -p

# 文件描述符限制
echo "设置文件描述符限制..."
cat >> /etc/security/limits.conf << EOF
bigdata soft nofile 65536
bigdata hard nofile 65536
bigdata soft nproc 65536
bigdata hard nproc 65536
EOF

# 关闭防火墙
echo "关闭防火墙..."
systemctl stop firewalld 2>/dev/null || true
systemctl disable firewalld 2>/dev/null || true

# 关闭SELinux
echo "关闭SELinux..."
setenforce 0 2>/dev/null || true
sed -i 's/SELINUX=enforcing/SELINUX=disabled/g' /etc/selinux/config

# SSH免密设置提示
echo "=========================================="
echo "提示: 请确保已配置SSH免密登录"
echo "=========================================="

echo "服务器初始化完成: realtime-es1"
