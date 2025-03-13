package main

import (
	"context"
	"log"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestEtcdAPI(t *testing.T) {
	// 创建一个新的 etcd 客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建 etcd 客户端失败: %v", err)
	}
	defer cli.Close()

	// 测试 Put 操作
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Put(ctx, "test_key", "test_value")
	cancel()
	if err != nil {
		t.Fatalf("Put 操作失败: %v", err)
	}
	log.Println("Put 操作成功: test_key -> test_value")

	// 测试 Range 操作
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := cli.Get(ctx, "test_key")
	cancel()
	if err != nil {
		t.Fatalf("Get 操作失败: %v", err)
	}
	if len(resp.Kvs) == 0 {
		t.Fatalf("Get 操作未返回任何结果")
	}
	if string(resp.Kvs[0].Key) != "test_key" || string(resp.Kvs[0].Value) != "test_value" {
		t.Fatalf("Get 操作返回的结果不正确: %s -> %s", resp.Kvs[0].Key, resp.Kvs[0].Value)
	}
	log.Printf("Get 操作成功: %s -> %s", resp.Kvs[0].Key, resp.Kvs[0].Value)

	// 测试 Delete 操作
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Delete(ctx, "test_key")
	cancel()
	if err != nil {
		t.Fatalf("Delete 操作失败: %v", err)
	}
	log.Println("Delete 操作成功: test_key")

	// 验证 Delete 操作是否成功
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err = cli.Get(ctx, "test_key")
	cancel()
	if err != nil {
		t.Fatalf("验证 Delete 操作失败: %v", err)
	}
	if len(resp.Kvs) != 0 {
		t.Fatalf("Delete 操作未成功删除键")
	}
	log.Println("验证 Delete 操作成功: test_key 已被删除")

	log.Println("所有测试通过！")
}
