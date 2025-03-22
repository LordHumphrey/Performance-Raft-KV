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
	if (err != nil) {
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

// TestEtcdScanAPI 测试 Scan 操作，即前缀查询功能
func TestEtcdScanAPI(t *testing.T) {
	// 创建一个新的 etcd 客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建 etcd 客户端失败: %v", err)
	}
	defer cli.Close()

	// 准备测试数据 - 插入多个具有相同前缀的键
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// 清理可能存在的测试数据
	_, err = cli.Delete(ctx, "test_scan_", clientv3.WithPrefix())
	if err != nil {
		t.Logf("清理测试数据时出错(可以忽略): %v", err)
	}
	
	// 插入测试数据
	testData := map[string]string{
		"test_scan_1": "value1",
		"test_scan_2": "value2",
		"test_scan_3": "value3",
		"test_scan_4": "value4",
		"test_scan_5": "value5",
	}
	
	for k, v := range testData {
		_, err = cli.Put(ctx, k, v)
		if err != nil {
			t.Fatalf("插入测试数据失败 %s -> %s: %v", k, v, err)
		}
	}
	log.Println("成功插入测试数据")
	
	// 测试 1: 使用 WithPrefix() 进行前缀查询
	resp, err := cli.Get(ctx, "test_scan_", clientv3.WithPrefix())
	if err != nil {
		t.Fatalf("前缀查询失败: %v", err)
	}
	
	if int(resp.Count) != len(testData) {
		t.Fatalf("前缀查询返回的结果数量不正确: 期望 %d, 实际 %d", len(testData), resp.Count)
	}
	
	// 验证所有键值对是否正确返回
	foundKeys := make(map[string]bool)
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)
		
		expectedValue, exists := testData[key]
		if !exists {
			t.Fatalf("前缀查询返回了意外的键: %s", key)
		}
		
		if expectedValue != value {
			t.Fatalf("键 %s 的值不正确: 期望 %s, 实际 %s", key, expectedValue, value)
		}
		
		foundKeys[key] = true
	}
	
	// 确保所有键都被找到
	for k := range testData {
		if !foundKeys[k] {
			t.Fatalf("键 %s 未在前缀查询结果中找到", k)
		}
	}
	log.Println("前缀查询测试通过")
	
	// 测试 2: 使用 WithLimit() 限制返回结果数量
	limitCount := 3
	resp, err = cli.Get(ctx, "test_scan_", clientv3.WithPrefix(), clientv3.WithLimit(int64(limitCount)))
	if err != nil {
		t.Fatalf("带限制的前缀查询失败: %v", err)
	}
	
	if int(resp.Count) != limitCount {
		t.Fatalf("带限制的前缀查询返回的结果数量不正确: 期望 %d, 实际 %d", limitCount, resp.Count)
	}
	log.Println("带限制的前缀查询测试通过")
	
	// 测试 3: 使用 WithFromKey() 进行范围查询
	// 从 "test_scan_3" 开始查询所有后续键
	resp, err = cli.Get(ctx, "test_scan_3", clientv3.WithFromKey())
	if err != nil {
		t.Fatalf("WithFromKey 范围查询失败: %v", err)
	}
	
	// 检查结果 - 应该返回 test_scan_3, test_scan_4, test_scan_5
	expectedKeys := []string{"test_scan_3", "test_scan_4", "test_scan_5"}
	if int(resp.Count) != len(expectedKeys) {
		t.Fatalf("WithFromKey 范围查询返回的结果数量不正确: 期望 %d, 实际 %d", len(expectedKeys), resp.Count)
	}
	
	// 验证返回的键是否正确
	foundKeys = make(map[string]bool)
	for _, kv := range resp.Kvs {
		foundKeys[string(kv.Key)] = true
	}
	
	for _, k := range expectedKeys {
		if !foundKeys[k] {
			t.Fatalf("键 %s 未在 WithFromKey 范围查询结果中找到", k)
		}
	}
	log.Println("WithFromKey 范围查询测试通过")
	
	// 测试 4: 结合 WithFromKey() 和 WithLimit()
	resp, err = cli.Get(ctx, "test_scan_2", clientv3.WithFromKey(), clientv3.WithLimit(2))
	if err != nil {
		t.Fatalf("WithFromKey+WithLimit 查询失败: %v", err)
	}
	
	if int(resp.Count) != 2 {
		t.Fatalf("WithFromKey+WithLimit 查询返回的结果数量不正确: 期望 2, 实际 %d", resp.Count)
	}
	
	// 验证返回的是 test_scan_2 和 test_scan_3
	expectedKeys = []string{"test_scan_2", "test_scan_3"}
	foundKeys = make(map[string]bool)
	for _, kv := range resp.Kvs {
		foundKeys[string(kv.Key)] = true
	}
	
	for _, k := range expectedKeys {
		if !foundKeys[k] {
			t.Fatalf("键 %s 未在 WithFromKey+WithLimit 查询结果中找到", k)
		}
	}
	log.Println("WithFromKey+WithLimit 查询测试通过")
	
	// 清理测试数据
	_, err = cli.Delete(ctx, "test_scan_", clientv3.WithPrefix())
	if err != nil {
		t.Fatalf("清理测试数据失败: %v", err)
	}
	log.Println("清理测试数据成功")
	
	log.Println("所有 Scan 测试通过！")
}
