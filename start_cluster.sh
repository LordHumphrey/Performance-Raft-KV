#!/bin/bash

# 集群启动脚本
# 用于启动指定数量的hraftd节点

# 默认参数
DEFAULT_NODE_COUNT=3
DEFAULT_BASE_HTTP_PORT=11000
DEFAULT_BASE_RAFT_PORT=12000
DEFAULT_BASE_ETCD_PORT=2379
DEFAULT_DATA_DIR="/tmp/hraftd_cluster"
DEFAULT_EXECUTABLE_PATH="./Raw/hraftd"

# 环境变量优先级高于默认值
HRAFTD_EXECUTABLE="${HRAFTD_EXECUTABLE:-$DEFAULT_EXECUTABLE_PATH}"
HRAFTD_DATA_DIR="${HRAFTD_DATA_DIR:-$DEFAULT_DATA_DIR}"

# 帮助信息
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -n, --nodes NUM     Number of nodes to start (default: $DEFAULT_NODE_COUNT)"
    echo "  -d, --data-dir DIR  Base directory for node data (default: $HRAFTD_DATA_DIR)"
    echo "  -e, --executable PATH  Path to hraftd executable (default: $HRAFTD_EXECUTABLE)"
    echo "  -h, --help          Show this help message"
    exit 1
}

# 解析命令行参数
NODES=$DEFAULT_NODE_COUNT
DATA_DIR="$HRAFTD_DATA_DIR"
INMEM=false

while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -n|--nodes)
            NODES="$2"
            shift 2
            ;;
        -d|--data-dir)
            DATA_DIR="$2"
            shift 2
            ;;
        -e|--executable)
            HRAFTD_EXECUTABLE="$2"
            shift 2
            ;;
        --inmem)
            INMEM=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# 在启动节点前检查可执行文件
if [ ! -x "$HRAFTD_EXECUTABLE" ]; then
    echo "错误: hraftd 可执行文件不存在或没有执行权限: $HRAFTD_EXECUTABLE"
    echo "请检查可执行文件路径或使用 -e 参数指定"
    exit 1
fi

# 确保数据目录存在并具有正确权限
mkdir -p "$DATA_DIR"
chmod 755 "$DATA_DIR"

# 启动第一个节点（初始集群）
echo "Starting first node (cluster leader)..."
FIRST_NODE_HTTP_PORT=$((DEFAULT_BASE_HTTP_PORT))
FIRST_NODE_RAFT_PORT=$((DEFAULT_BASE_RAFT_PORT))
FIRST_NODE_ETCD_PORT=$((DEFAULT_BASE_ETCD_PORT))
FIRST_NODE_DATA_DIR="$DATA_DIR/node0"

# 确保第一个节点的数据目录存在
mkdir -p "$FIRST_NODE_DATA_DIR"
chmod 755 "$FIRST_NODE_DATA_DIR"
touch "$FIRST_NODE_DATA_DIR/node.log"

INMEM_FLAG=""
if [ "$INMEM" = true ]; then
    INMEM_FLAG="--inmem"
fi

"$HRAFTD_EXECUTABLE" \
    -id node0 \
    -haddr "localhost:$FIRST_NODE_HTTP_PORT" \
    -raddr "localhost:$FIRST_NODE_RAFT_PORT" \
    -eaddr "localhost:$FIRST_NODE_ETCD_PORT" \
    $INMEM_FLAG \
    "$FIRST_NODE_DATA_DIR" &> "$FIRST_NODE_DATA_DIR/node.log" &

# 等待第一个节点启动
sleep 2

# 启动其他节点
for ((i=1; i<NODES; i++)); do
    echo "Starting node $i..."
    NODE_HTTP_PORT=$((DEFAULT_BASE_HTTP_PORT + i))
    NODE_RAFT_PORT=$((DEFAULT_BASE_RAFT_PORT + i))
    NODE_ETCD_PORT=$((DEFAULT_BASE_ETCD_PORT + i))
    NODE_DATA_DIR="$DATA_DIR/node$i"

    # 确保每个节点的数据目录存在
    mkdir -p "$NODE_DATA_DIR"
    chmod 755 "$NODE_DATA_DIR"
    touch "$NODE_DATA_DIR/node.log"

    "$HRAFTD_EXECUTABLE" \
        -id "node$i" \
        -haddr "localhost:$NODE_HTTP_PORT" \
        -raddr "localhost:$NODE_RAFT_PORT" \
        -eaddr "localhost:$NODE_ETCD_PORT" \
        -join "localhost:$FIRST_NODE_HTTP_PORT" \
        $INMEM_FLAG \
        "$NODE_DATA_DIR" &> "$NODE_DATA_DIR/node.log" &

    # 等待节点加入集群
    sleep 1
done

# 打印集群信息
echo "Cluster started with $NODES nodes:"
for ((i=0; i<NODES; i++)); do
    HTTP_PORT=$((DEFAULT_BASE_HTTP_PORT + i))
    RAFT_PORT=$((DEFAULT_BASE_RAFT_PORT + i))
    ETCD_PORT=$((DEFAULT_BASE_ETCD_PORT + i))
    echo "Node $i:"
    echo "  HTTP: localhost:$HTTP_PORT"
    echo "  Raft: localhost:$RAFT_PORT"
    echo "  Etcd: localhost:$ETCD_PORT"
    echo "  Log: $DATA_DIR/node$i/node.log"
done
