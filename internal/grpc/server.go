package grpc

import (
	"context"

	pb "github.com/Pavan-Rana/rate-limiter/proto"
)

type Limiter interface {
	AllowRequest(ctx context.Context, apiKey string) (bool, error)
}

type Server struct {
	pb.UnimplementedRateLimiterServer
	limiter Limiter
}

func New(l Limiter) *Server {
	return &Server{limiter: l}
}

func (s *Server) AllowRequest(ctx context.Context, req *pb.AllowRequestMessage) (*pb.AllowResponse, error) {
	allowed, err := s.limiter.AllowRequest(ctx, req.ApiKey)
	if err != nil {
		return &pb.AllowResponse{Allowed: true, Reason: "fail-open: " + err.Error()}, nil
	}

	reason := "allowed"
	if !allowed {
		reason = "rate limit exceeded"
	}
	return &pb.AllowResponse{Allowed: allowed, Reason: reason}, nil
}
