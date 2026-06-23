/*
Package grpc provides the gRPC transport layer for the Cloud service.

StartServer initializes a gRPC server with keepalive, logging, recovery, and
timeout interceptors, registers the PolicyEvaluationService, and listens on
the configured port.
*/
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

func StartServer(port int, handler pb.PolicyEvaluationServiceServer, logger *slog.Logger) error {
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		MaxConnectionAge:  30 * time.Minute,
		Time:              60 * time.Second,
		Timeout:           20 * time.Second,
	}

	srv := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.ChainUnaryInterceptor(
			loggingInterceptor(logger),
			recoveryInterceptor(logger),
			timeoutInterceptor(30*time.Second),
		),
	)

	pb.RegisterPolicyEvaluationServiceServer(srv, handler)

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("grpc listen on %s: %w", addr, err)
	}

	logger.Info("grpc server started", "addr", addr)
	return srv.Serve(ln)
}

func loggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		logger.Info("grpc call",
			"method", info.FullMethod,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return resp, err
	}
}

func recoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered in grpc handler", "method", info.FullMethod, "panic", r)
				err = status.Errorf(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

func timeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(ctx, req)
	}
}
