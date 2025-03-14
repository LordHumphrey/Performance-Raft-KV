#!/bin/bash

# 停止集群脚本
# 用于停止所有正在运行的hraftd节点并管理数据目录

# 默认数据目录
DEFAULT_DATA_DIR="/tmp/hraftd_cluster"

# 帮助信息
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -f, --force    Force kill all processes"
    echo "  -d, --data-dir DIR  Data directory to clean (default: $DEFAULT_DATA_DIR)"
    echo "  -h, --help     Show this help message"
    exit 1
}

# 解析命令行参数
FORCE=false
DATA_DIR="$DEFAULT_DATA_DIR"

while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -f|--force)
            FORCE=true
            shift
            ;;
        -d|--data-dir)
            DATA_DIR="$2"
            shift 2
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

# 停止进程的函数
stop_processes() {
    local signal="$1"
    echo "Stopping hraftd processes..."

    if [ "$FORCE" = true ]; then
        pkill -9 hraftd
        echo "Force killed all hraftd processes."
    else
        pkill "$signal" hraftd

        # 等待进程退出
        sleep 2

        # 检查是否还有进程存在
        if pgrep hraftd > /dev/null; then
            echo "Some hraftd processes did not stop. Use -f to force kill."
            exit 1
        fi
    fi
}

# 清理数据目录的函数
clean_data_dir() {
    local data_dir="$1"

    if [ -d "$data_dir" ]; then
        echo "Cleaning data directory: $data_dir"
        rm -rf "$data_dir"
    fi

    mkdir -p "$data_dir"
    echo "Data directory reset: $data_dir"
}

# 主逻辑
stop_processes "-SIGTERM"
clean_data_dir "$DATA_DIR"

echo "Cluster stopped and data directory cleaned successfully."
exit 0
