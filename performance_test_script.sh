#!/bin/bash

# 性能测试脚本 - 键值存储系统

# 设置基础路径和变量
HRAFTD_EXECUTABLE="./Raw/hraftd"
HRAFTD_DATA_DIR="./Raw/cluster_data"
WORKLOAD_PATH="/home/taowong/Dev/Perf-KV"
ETCD_ENDPOINTS="http://localhost:2379"
YCSB_EXECUTABLE="./go-ycsb"

# 测试结果保存在可执行文件目录下
BASE_RESULT_DIR="./Raw/Test-Result-$(date +"%Y%m%d_%H%M%S")"
WORKLOAD_TYPES=("workloada" "workloadb" "workloadc" "workloadd" "workloade" "workloadf")
CLUSTER_SIZES=(3 5 7 9 11 15 21 31)

# 创建结果目录
mkdir -p "$BASE_RESULT_DIR"

# 日志记录函数
log_message() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$BASE_RESULT_DIR/test_log.txt"
}

# 清理和重置集群函数
reset_cluster() {
    local node_count=$1
    log_message "重置集群，节点数量：$node_count"

    # 使用stop_cluster.sh停止集群并清理数据目录
    ./stop_cluster.sh -d "$HRAFTD_DATA_DIR"

    # 启动新集群
    ./start_cluster.sh -n "$node_count" -d "$HRAFTD_DATA_DIR" -e "$HRAFTD_EXECUTABLE"

    # 等待集群稳定 - 增加更可靠的检测机制
    local max_wait=60  # 最大等待时间
    local wait_interval=2
    local total_wait=0

    while [ $total_wait -lt $max_wait ]; do
        # 检查是否有足够的节点启动
        local running_nodes=$(ps aux | grep -c "[h]raftd")
        if [ "$running_nodes" -eq "$node_count" ]; then
            log_message "集群启动成功，节点数：$running_nodes"
            break
        fi

        sleep $wait_interval
        total_wait=$((total_wait + wait_interval))
    done

    if [ $total_wait -ge $max_wait ]; then
        log_message "错误：集群启动超时"
        exit 1
    fi

    # 额外等待，确保集群完全稳定
    sleep 5
}

# 执行性能测试函数
run_performance_test() {
    local cluster_size=$1
    local workload_type=$2
    local iteration=$3

    local result_dir="$BASE_RESULT_DIR/nodes_${cluster_size}/${workload_type}/iteration_${iteration}"
    mkdir -p "$result_dir"

    log_message "开始测试 - 集群大小: $cluster_size, 负载类型: $workload_type, 迭代: $iteration"

    # 加载数据
    "$YCSB_EXECUTABLE" load etcd -P "$WORKLOAD_PATH/$workload_type" -p etcd.endpoints="$ETCD_ENDPOINTS" \
        > "$result_dir/load_output.txt" 2>&1

    # 运行测试
    "$YCSB_EXECUTABLE" run etcd -P "$WORKLOAD_PATH/$workload_type" -p etcd.endpoints="$ETCD_ENDPOINTS" \
        > "$result_dir/run_output.txt" 2>&1

    # 收集性能指标
    cp INSERT-percentiles.txt "$result_dir/insert_percentiles.txt"
    cp READ-percentiles.txt "$result_dir/read_percentiles.txt"
    cp UPDATE-percentiles.txt "$result_dir/update_percentiles.txt"
    cp TOTAL-percentiles.txt "$result_dir/total_percentiles.txt"

    log_message "测试完成 - 集群大小: $cluster_size, 负载类型: $workload_type, 迭代: $iteration"
}

# 主测试循环
main() {
    log_message "开始性能测试"

    # 三轮测试
    for round in {1..3}; do
        log_message "开始第 $round 轮测试"

        for cluster_size in "${CLUSTER_SIZES[@]}"; do
            # 重置集群
            reset_cluster "$cluster_size"

            for workload_type in "${WORKLOAD_TYPES[@]}"; do
                # 执行测试
                run_performance_test "$cluster_size" "$workload_type" "$round"
            done
        done
    done

    log_message "所有性能测试完成"
}

# 执行主测试
main

# 生成总结报告
generate_summary() {
    find "$BASE_RESULT_DIR" -name "run_output.txt" | xargs grep -H "Throughput" > "$BASE_RESULT_DIR/performance_summary.txt"
}

generate_summary

log_message "测试结果已保存在 $BASE_RESULT_DIR"
