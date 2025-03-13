#!/bin/bash

# 集群启动脚本
# 用于启动指定数量的hraftd节点

# 默认参数
DEFAULT_NODE_COUNT=3
DEFAULT_BASE_HTTP_PORT=11000
DEFAULT_BASE_RAFT_PORT=12000
DEFAULT_BASE_ETCD_PORT=2379
DEFAULT_DATA_DIR="/tmp/hraftd_cluster"

# 帮助信息
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -n, --nodes NUM     Number of nodes to start (default: $DEFAULT_NODE_COUNT)"
    echo "  -d, --data-dir DIR  Base directory for node data (default: $DEFAULT_DATA_DIR)"
    echo "  -h, --help          Show this help message"
    exit 1
}

# 清理和准备数据目录的函数
prepare_data_dir() {
    local data_dir="$1"

    # 删除并重新创建数据目录
    rm -rf "$data_dir"
    mkdir -p "$data_dir"

    # 设置目录权限，确保当前用户可以读写
    chmod 755 "$data_dir"

    # 创建节点信息文件
    for ((i=0; i<NODES; i++)); do
        local node_dir="$data_dir/node$i"
        mkdir -p "$node_dir"
        chmod 755 "$node_dir"

        # 创建节点信息文件
        cat > "$node_dir/node_info.txt" << EOF
Node ID: node$i
HTTP Port: $((DEFAULT_BASE_HTTP_PORT + i))
Raft Port: $((DEFAULT_BASE_RAFT_PORT + i))
Etcd Port: $((DEFAULT_BASE_ETCD_PORT + i))
Data Directory: $node_dir
Timestamp: $(date '+%Y-%m-%d %H:%M:%S')
EOF
        chmod 644 "$node_dir/node_info.txt"

        # 创建空的键值存储文件
        touch "$node_dir/kv_store.json"
        chmod 644 "$node_dir/kv_store.json"

        # 创建日志文件
        touch "$node_dir/node.log"
        chmod 644 "$node_dir/node.log"
    done
}

# 解析命令行参数
NODES=$DEFAULT_NODE_COUNT
DATA_DIR=$DEFAULT_DATA_DIR
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

# 准备数据目录
prepare_data_dir "$DATA_DIR"

# 在启动节点前检查可执行文件
if [ ! -x "./hraftd" ]; then
    echo "错误: hraftd 可执行文件不存在或没有执行权限"
    echo "请先编译项目: go build 或 make"
    exit 1
fi

# 启动第一个节点（初始集群）
echo "Starting first node (cluster leader)..."
FIRST_NODE_HTTP_PORT=$((DEFAULT_BASE_HTTP_PORT))
FIRST_NODE_RAFT_PORT=$((DEFAULT_BASE_RAFT_PORT))
FIRST_NODE_ETCD_PORT=$((DEFAULT_BASE_ETCD_PORT))
FIRST_NODE_DATA_DIR="$DATA_DIR/node0"

INMEM_FLAG=""
if [ "$INMEM" = true ]; then
    INMEM_FLAG="--inmem"
fi

./hraftd \
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

    ./hraftd \
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

echo "Cluster is running. Press Ctrl+C to stop."
wait
