#!/bin/bash
# 测试辅助脚本：先构建应用，启动集群，再运行测试，最后停止集群

set -e  # 任何命令失败立即退出脚本

echo "===== 开始构建应用 ====="
go build -o build/hraftd ./main.go

echo "===== 尝试停止可能已运行的集群 ====="
# 先尝试停止可能已存在的集群
if [ -f "./test/stop_cluster.sh" ]; then
    bash ./test/stop_cluster.sh || true  # 即使停止失败也继续
fi

echo "===== 启动测试集群 ====="
# 使用build目录中的可执行文件启动三节点集群
export HRAFTD_EXECUTABLE="./build/hraftd"
export HRAFTD_DATA_DIR="/tmp/hraftd_test_cluster"
bash ./test/start_cluster.sh --nodes 3

# 等待集群完全启动
echo "===== 等待集群启动 ====="
sleep 5

echo "===== 运行测试 ====="
# 运行etcd API测试
go test -v ./etcd_test.go

echo "===== 停止测试集群 ====="
# 测试完成后停止集群
bash ./test/stop_cluster.sh

echo "===== 测试流程完成 ====="