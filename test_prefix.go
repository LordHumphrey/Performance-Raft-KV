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

	// 使用WithPrefix进行查询
	fmt.Println("使用WithPrefix进行查询:")
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

	// 打印WithPrefix的原理
	fmt.Println("\nWithPrefix的实现原理:")
	// 获取前缀范围结束
	end := clientv3.GetPrefixRangeEnd("test_scan_")
	fmt.Printf("前缀: %q, 范围结束: %q\n", "test_scan_", end)

	// 打印每个字节的十六进制值
	fmt.Print("前缀的字节: ")
	for _, b := range []byte("test_scan_") {
		fmt.Printf("%02x ", b)
	}
	fmt.Println()

	fmt.Print("范围结束的字节: ")
	for _, b := range []byte(end) {
		fmt.Printf("%02x ", b)
	}
	fmt.Println()

	// 手动创建和验证前缀查询的rangeEnd
	testPrefix := "test_scan_"
	lastByte := testPrefix[len(testPrefix)-1]
	prefixRangeEnd := testPrefix[:len(testPrefix)-1] + string(lastByte+1)
	fmt.Printf("手动创建的范围结束: %q\n", prefixRangeEnd)

	// 验证是否与clientv3.GetPrefixRangeEnd相同
	if prefixRangeEnd == end {
		fmt.Println("手动创建的范围结束与clientv3.GetPrefixRangeEnd相同")
	} else {
		fmt.Println("手动创建的范围结束与clientv3.GetPrefixRangeEnd不同")
	}

	// 尝试手动构造范围查询
	fmt.Println("\n手动构造范围查询:")
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err = cli.Get(ctx, "test_scan_", clientv3.WithRange(prefixRangeEnd))
	cancel()
	if err != nil {
		log.Fatalf("failed to get with range: %v", err)
	}
	fmt.Printf("得到手动范围查询结果数量: %d\n", resp.Count)
	for _, kv := range resp.Kvs {
		fmt.Printf("键: %s, 值: %s\n", kv.Key, kv.Value)
	}

	// 打印所有存在的键
	fmt.Println("\n所有的键:")
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err = cli.Get(ctx, "", clientv3.WithFromKey())
	cancel()
	if err != nil {
		log.Fatalf("failed to get all keys: %v", err)
	}
	fmt.Printf("得到所有键的数量: %d\n", resp.Count)
	for _, kv := range resp.Kvs {
		fmt.Printf("键: %s, 值: %s\n", kv.Key, kv.Value)
	}
}
