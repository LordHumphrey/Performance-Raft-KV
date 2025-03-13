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
		value, err := s.store.Get(key)
		if err != nil {
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

		// Check if this is a prefix query (e.g., "foo" with range_end "foo\0")
		isPrefixQuery := false
		if len(rangeEnd) > 0 && rangeEnd[len(rangeEnd)-1] == 0 {
			isPrefixQuery = true
			rangeEnd = rangeEnd[:len(rangeEnd)-1]
		}

		// For simplicity, we'll just handle prefix queries
		// In a real implementation, we would need to handle all range types
		if isPrefixQuery {
			// Since our store doesn't support listing keys, we can't properly implement this
			// In a real implementation, we would scan all keys with the prefix
			// For now, we'll just return an empty result
			// TODO: Implement proper prefix scanning
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
