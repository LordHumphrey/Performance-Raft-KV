package etcdapi

import (
	"context"

	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Txn is not implemented in this version.
func (s *Service) Txn(ctx context.Context, req *pb.TxnRequest) (*pb.TxnResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Txn not implemented")
}

// Compact is not implemented in this version.
func (s *Service) Compact(ctx context.Context, req *pb.CompactionRequest) (*pb.CompactionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Compact not implemented")
}
