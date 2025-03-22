package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/otoolep/hraftd/etcdapi"
	"github.com/otoolep/hraftd/store"
)

// 创建并启动自己的etcdapi服务，返回服务实例
func setupEtcdService(t *testing.T, grpcPort, raftPort string) (*etcdapi.Service, *store.Store, *clientv3.Client) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "etcd-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// 创建并设置存储
	s := store.New(false)
	s.RaftDir = tempDir
	s.RaftBind = fmt.Sprintf("localhost:%s", raftPort)
	if err := s.Open(true, "test-node"); err != nil {
		t.Fatalf("failed to open store: %s", err.Error())
	}

	// 启动etcd API服务
	e := etcdapi.New(fmt.Sprintf("localhost:%s", grpcPort), s)
	if err := e.Start(); err != nil {
		t.Fatalf("failed to start etcd API service: %s", err.Error())
	}

	// 等待服务完全启动和Leader选举完成
	time.Sleep(5 * time.Second)

	// 等待确保节点成为Leader
	waitForLeader(t, s)

	// 创建连接到我们服务的客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("localhost:%s", grpcPort)},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建 etcd 客户端失败: %v", err)
	}

	return e, s, cli
}

// waitForLeader等待直到存储节点成为Leader
func waitForLeader(t *testing.T, s *store.Store) {
	maxWait := 10 * time.Second
	checkInterval := 500 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		// 尝试设置一个测试键，这只有Leader才能成功
		if err := s.Set("_test_leader_probe", "test"); err == nil {
			t.Logf("节点已成为Leader")
			// 删除测试键
			_ = s.Delete("_test_leader_probe")
			return
		}

		t.Logf("等待节点成为Leader...")
		time.Sleep(checkInterval)
	}

	t.Fatalf("节点未能在%v内成为Leader", maxWait)
}

// TestEtcdScanAPI 测试 Scan 操作，即前缀查询功能
func TestEtcdScanAPI(t *testing.T) {
	// 设置我们自己的etcd服务，使用端口62479
	e, s, cli := setupEtcdService(t, "62479", "62480")
	defer e.Close()
	defer cli.Close()

	// 清理可能存在的测试数据
	// 替换 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// 为空的注释行

	t.Log("测试开始: 首先检查Raft存储状态")

	// 检查存储节点状态
	for i := 0; i < 5; i++ {
		if s.IsLeader() {
			t.Logf("节点确认为Leader，继续测试")
			break
		}
		t.Logf("等待节点成为Leader，尝试 %d/5...", i+1)
		time.Sleep(1 * time.Second)
	}

	if !s.IsLeader() {
		t.Fatalf("节点未能成为Leader，无法执行测试")
	}

	// 测试键值对
	testData := map[string]string{
		"test_scan_1": "value1",
		"test_scan_2": "value2",
		"test_scan_3": "value3",
		"test_scan_4": "value4",
		"test_scan_5": "value5",
	}

	// 一次性清空所有测试数据并等待一段时间
	t.Log("清理所有测试数据")
	for k := range testData {
		if err := s.Delete(k); err != nil {
			t.Logf("删除键 %s 时出错: %v", k, err)
		} else {
			t.Logf("已删除键: %s", k)
		}
	}

	// 等待删除操作完成，确保Raft达成共识
	time.Sleep(3 * time.Second)

	// 验证数据是否被清理
	allDataAfterDelete := s.ListN(0, false)
	t.Log("删除后存储中的键值对:")
	for k, v := range allDataAfterDelete {
		if strings.HasPrefix(k, "test_scan_") {
			t.Logf("警告: 键 %q 仍然存在，值为 %q", k, v)
		}
	}

	t.Log("开始插入测试数据")
	// 逐一插入数据并验证
	for k, v := range testData {
		// 直接使用存储设置值
		t.Logf("直接插入键值对: %s -> %s", k, v)
		if err := s.Set(k, v); err != nil {
			t.Fatalf("设置键 %s 失败: %v", k, err)
		}

		// 验证键是否设置成功
		time.Sleep(500 * time.Millisecond) // 给Raft一些时间进行日志复制
		val, err := s.Get(k, false)
		if err != nil {
			t.Fatalf("获取键 %s 失败: %v", k, err)
		}
		if val != v {
			t.Fatalf("键 %s 的值不正确: 期望 %s, 实际 %s", k, v, val)
		}
		t.Logf("键 %s 设置成功，值为 %s", k, val)
	}

	// 等待Raft完成日志复制和应用
	t.Log("等待Raft日志复制，确保数据一致性...")
	time.Sleep(5 * time.Second)

	// 最后检查所有数据
	t.Log("最终检查存储状态:")
	finalData := s.ListN(0, false)
	for k, v := range finalData {
		if strings.HasPrefix(k, "test_scan_") {
			t.Logf("存储中的键值对: %s -> %s", k, v)
		}
	}

	// 确保所有测试数据都被正确存储
	for k, expectedVal := range testData {
		actualVal, err := s.Get(k, false)
		if err != nil {
			t.Fatalf("最终验证: 获取键 %s 失败: %v", k, err)
		}
		if actualVal != expectedVal {
			t.Fatalf("最终验证: 键 %s 的值不正确: 期望 %s, 实际 %s", k, expectedVal, actualVal)
		}
		t.Logf("最终验证: 键 %s 验证成功", k)
	}

	// 测试 Range 功能
	t.Log("测试前缀查询功能")

	// 创建前缀查询上下文
	prefixCtx, prefixCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer prefixCancel()

	// 执行前缀查询
	t.Logf("执行前缀查询: test_scan_")
	resp, err := cli.Get(prefixCtx, "test_scan_", clientv3.WithPrefix())
	if err != nil {
		t.Fatalf("前缀查询失败: %v", err)
	}

	// 记录查询结果
	t.Logf("前缀查询返回 %d 条结果", resp.Count)
	for i, kv := range resp.Kvs {
		t.Logf("结果 %d: %s -> %s", i, string(kv.Key), string(kv.Value))
	}

	// 验证是否返回了所有测试数据
	if resp.Count != int64(len(testData)) {
		t.Fatalf("前缀查询返回结果数量不正确: 期望 %d, 实际 %d", len(testData), resp.Count)
	}

	// 验证返回的每个键值对是否正确
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)
		expectedValue, exists := testData[key]
		if !exists {
			t.Fatalf("前缀查询返回了意外的键: %s", key)
		}
		if value != expectedValue {
			t.Fatalf("键 %s 的值不正确: 期望 %s, 实际 %s", key, expectedValue, value)
		}
	}

	t.Log("测试通过")
}

func TestEtcdAPI(t *testing.T) {
	// 设置我们自己的etcd服务，使用端口62379
	e, _, cli := setupEtcdService(t, "62379", "62380")
	defer e.Close()
	defer cli.Close()

	// 测试 Put 操作
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := cli.Put(ctx, "test_key", "test_value")
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
