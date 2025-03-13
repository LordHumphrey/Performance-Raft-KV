package etcdapi

import (
	"context"

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

	// Set the key-value pair in the store
	if err := s.store.Set(key, value); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
