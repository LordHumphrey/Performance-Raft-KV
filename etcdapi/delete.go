package etcdapi

import (
	"context"

	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DeleteRange implements the etcd v3 DeleteRange API.
func (s *Service) DeleteRange(ctx context.Context, req *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	if len(req.Key) == 0 {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}

	// Convert byte slice to string
	key := string(req.Key)

	// For now, we only support deleting a single key
	// In a real implementation, we would handle range deletes
	if len(req.RangeEnd) > 0 {
		// This is a simplified implementation that doesn't handle range deletes
		// In a real implementation, we would need to handle all range types
		return nil, status.Error(codes.Unimplemented, "range delete not implemented")
	}

	// Delete the key from the store
	if err := s.store.Delete(key); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Build response
	resp := &pb.DeleteRangeResponse{
		Header: &pb.ResponseHeader{
			// In a real implementation, these would be actual cluster information
			ClusterId: 1,
			MemberId:  1,
			Revision:  1,
			RaftTerm:  1,
		},
		// We always delete at most one key in this implementation
		Deleted: 1,
		// In a real etcd implementation, we would return the previous values if prev_kv was set
		// But our current store implementation doesn't support getting the previous values
	}

	return resp, nil
}
