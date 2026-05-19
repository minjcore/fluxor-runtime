package queue

import (
	"context"
	"fmt"
	"net"

	"github.com/fluxorio/fluxor/pkg/cache"
	pb "github.com/fluxorio/fluxor/proto/fluxor/queue"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer provides gRPC server functionality for queue RPC operations
type GRPCServer struct {
	pb.UnimplementedQueueRPCServer
	cache   cache.Cache
	server  *grpc.Server
	address string
}

// NewGRPCServer creates a new gRPC server
// Fail-fast: Validates inputs
func NewGRPCServer(cache cache.Cache, address string) (*GRPCServer, error) {
	if cache == nil {
		return nil, fmt.Errorf("cache cannot be nil")
	}
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	return &GRPCServer{
		cache:   cache,
		address: address,
		server:  grpc.NewServer(),
	}, nil
}

// GetCache implements the QueueRPC.GetCache RPC method
func (s *GRPCServer) GetCache(ctx context.Context, req *pb.GetCacheRequest) (*pb.GetCacheResponse, error) {
	// Fail-fast: Validate cache key
	if req.CacheKey == "" {
		return nil, status.Error(codes.InvalidArgument, "cache_key cannot be empty")
	}

	// Get data from cache
	cacheData, err := s.cache.Get(ctx, req.CacheKey)
	if err != nil {
		// Cache miss or error - return error response
		return &pb.GetCacheResponse{
			Success: false,
			Error:   fmt.Sprintf("cache error for key %s: %v", req.CacheKey, err),
		}, nil
	}

	// Return success response
	return &pb.GetCacheResponse{
		Success: true,
		Data:    cacheData,
	}, nil
}

// Start starts the gRPC server
// Fail-fast: Validates state
func (s *GRPCServer) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if s.server == nil {
		return fmt.Errorf("server not initialized")
	}

	// Register the service
	pb.RegisterQueueRPCServer(s.server, s)

	// Create listener
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}

	// Start server in a goroutine
	go func() {
		if err := s.server.Serve(lis); err != nil {
			// Log error if context is not cancelled
			select {
			case <-ctx.Done():
				// Context cancelled, this is expected
			default:
				// Unexpected error
			}
		}
	}()

	return nil
}

// Stop stops the gRPC server gracefully
func (s *GRPCServer) Stop() error {
	if s.server != nil {
		s.server.GracefulStop()
	}
	return nil
}

// Close closes the gRPC server (alias for Stop for consistency with other interfaces)
func (s *GRPCServer) Close() error {
	return s.Stop()
}
