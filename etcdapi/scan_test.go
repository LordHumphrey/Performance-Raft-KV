package etcdapi_test

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"google.golang.org/grpc"
)

// 为测试自定义的Service结构体，模拟实际的Service
type testService struct {
	addr  string
	store testStore
	srv   *grpc.Server
}

// testStore接口定义了存储需要实现的方法
type testStore interface {
	Get(key string, decode bool) (string, error)
	ListPrefix(prefix string, limit int64, decode bool) map[string]string
	ListN(n int, decode bool) map[string]string
}

// 为测试创建一个带有自定义存储的服务
func newTestService(addr string, store *testStoreImpl) *testService {
	return &testService{
		addr:  addr,
		store: store,
	}
}

// 一个简化的存储，仅用于测试
type testStoreImpl struct {
	m  map[string]string
	mu sync.Mutex
}

func newTestStore() *testStoreImpl {
	return &testStoreImpl{
		m: make(map[string]string),
	}
}

func (s *testStoreImpl) Get(key string, decode bool) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, exists := s.m[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}

	return value, nil
}

// Set 设置键值对
func (s *testStoreImpl) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
	return nil
}

// Delete 删除指定的键
func (s *testStoreImpl) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
	return nil
}

// DirectSet 直接设置键值对，用于测试数据准备，不返回错误
func (s *testStoreImpl) DirectSet(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

func (s *testStoreImpl) GetPrefix(prefix string, limit int, decode bool) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]string)
	count := 0

	for k, v := range s.m {
		// 检查键是否具有指定的前缀
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result[k] = v

			count++
			// 如果设置了限制且已达到限制，则中断循环
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	return result, nil
}

// 实现为testStoreImpl添加ListPrefix和ListN方法，使其与store.Store接口兼容
func (s *testStoreImpl) ListPrefix(prefix string, limit int64, decode bool) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]string)
	count := int64(0)

	for k, v := range s.m {
		// 检查键是否有指定前缀
		if strings.HasPrefix(k, prefix) {
			result[k] = v

			count++
			// 如果达到限制数量，中断循环
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	return result
}

func (s *testStoreImpl) ListN(n int, decode bool) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]string)
	count := 0

	for k, v := range s.m {
		result[k] = v

		count++
		// 如果达到限制数量，中断循环
		if n > 0 && count >= n {
			break
		}
	}

	return result
}

// 实现Range方法用于测试
func (s *testService) Range(ctx context.Context, req *etcdserverpb.RangeRequest) (*etcdserverpb.RangeResponse, error) {
	// 提取关键参数
	key := string(req.Key)
	rangeEnd := string(req.RangeEnd)
	limit := req.Limit

	// 初始化响应
	resp := &etcdserverpb.RangeResponse{
		Header: &etcdserverpb.ResponseHeader{
			ClusterId: 1,
			MemberId:  1,
			Revision:  1,
			RaftTerm:  1,
		},
	}

	// 根据不同的查询类型处理
	if len(req.RangeEnd) == 0 {
		// 单键查询
		value, err := s.store.Get(key, false)
		if err != nil {
			if err.Error() == "key not found" {
				return resp, nil // 返回空结果
			}
			return nil, err
		}

		// 构建KeyValue
		kv := &mvccpb.KeyValue{
			Key:            []byte(key),
			Value:          []byte(value),
			CreateRevision: 1,
			ModRevision:    1,
			Version:        1,
		}

		resp.Kvs = append(resp.Kvs, kv)
		resp.Count = 1
	} else {
		// 范围查询
		isPrefixQuery := false
		isFromKeyQuery := false

		// 检查是否是前缀查询
		if len(rangeEnd) > 0 {
			// WithPrefix() 将rangeEnd设置为前缀后面的下一个字节
			if len(key) > 0 && len(rangeEnd) == len(key)+1 &&
				rangeEnd[:len(key)] == key && rangeEnd[len(key)] == 0 {
				isPrefixQuery = true
			}
		}

		// 检查是否是FromKey查询
		if rangeEnd == "\x00" {
			isFromKeyQuery = true
		}

		var results map[string]string

		if isPrefixQuery {
			// 前缀查询
			results = s.store.ListPrefix(key, limit, false)
		} else if isFromKeyQuery {
			// WithFromKey() 查询 - 获取所有大于等于key的键
			allResults := s.store.ListN(0, false)
			results = make(map[string]string)

			// 保存排序后的键
			var keys []string
			for k := range allResults {
				if k >= key {
					keys = append(keys, k)
				}
			}

			// 对键进行排序
			sort.Strings(keys)

			// 添加到结果中
			count := int64(0)
			for _, k := range keys {
				results[k] = allResults[k]
				count++
				if limit > 0 && count >= limit {
					break
				}
			}
		} else {
			// 普通范围查询
			allResults := s.store.ListN(0, false)
			results = make(map[string]string)

			// 保存排序后的键
			var keys []string
			for k := range allResults {
				if k >= key && k < rangeEnd {
					keys = append(keys, k)
				}
			}

			// 对键进行排序
			sort.Strings(keys)

			// 添加到结果中
			count := int64(0)
			for _, k := range keys {
				results[k] = allResults[k]
				count++
				if limit > 0 && count >= limit {
					break
				}
			}
		}

		// 构建响应
		for k, v := range results {
			kv := &mvccpb.KeyValue{
				Key:            []byte(k),
				Value:          []byte(v),
				CreateRevision: 1,
				ModRevision:    1,
				Version:        1,
			}
			resp.Kvs = append(resp.Kvs, kv)
		}

		resp.Count = int64(len(resp.Kvs))
	}

	return resp, nil
}

// 测试Scan API的实现
func TestScan(t *testing.T) {
	// 创建一个测试存储实例
	ts := newTestStore()

	// 设置一些测试数据
	keys := []string{
		"test:key1",
		"test:key2",
		"test:key3",
		"other:key1",
		"other:key2",
	}

	values := []string{
		`{"field1":"value1"}`,
		`{"field1":"value2"}`,
		`{"field1":"value3"}`,
		`{"field1":"value4"}`,
		`{"field1":"value5"}`,
	}

	// 将测试数据添加到存储中
	for i, key := range keys {
		ts.DirectSet(key, values[i])
	}

	// 创建etcd API服务
	addr := "localhost:22380" // 使用一个可能不会冲突的端口
	service := newTestService(addr, ts)
	err := service.Start()
	if err != nil {
		t.Fatalf("无法启动etcd API服务: %v", err)
	}
	defer service.Close()

	// 等待服务启动
	time.Sleep(1 * time.Second)

	// 创建etcd客户端
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{addr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("无法创建etcd客户端: %v", err)
	}
	defer client.Close()

	// 测试场景1: 获取单个键
	t.Run("GetSingleKey", func(t *testing.T) {
		resp, err := client.Get(context.Background(), "test:key1")
		if err != nil {
			t.Fatalf("获取单个键失败: %v", err)
		}

		if len(resp.Kvs) != 1 {
			t.Fatalf("期望获取1个键值对，实际获取到%d个", len(resp.Kvs))
		}

		if string(resp.Kvs[0].Key) != "test:key1" {
			t.Errorf("键不匹配: 期望 %s, 实际 %s", "test:key1", string(resp.Kvs[0].Key))
		}

		if string(resp.Kvs[0].Value) != values[0] {
			t.Errorf("值不匹配: 期望 %s, 实际 %s", values[0], string(resp.Kvs[0].Value))
		}
	})

	// 测试场景2: 使用前缀查询
	t.Run("ScanWithPrefix", func(t *testing.T) {
		// 使用前缀 "test:" 查询
		resp, err := client.Get(context.Background(), "test:", clientv3.WithPrefix())
		if err != nil {
			t.Fatalf("前缀查询失败: %v", err)
		}

		if len(resp.Kvs) != 3 {
			t.Fatalf("期望获取3个键值对，实际获取到%d个", len(resp.Kvs))
		}

		// 验证返回的键值对是否正确
		expectedKeys := map[string]bool{
			"test:key1": true,
			"test:key2": true,
			"test:key3": true,
		}

		for _, kv := range resp.Kvs {
			key := string(kv.Key)
			if !expectedKeys[key] {
				t.Errorf("意外的键: %s", key)
			}
		}
	})

	// 测试场景3: 使用限制数量的前缀查询
	t.Run("ScanWithPrefixAndLimit", func(t *testing.T) {
		// 使用前缀 "test:" 查询，限制返回2个结果
		resp, err := client.Get(context.Background(), "test:", clientv3.WithPrefix(), clientv3.WithLimit(2))
		if err != nil {
			t.Fatalf("带限制的前缀查询失败: %v", err)
		}

		t.Logf("ScanWithPrefixAndLimit: 找到 %d 个结果", len(resp.Kvs))
		for i, kv := range resp.Kvs {
			t.Logf("结果 %d: 键=%s", i, string(kv.Key))
		}

		if len(resp.Kvs) != 2 {
			t.Fatalf("期望获取2个键值对，实际获取到%d个", len(resp.Kvs))
		}
	})

	// 测试场景4: 范围查询
	t.Run("RangeQuery", func(t *testing.T) {
		// 查询从 "test:key1" 到 "test:key3" 的范围（不包括 "test:key3"）
		resp, err := client.Get(context.Background(), "test:key1", clientv3.WithRange("test:key3"))
		if err != nil {
			t.Fatalf("范围查询失败: %v", err)
		}

		if len(resp.Kvs) != 2 {
			t.Fatalf("期望获取2个键值对，实际获取到%d个", len(resp.Kvs))
		}

		// 验证返回的键值对是否正确
		expectedKeys := map[string]bool{
			"test:key1": true,
			"test:key2": true,
		}

		for _, kv := range resp.Kvs {
			key := string(kv.Key)
			if !expectedKeys[key] {
				t.Errorf("意外的键: %s", key)
			}
		}
	})

	// 测试场景5: 模拟YCSB Scan操作
	t.Run("YCSBScanOperation", func(t *testing.T) {
		// 准备一组顺序键
		scanKeys := []string{
			"scan:key1",
			"scan:key2",
			"scan:key3",
			"scan:key4",
			"scan:key5",
		}

		scanValues := []string{
			`{"field1":"scan1"}`,
			`{"field1":"scan2"}`,
			`{"field1":"scan3"}`,
			`{"field1":"scan4"}`,
			`{"field1":"scan5"}`,
		}

		// 添加测试数据
		for i, key := range scanKeys {
			ts.DirectSet(key, scanValues[i])
		}

		// 显示所有存储中的键值对
		allData, _ := ts.GetPrefix("", 0, false)
		t.Logf("测试存储中的所有键值对: %v", allData)

		// 测试方法1：使用WithFromKey扫描
		resp, err := client.Get(context.Background(), "scan:key2", clientv3.WithFromKey(), clientv3.WithLimit(3))
		if err != nil {
			t.Fatalf("使用WithFromKey扫描失败: %v", err)
		}

		t.Logf("方法1扫描结果数量: %d", len(resp.Kvs))
		for i, kv := range resp.Kvs {
			t.Logf("方法1结果 %d: 键=%s, 值=%s", i, string(kv.Key), string(kv.Value))
		}

		// 测试方法2：使用WithRange扫描
		// 计算范围结束键
		endKey := clientv3.GetPrefixRangeEnd("scan:key2")
		t.Logf("方法2范围结束键: %q", endKey)

		resp2, err := client.Get(context.Background(), "scan:key2", clientv3.WithRange(endKey), clientv3.WithLimit(3))
		if err != nil {
			t.Fatalf("使用WithRange扫描失败: %v", err)
		}

		t.Logf("方法2扫描结果数量: %d", len(resp2.Kvs))
		for i, kv := range resp2.Kvs {
			t.Logf("方法2结果 %d: 键=%s, 值=%s", i, string(kv.Key), string(kv.Value))
		}

		// 测试方法3：手动扫描
		// 获取所有键并找出大于等于"scan:key2"的键
		var keys []string

		// 获取所有键
		for k := range allData {
			// 找出大于等于起始键的键
			if k >= "scan:key2" {
				keys = append(keys, k)
			}
		}

		// 从所有键中手动选择3个
		sort.Strings(keys)
		if len(keys) > 3 {
			keys = keys[:3]
		}
		t.Logf("方法3手动扫描找到的键: %v", keys)
	})

	// 测试场景6: 简单的Scan测试
	t.Run("SimpleScanTest", func(t *testing.T) {
		// 创建简单的测试数据
		simpleKeys := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		simpleValues := []string{"value-a", "value-b", "value-c", "value-d", "value-e", "value-f", "value-g", "value-h"}

		// 重置测试存储
		ts.m = make(map[string]string)
		for i, key := range simpleKeys {
			ts.DirectSet(key, simpleValues[i])
		}

		// 显示所有存储中的键值对
		allData, _ := ts.GetPrefix("", 0, false)
		t.Logf("简单测试存储中的所有键值对: %v", allData)

		// 测试从"c"开始扫描3个键
		resp, err := client.Get(context.Background(), "c", clientv3.WithFromKey(), clientv3.WithLimit(3))
		if err != nil {
			t.Fatalf("简单扫描失败: %v", err)
		}

		t.Logf("简单扫描结果数量: %d", len(resp.Kvs))
		for i, kv := range resp.Kvs {
			t.Logf("结果 %d: 键=%s, 值=%s", i, string(kv.Key), string(kv.Value))
		}

		if len(resp.Kvs) != 3 {
			t.Fatalf("期望获取3个键值对，实际获取到%d个", len(resp.Kvs))
		}

		// 验证键的顺序是否正确
		expectedKeys := []string{"c", "d", "e"}
		for i, kv := range resp.Kvs {
			if string(kv.Key) != expectedKeys[i] {
				t.Errorf("键顺序不正确: 位置 %d 期望 %s, 实际 %s", i, expectedKeys[i], string(kv.Key))
			}
		}

		// 测试范围查询
		// 从"c"到"d"的范围查询（不包括"d"）
		rangeEnd := "d"
		t.Logf("range结束键: %q", rangeEnd)
		resp2, err := client.Get(context.Background(), "c", clientv3.WithRange(rangeEnd))
		if err != nil {
			t.Fatalf("带range的简单扫描失败: %v", err)
		}

		t.Logf("带range的简单扫描结果数量: %d", len(resp2.Kvs))
		for i, kv := range resp2.Kvs {
			t.Logf("结果 %d: 键=%s, 值=%s", i, string(kv.Key), string(kv.Value))
		}
	})

	t.Log("所有扫描测试通过")
}

// Start 启动服务
func (s *testService) Start() error {
	// 创建gRPC服务器
	s.srv = grpc.NewServer()

	// 注册KV服务
	etcdserverpb.RegisterKVServer(s.srv, s)

	// 启动监听
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	// 启动服务器
	go func() {
		if err := s.srv.Serve(ln); err != nil {
			fmt.Printf("服务器错误: %v\n", err)
		}
	}()

	return nil
}

// Close 关闭服务
func (s *testService) Close() {
	if s.srv != nil {
		s.srv.GracefulStop()
	}
}

// Put 实现Put方法
func (s *testService) Put(ctx context.Context, req *etcdserverpb.PutRequest) (*etcdserverpb.PutResponse, error) {
	// 不需要完整实现
	return &etcdserverpb.PutResponse{}, nil
}

// DeleteRange 实现DeleteRange方法
func (s *testService) DeleteRange(ctx context.Context, req *etcdserverpb.DeleteRangeRequest) (*etcdserverpb.DeleteRangeResponse, error) {
	// 不需要完整实现
	return &etcdserverpb.DeleteRangeResponse{}, nil
}

// Txn 实现Txn方法
func (s *testService) Txn(ctx context.Context, req *etcdserverpb.TxnRequest) (*etcdserverpb.TxnResponse, error) {
	// 不需要完整实现
	return &etcdserverpb.TxnResponse{}, nil
}

// Compact 实现Compact方法
func (s *testService) Compact(ctx context.Context, req *etcdserverpb.CompactionRequest) (*etcdserverpb.CompactionResponse, error) {
	// 不需要完整实现
	return &etcdserverpb.CompactionResponse{}, nil
}
