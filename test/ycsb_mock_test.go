package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/otoolep/hraftd/store"
	"github.com/otoolep/hraftd/etcdapi"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 模拟YCSB属性
type mockProperties struct {
	properties map[string]interface{}
}

func newMockProperties() *mockProperties {
	return &mockProperties{
		properties: make(map[string]interface{}),
	}
}

func (p *mockProperties) GetString(key string, defaultValue string) string {
	if val, ok := p.properties[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

func (p *mockProperties) GetDuration(key string, defaultValue time.Duration) time.Duration {
	if val, ok := p.properties[key]; ok {
		if durVal, ok := val.(time.Duration); ok {
			return durVal
		}
	}
	return defaultValue
}

func (p *mockProperties) GetBool(key string, defaultValue bool) bool {
	if val, ok := p.properties[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

func (p *mockProperties) MustGetString(key string) string {
	if val, ok := p.properties[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	panic(fmt.Sprintf("必须的属性 %s 不存在", key))
}

func (p *mockProperties) Set(key string, value interface{}) {
	p.properties[key] = value
}

// 模拟YCSB的DB接口和实现，参考提供的etcdDB代码
type mockDB interface {
	Close() error
	Read(ctx context.Context, table string, key string, fields []string) (map[string][]byte, error)
	Scan(ctx context.Context, table string, startKey string, count int, fields []string) ([]map[string][]byte, error)
	Update(ctx context.Context, table string, key string, values map[string][]byte) error
	Insert(ctx context.Context, table string, key string, values map[string][]byte) error
	Delete(ctx context.Context, table string, key string) error
}

type mockEtcdDB struct {
	p      *mockProperties
	client *clientv3.Client
}

func newMockEtcdDB(addr string) (*mockEtcdDB, error) {
	p := newMockProperties()
	p.Set("endpoints", addr)
	p.Set("dial_timeout", 5*time.Second)

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{addr},
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	return &mockEtcdDB{
		p:      p,
		client: client,
	}, nil
}

func (db *mockEtcdDB) Close() error {
	return db.client.Close()
}

func getRowKey(table string, key string) string {
	return fmt.Sprintf("%s:%s", table, key)
}

func (db *mockEtcdDB) Read(ctx context.Context, table string, key string, _ []string) (map[string][]byte, error) {
	rkey := getRowKey(table, key)
	var options []clientv3.OpOption
	if db.p.GetBool("serializable_reads", false) {
		options = append(options, clientv3.WithSerializable())
	}

	value, err := db.client.Get(ctx, rkey, options...)
	if err != nil {
		return nil, err
	}

	if value.Count == 0 {
		return nil, fmt.Errorf("could not find value for key [%s]", rkey)
	}

	var r map[string][]byte
	err = json.Unmarshal(value.Kvs[0].Value, &r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (db *mockEtcdDB) Scan(ctx context.Context, table string, startKey string, count int, _ []string) ([]map[string][]byte, error) {
	res := make([]map[string][]byte, 0, count)
	rkey := getRowKey(table, startKey)
	options := []clientv3.OpOption{clientv3.WithFromKey(), clientv3.WithLimit(int64(count))}

	if db.p.GetBool("serializable_reads", false) {
		options = append(options, clientv3.WithSerializable())
	}

	values, err := db.client.Get(ctx, rkey, options...)
	if err != nil {
		return nil, err
	}

	if values.Count == 0 {
		return res, nil
	}

	for _, v := range values.Kvs {
		var r map[string][]byte
		err = json.Unmarshal(v.Value, &r)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (db *mockEtcdDB) Update(ctx context.Context, table string, key string, values map[string][]byte) error {
	rkey := getRowKey(table, key)
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}

	_, err = db.client.Put(ctx, rkey, string(data))
	if err != nil {
		return err
	}

	return nil
}

func (db *mockEtcdDB) Insert(ctx context.Context, table string, key string, values map[string][]byte) error {
	return db.Update(ctx, table, key, values)
}

func (db *mockEtcdDB) Delete(ctx context.Context, table string, key string) error {
	_, err := db.client.Delete(ctx, getRowKey(table, key))
	if err != nil {
		return err
	}
	return nil
}

// 测试模拟YCSB的Scan操作
func TestMockYCSBScan(t *testing.T) {
	// 创建一个内存存储实例
	s := store.New(true)

	// 创建etcd API服务
	addr := "localhost:22379" // 使用一个可能不会冲突的端口
	service := etcdapi.New(addr, s)
	err := service.Start()
	if err != nil {
		t.Fatalf("无法启动etcd API服务: %v", err)
	}
	defer service.Close()

	// 等待服务启动
	time.Sleep(1 * time.Second)

	// 创建模拟的YCSB etcdDB
	db, err := newMockEtcdDB(addr)
	if err != nil {
		t.Fatalf("无法创建模拟YCSB etcdDB: %v", err)
	}
	defer db.Close()

	// 准备测试数据
	table := "usertable"
	keyPrefix := "user"
	recordCount := 10

	// 插入测试数据
	for i := 0; i < recordCount; i++ {
		key := fmt.Sprintf("%s%d", keyPrefix, i)
		values := map[string][]byte{
			"field0": []byte(fmt.Sprintf("value%d-0", i)),
			"field1": []byte(fmt.Sprintf("value%d-1", i)),
			"field2": []byte(fmt.Sprintf("value%d-2", i)),
		}

		err := db.Insert(context.Background(), table, key, values)
		if err != nil {
			t.Fatalf("插入数据失败: %v", err)
		}
	}

	// 测试1: 从第一个键开始扫描5条记录
	t.Run("ScanFromStart", func(t *testing.T) {
		startKey := "user0"
		count := 5

		results, err := db.Scan(context.Background(), table, startKey, count, nil)
		if err != nil {
			t.Fatalf("扫描操作失败: %v", err)
		}

		if len(results) != count {
			t.Fatalf("期望获取%d条记录，实际获取到%d条", count, len(results))
		}

		// 验证返回的数据
		for i, res := range results {
			expectedField0 := fmt.Sprintf("value%d-0", i)
			if string(res["field0"]) != expectedField0 {
				t.Errorf("记录%d的field0不匹配: 期望 %s, 实际 %s", i, expectedField0, string(res["field0"]))
			}
		}
	})

	// 测试2: 从中间键开始扫描
	t.Run("ScanFromMiddle", func(t *testing.T) {
		startKey := "user5"
		count := 3

		results, err := db.Scan(context.Background(), table, startKey, count, nil)
		if err != nil {
			t.Fatalf("扫描操作失败: %v", err)
		}

		if len(results) != count {
			t.Fatalf("期望获取%d条记录，实际获取到%d条", count, len(results))
		}

		// 验证返回的数据
		for i, res := range results {
			expectedIndex := i + 5
			expectedField0 := fmt.Sprintf("value%d-0", expectedIndex)
			if string(res["field0"]) != expectedField0 {
				t.Errorf("记录%d的field0不匹配: 期望 %s, 实际 %s", i, expectedField0, string(res["field0"]))
			}
		}
	})

	// 测试3: 扫描超过可用记录数
	t.Run("ScanBeyondAvailable", func(t *testing.T) {
		startKey := "user8"
		count := 5 // 只应该返回2条记录 (user8, user9)

		results, err := db.Scan(context.Background(), table, startKey, count, nil)
		if err != nil {
			t.Fatalf("扫描操作失败: %v", err)
		}

		expectedCount := 2 // 只有user8和user9两条记录
		if len(results) != expectedCount {
			t.Fatalf("期望获取%d条记录，实际获取到%d条", expectedCount, len(results))
		}
	})

	// 测试4: 扫描不存在的键
	t.Run("ScanNonExistentKey", func(t *testing.T) {
		startKey := "userX"
		count := 5

		results, err := db.Scan(context.Background(), table, startKey, count, nil)
		if err != nil {
			t.Fatalf("扫描操作失败: %v", err)
		}

		if len(results) != 0 {
			t.Fatalf("期望获取0条记录，实际获取到%d条", len(results))
		}
	})

	t.Log("所有YCSB模拟扫描测试通过")
}
