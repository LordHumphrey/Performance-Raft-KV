# Etcd v3 API 支持

本项目已添加对 etcd v3 API 的支持，允许使用 etcd 客户端与我们的分布式 KV 存储进行交互。

## 支持的 API

目前支持以下 etcd v3 API：

1. **Range** - 用于获取键值对
2. **Put** - 用于设置键值对
3. **Delete** - 用于删除键值对

## 使用方法

### 启动服务器

使用`-eaddr`参数指定 etcd API 的监听地址：

```bash
./hraftd -eaddr localhost:2379 ~/node0
```

默认情况下，etcd API 将在`localhost:2379`上监听。

### 使用 etcd 客户端

您可以使用任何支持 etcd v3 API 的客户端与服务器进行交互。以下是一些示例：

#### 使用提供的命令行工具

我们提供了一个简单的命令行工具来测试 etcd API：

```bash
# 编译命令行工具
go build -o etcdclient cmd/etcdclient/main.go

# 设置键值对
./etcdclient -cmd put -key foo -value bar

# 获取键值对
./etcdclient -cmd get -key foo

# 删除键值对
./etcdclient -cmd delete -key foo
```

#### 使用官方 etcd 客户端

您也可以使用官方的 etcdctl 工具：

```bash
# 设置键值对
ETCDCTL_API=3 etcdctl --endpoints=localhost:2379 put foo bar

# 获取键值对
ETCDCTL_API=3 etcdctl --endpoints=localhost:2379 get foo

# 删除键值对
ETCDCTL_API=3 etcdctl --endpoints=localhost:2379 del foo
```

#### 使用 Go 客户端

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	// 创建etcd客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer cli.Close()

	// 设置键值对
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Put(ctx, "foo", "bar")
	cancel()
	if err != nil {
		log.Fatalf("failed to put key: %v", err)
	}

	// 获取键值对
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := cli.Get(ctx, "foo")
	cancel()
	if err != nil {
		log.Fatalf("failed to get key: %v", err)
	}
	for _, ev := range resp.Kvs {
		fmt.Printf("Key: %s, Value: %s\n", ev.Key, ev.Value)
	}

	// 删除键值对
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Delete(ctx, "foo")
	cancel()
	if err != nil {
		log.Fatalf("failed to delete key: %v", err)
	}
}
```

## 限制

当前实现有以下限制：

1. 不支持 etcd v2 API
2. 不支持事务（Txn）
3. 不支持压缩（Compact）
4. 不支持 Watch API
5. 不支持 Lease API
6. Range API 的前缀查询功能有限
7. 不支持 DeleteRange 的范围删除功能

这些限制可能会在未来的版本中解决。
