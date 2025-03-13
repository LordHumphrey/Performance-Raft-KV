# Raft 集群管理脚本

## 启动集群

使用 `start_cluster.sh` 脚本启动 Raft 集群。

### 基本用法

```bash
# 启动默认的3节点集群
./start_cluster.sh

# 启动5节点集群
./start_cluster.sh -n 5

# 启动使用内存存储的集群
./start_cluster.sh --inmem

# 指定自定义数据目录
./start_cluster.sh -d /path/to/cluster/data
```

### 参数说明

- `-n, --nodes NUM`：指定要启动的节点数（默认为 3）
- `-d, --data-dir DIR`：指定集群数据目录（默认为 `./cluster_data`）
- `--inmem`：使用内存存储而非持久化存储
- `-h, --help`：显示帮助信息

## 停止集群

使用 `stop_cluster.sh` 脚本停止 Raft 集群。

### 基本用法

```bash
# 正常停止集群
./stop_cluster.sh

# 强制停止集群（如果正常停止失败）
./stop_cluster.sh -f
```

### 参数说明

- `-f, --force`：强制终止所有 hraftd 进程
- `-h, --help`：显示帮助信息

## 注意事项

1. 确保在运行脚本之前已编译 `hraftd` 可执行文件
2. 第一个节点将作为集群的初始领导者
3. 其他节点将通过 `-join` 参数加入集群
4. 每个节点使用不同的端口：
   - HTTP: 11000, 11001, 11002...
   - Raft: 12000, 12001, 12002...
   - Etcd: 2379, 2380, 2381...

## 测试集群

启动集群后，您可以使用以下方式测试：

1. 使用 HTTP API

```bash
# 在任意节点设置键值对
curl -XPOST localhost:11000/key -d '{"test_key": "test_value"}'

# 在其他节点读取键值对
curl -XGET localhost:11001/key/test_key
```

2. 使用 Etcd API

```bash
# 使用 etcdctl 或提供的 etcdclient
./etcdclient -cmd put -key test_key -value test_value
./etcdclient -cmd get -key test_key
```

## 故障排除

- 如果节点无法加入集群，检查网络和防火墙设置
- 使用 `--inmem` 可以快速测试，但不提供持久化
- 查看日志输出以获取详细信息
