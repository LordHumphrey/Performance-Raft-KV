package etcdapi

import (
	"context"
	"fmt"

	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Put implements the etcd v3 Put API.
func (s *Service) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	if len(req.Key) == 0 {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}

	// Convert byte slices to strings
	key := string(req.Key)
	value := string(req.Value)

	fmt.Printf("DEBUG: Put操作 - key=%q, value=%q\n", key, value)

	// Set the key-value pair in the store
	if err := s.store.Set(key, value); err != nil {
		fmt.Printf("DEBUG: Set操作失败 - key=%q, value=%q, error=%v\n", key, value, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 验证键值对是否已成功存储
	storedValue, err := s.store.Get(key, false)
	if err != nil {
		fmt.Printf("DEBUG: 验证Set操作失败 - 无法获取key=%q: %v\n", key, err)
	} else if storedValue != value {
		fmt.Printf("DEBUG: 验证Set操作失败 - key=%q, 期望值=%q, 实际值=%q\n", key, value, storedValue)
	} else {
		fmt.Printf("DEBUG: 验证Set操作成功 - key=%q 已正确存储\n", key)
	}

	// Build response
	resp := &pb.PutResponse{
		Header: &pb.ResponseHeader{
			// In a real implementation, these would be actual cluster information
			ClusterId: 1,
			MemberId:  1,
			Revision:  1,
			RaftTerm:  1,
		},
		// In a real etcd implementation, we would return the previous value if prev_kv was set
		// But our current store implementation doesn't support atomically getting the previous value
	}

	return resp, nil
}
