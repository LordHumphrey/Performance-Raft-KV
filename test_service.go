package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"os"

	"github.com/otoolep/hraftd/store"
	"github.com/otoolep/hraftd/etcdapi"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	// 创建临时目录用于存储
	tempDir, err := os.MkdirTemp("", "etcd-test")
	if err != nil {
		log.Fatalf("failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建存储实例
	s := store.New(false)
	s.RaftDir = tempDir
	s.RaftBind = "localhost:12345" // 任意未使用的端口
	if err := s.Open(true, "test-node"); err != nil {
		log.Fatalf("failed to open store: %s", err.Error())
	}

	// 在测试端口启动etcd API服务
	e := etcdapi.New("localhost:22379", s)
	if err := e.Start(); err != nil {
		log.Fatalf("failed to start etcd API service: %s", err.Error())
	}
	defer e.Close()

	// 等待服务启动
	time.Sleep(2 * time.Second)

	// 创建etcd客户端连接到我们的服务
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:22379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer cli.Close()

	// 插入测试数据
	fmt.Println("插入测试数据...")
	testData := map[string]string{
		"test_scan_1": "value1",
		"test_scan_2": "value2",
		"test_scan_3": "value3",
		"test_scan_4": "value4",
		"test_scan_5": "value5",
	}

	for k, v := range testData {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err = cli.Put(ctx, k, v)
		cancel()
		if err != nil {
			log.Fatalf("failed to put %s: %v", k, err)
		}
		fmt.Printf("插入键值对: %s -> %s\n", k, v)
	}

	// 打印所有键值
	fmt.Println("\n存储中的所有键值对:")
	allData := s.ListN(0, false)
	for k, v := range allData {
		fmt.Printf("键: %s, 值: %s\n", k, v)
	}

	// 使用WithPrefix进行查询
	fmt.Println("\n使用WithPrefix进行查询:")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := cli.Get(ctx, "test_scan_", clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Fatalf("failed to get with prefix: %v", err)
	}
	fmt.Printf("得到前缀查询结果数量: %d\n", resp.Count)
	for _, kv := range resp.Kvs {
		fmt.Printf("键: %s, 值: %s\n", kv.Key, kv.Value)
	}

	// 使用WithRange进行查询
	fmt.Println("\n使用WithRange进行查询:")
	// 手动构造前缀查询的范围结束
	prefixRangeEnd := clientv3.GetPrefixRangeEnd("test_scan_")
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err = cli.Get(ctx, "test_scan_", clientv3.WithRange(prefixRangeEnd))
	cancel()
	if err != nil {
		log.Fatalf("failed to get with range: %v", err)
	}
	fmt.Printf("得到范围查询结果数量: %d\n", resp.Count)
	for _, kv := range resp.Kvs {
		fmt.Printf("键: %s, 值: %s\n", kv.Key, kv.Value)
	}

	// 使用WithLimit进行查询
	fmt.Println("\n使用WithLimit进行查询:")
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err = cli.Get(ctx, "test_scan_", clientv3.WithPrefix(), clientv3.WithLimit(3))
	cancel()
	if err != nil {
		log.Fatalf("failed to get with limit: %v", err)
	}
	fmt.Printf("得到带限制的查询结果数量: %d\n", resp.Count)
	for _, kv := range resp.Kvs {
		fmt.Printf("键: %s, 值: %s\n", kv.Key, kv.Value)
	}

	// 使用WithFromKey进行查询
	fmt.Println("\n使用WithFromKey进行查询:")
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err = cli.Get(ctx, "test_scan_3", clientv3.WithFromKey())
	cancel()
	if err != nil {
		log.Fatalf("failed to get with from key: %v", err)
	}
	fmt.Printf("得到从特定键开始的查询结果数量: %d\n", resp.Count)
	for _, kv := range resp.Kvs {
		fmt.Printf("键: %s, 值: %s\n", kv.Key, kv.Value)
	}

	// 删除测试数据
	fmt.Println("\n删除测试数据...")
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	delResp, err := cli.Delete(ctx, "test_scan_", clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Fatalf("failed to delete prefix: %v", err)
	}
	fmt.Printf("删除了 %d 个键值对\n", delResp.Deleted)

	fmt.Println("测试完成!")
}
