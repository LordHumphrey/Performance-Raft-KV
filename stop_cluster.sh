#!/bin/bash

# 停止集群脚本
# 用于停止所有正在运行的hraftd节点

# 帮助信息
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -f, --force    Force kill all processes"
    echo "  -h, --help     Show this help message"
    exit 1
}

# 解析命令行参数
FORCE=false

while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -f|--force)
            FORCE=true
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
clean_data_dirs() {
    read -p "Do you want to remove cluster data directories? (y/N) " confirm
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        rm -rf ./cluster_data
        echo "Cluster data directories removed."
    fi
}

# 主逻辑
stop_processes "-SIGTERM"
clean_data_dirs

echo "Cluster stopped successfully."
