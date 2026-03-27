#!/bin/bash
# Hadoop 3 YARN 启动脚本
# 使用方法: bash start_yarn.sh
# 说明: 使用 systemd 启动 YARN 组件 (ResourceManager 和 NodeManager)
# 注意: 必须确保 HDFS 已启动

set -e

# ============================================================
# 全局变量
# ============================================================
INSTALL_BASE_DIR="/data/localization"
DATA_BASE_DIR="/data"
RUN_USER="bigdata"
RUN_GROUP="bigdata"
JAVA_HOME="/data/jdk/"

# ============================================================
# 辅助函数
# ============================================================
run_as_root() {
    if [ "$RUN_USER" = "root" ]; then
        "$@"
    else
        sudo "$@"
    fi
}

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $1" >&2
}

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $1"
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SUCCESS] $1"
}

check_port() {
    local host=$1
    local port=$2
    if timeout 5 bash -c "echo > /dev/tcp/${host}/${port}" 2>/dev/null; then
        return 0
    fi
    return 1
}

# ============================================================
# Hadoop 配置
# ============================================================
HADOOP_HOME="${INSTALL_BASE_DIR}/hadoop"
HADOOP_CONF_DIR="${HADOOP_HOME}/etc/hadoop"
YARN_DIR="${DATA_BASE_DIR}/hadoop/hadoop-yarn"
NAMESERVICE="zhugeio"
NN_RPC_PORT="8020"
RM_PORT="8032"
RM_WEB_PORT="8089"

# ============================================================
# 开始启动
# ============================================================
echo "============================================"
echo "Hadoop 3 YARN 启动脚本"
echo "============================================"
echo "主机名: $(hostname)"
echo "============================================"

# ============================================================
# 步骤1: 检查前置条件
# ============================================================
log_info "步骤1: 检查前置条件..."

# 检查Hadoop安装
if [ ! -d "${HADOOP_HOME}" ]; then
    log_error "Hadoop未安装，请先执行 install_binary.sh"
    exit 1
fi

# 检查配置文件
if [ ! -f "${HADOOP_CONF_DIR}/yarn-site.xml" ]; then
    log_error "配置文件不存在，请先执行 deploy_config.sh"
    exit 1
fi

log_info "前置条件检查通过"

# ============================================================
# 步骤2: 检查 HDFS 状态
# ============================================================
log_info "步骤2: 检查 HDFS 状态..."
log_warn "必须确保 HDFS 已启动！"

HDFS_READY=false

if [ "$HDFS_READY" = "false" ]; then
    log_error "无法连接到任何 NameNode，请先启动 HDFS"
    exit 1
fi

# ============================================================
# 步骤3: 检查 ZooKeeper 状态
# ============================================================
log_info "步骤3: 检查 ZooKeeper 状态..."

ZK_READY=false
if check_port "192.168.10.10" "2181"; then
    log_info "ZooKeeper dw-master1 可连接"
    ZK_READY=true
fi
if check_port "192.168.10.11" "2181"; then
    log_info "ZooKeeper dw-master2 可连接"
    ZK_READY=true
fi
if check_port "192.168.10.12" "2181"; then
    log_info "ZooKeeper dw-master3 可连接"
    ZK_READY=true
fi

if [ "$ZK_READY" = "false" ]; then
    log_warn "无法连接到 ZooKeeper，ResourceManager HA 可能无法正常工作"
fi

# ============================================================
# 步骤4: 准备 YARN 目录
# ============================================================
log_info "步骤4: 准备 YARN 目录..."

YARN_DIRS=(
    "${YARN_DIR}/cache"
    "${YARN_DIR}/containers"
)

for dir_path in "${YARN_DIRS[@]}"; do
    if [ ! -d "${dir_path}" ]; then
        run_as_root mkdir -p "${dir_path}"
        run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${dir_path}"
        log_info "创建目录: ${dir_path}"
    else
        run_as_root chown -R ${RUN_USER}:${RUN_GROUP} "${dir_path}"
        log_info "目录已存在: ${dir_path}"
    fi
done

# ============================================================
# 步骤5: 判断当前节点角色
# ============================================================
log_info "步骤5: 判断当前节点角色..."

IS_RM_NODE=false
IS_NM_NODE=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ============================================================
# 步骤6: 启动 ResourceManager（使用 systemd）
# ============================================================
if [ "$IS_RM_NODE" = "true" ]; then
    log_info "步骤6: 启动 ResourceManager..."
    
    RM_SERVICE="hadoop-yarn-resourcemanager"
    SERVICE_FILE="${SCRIPT_DIR}/systemd/${RM_SERVICE}.service"
    
    if [ ! -f "${SERVICE_FILE}" ]; then
        log_error "未找到 ResourceManager 服务文件: ${SERVICE_FILE}"
        exit 1
    fi
    
    # 部署服务文件
    run_as_root cp "${SERVICE_FILE}" /etc/systemd/system/
    run_as_root chmod 644 /etc/systemd/system/${RM_SERVICE}.service
    run_as_root systemctl daemon-reload
    run_as_root systemctl enable ${RM_SERVICE}
    log_info "ResourceManager 服务已部署并启用"
    
    # 检查是否已运行
    if systemctl is-active --quiet ${RM_SERVICE} 2>/dev/null; then
        log_info "ResourceManager 已在运行"
    else
        # 启动服务
        run_as_root systemctl start ${RM_SERVICE}
        sleep 3
        
        if systemctl is-active --quiet ${RM_SERVICE}; then
            log_success "ResourceManager 启动成功"
        else
            log_error "ResourceManager 启动失败"
            log_error "查看日志: sudo journalctl -u ${RM_SERVICE} -n 50"
        fi
    fi
else
    log_info "当前节点不是 ResourceManager 节点，跳过"
fi

# ============================================================
# 步骤7: 启动 NodeManager（使用 systemd）
# ============================================================
if [ "$IS_NM_NODE" = "true" ]; then
    log_info "步骤7: 启动 NodeManager..."
    
    NM_SERVICE="hadoop-yarn-nodemanager"
    SERVICE_FILE="${SCRIPT_DIR}/systemd/${NM_SERVICE}.service"
    
    if [ ! -f "${SERVICE_FILE}" ]; then
        log_error "未找到 NodeManager 服务文件: ${SERVICE_FILE}"
        exit 1
    fi
    
    # 部署服务文件
    run_as_root cp "${SERVICE_FILE}" /etc/systemd/system/
    run_as_root chmod 644 /etc/systemd/system/${NM_SERVICE}.service
    run_as_root systemctl daemon-reload
    run_as_root systemctl enable ${NM_SERVICE}
    log_info "NodeManager 服务已部署并启用"
    
    # 检查是否已运行
    if systemctl is-active --quiet ${NM_SERVICE} 2>/dev/null; then
        log_info "NodeManager 已在运行"
    else
        # 启动服务
        run_as_root systemctl start ${NM_SERVICE}
        sleep 3
        
        if systemctl is-active --quiet ${NM_SERVICE}; then
            log_success "NodeManager 启动成功"
        else
            log_error "NodeManager 启动失败"
            log_error "查看日志: sudo journalctl -u ${NM_SERVICE} -n 50"
        fi
    fi
else
    log_info "当前节点不是 NodeManager 节点，跳过"
fi

# ============================================================
# 步骤8: 验证 YARN 状态
# ============================================================
log_info "步骤8: 验证 YARN 状态..."

sleep 5

if [ "$IS_RM_NODE" = "true" ]; then
    echo ""
    echo "ResourceManager HA 状态:"
    ${HADOOP_HOME}/bin/yarn rmadmin -getAllServiceState 2>/dev/null || log_warn "无法获取 HA 状态"
    
    echo ""
    echo "NodeManager 列表:"
    ${HADOOP_HOME}/bin/yarn node -list 2>/dev/null || log_warn "无法获取节点列表"
fi

# ============================================================
# 启动完成
# ============================================================
echo ""
echo "============================================"
echo "YARN 启动完成！"
echo "============================================"
echo ""
echo "管理命令:"
if [ "$IS_RM_NODE" = "true" ]; then
    echo "  ResourceManager:"
    echo "    查看状态: sudo systemctl status hadoop-yarn-resourcemanager"
    echo "    停止服务: sudo systemctl stop hadoop-yarn-resourcemanager"
    echo "    查看日志: sudo journalctl -u hadoop-yarn-resourcemanager -f"
    echo ""
fi
if [ "$IS_NM_NODE" = "true" ]; then
    echo "  NodeManager:"
    echo "    查看状态: sudo systemctl status hadoop-yarn-nodemanager"
    echo "    停止服务: sudo systemctl stop hadoop-yarn-nodemanager"
    echo "    查看日志: sudo journalctl -u hadoop-yarn-nodemanager -f"
    echo ""
fi
echo "YARN 命令:"
echo "  yarn node -list"
echo ""
if [ "$IS_RM_NODE" = "true" ]; then
    echo "Web UI:"
    echo "  ResourceManager: http://$(hostname):${RM_WEB_PORT}"
fi
