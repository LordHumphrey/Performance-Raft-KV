package etcdapi

import (
	"context"

	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Range implements the etcd v3 Range API.
func (s *Service) Range(ctx context.Context, req *pb.RangeRequest) (*pb.RangeResponse, error) {
	if len(req.Key) == 0 {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}

	// Convert byte slice to string
	key := string(req.Key)
	
	// Handle range query
	var kvs []*mvccpb.KeyValue
	var count int64
	
	if len(req.RangeEnd) == 0 {
		// Single key lookup
		// 默认不解码
		value, err := s.store.Get(key, false)
		if err != nil {
			// 如果是键不存在的错误，返回空结果而不是错误
			if err.Error() == "key not found" {
				// 返回空结果
				return &pb.RangeResponse{
					Header: &pb.ResponseHeader{
						ClusterId: 1,
						MemberId:  1,
						Revision:  1,
						RaftTerm:  1,
					},
					Kvs:   kvs,
					Count: 0,
				}, nil
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		
		// If key exists, add to result
		if value != "" {
			kv := &mvccpb.KeyValue{
				Key:   req.Key,
				Value: []byte(value),
				// Note: In a real etcd implementation, these would be actual revision numbers
				CreateRevision: 1,
				ModRevision:    1,
				Version:        1,
			}
			kvs = append(kvs, kv)
			count = 1
		}
	} else {
		// Range query - we need to implement a simple prefix match
		// This is a simplified implementation that doesn't handle all etcd range cases
		rangeEnd := string(req.RangeEnd)
		
		// 确定是否是前缀查询
		isPrefixQuery := false
		if len(rangeEnd) > 0 && rangeEnd[len(rangeEnd)-1] == 0 {
			isPrefixQuery = true
		}
		
		// WithRange(), WithFromKey(), WithPrefix() 支持
		// WithRange(): key到rangeEnd
		// WithFromKey(): key到最大键
		// WithPrefix(): 前缀查询
		
		var prefixResults map[string]string
		var limit int64 = 0 // 默认不限制返回数量
		
		// 处理限制返回数量
		if req.Limit > 0 {
			limit = req.Limit
		}
		
		if isPrefixQuery {
			// 前缀查询
			prefix := key
			if len(prefix) > 0 && prefix[len(prefix)-1] == 0 {
				prefix = prefix[:len(prefix)-1]
			}
			
			// 使用ListPrefix获取前缀结果
			prefixResults = s.store.ListPrefix(prefix, limit, false)
		} else if len(rangeEnd) > 0 {
			// 普通范围查询 (WithRange)
			// 在实际的etcd实现中，这里需要处理所有在key和rangeEnd之间的键
			// 由于我们的简化实现，这里只返回key的值
			value, err := s.store.Get(key, false)
			if err == nil && value != "" {
				kv := &mvccpb.KeyValue{
					Key:   req.Key,
					Value: []byte(value),
					CreateRevision: 1,
					ModRevision:    1,
					Version:        1,
				}
				kvs = append(kvs, kv)
				count = 1
			}
			
			// 注意：这是一个简化实现，没有处理rangeEnd
		} else if string(req.Key) == "\x00" {
			// WithFromKey()：从最小键到最大键
			// 简化实现：返回所有键(限制数量)
			prefixResults = s.store.ListN(int(limit), false)
		}
		
		// 处理前缀查询结果
		if prefixResults != nil && len(prefixResults) > 0 {
			for k, v := range prefixResults {
				kv := &mvccpb.KeyValue{
					Key:   []byte(k),
					Value: []byte(v),
					CreateRevision: 1,
					ModRevision:    1,
					Version:        1,
				}
				kvs = append(kvs, kv)
			}
			count = int64(len(kvs))
		}
	}
	
	// Build response
	resp := &pb.RangeResponse{
		Header: &pb.ResponseHeader{
			// In a real implementation, these would be actual cluster information
			ClusterId: 1,
			MemberId:  1,
			Revision:  1,
			RaftTerm:  1,
		},
		Kvs:   kvs,
		Count: count,
	}
	
	return resp, nil
}
