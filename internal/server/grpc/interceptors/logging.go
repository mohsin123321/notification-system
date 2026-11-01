package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

// UnaryLogging returns a gRPC unary server interceptor that logs each call.
func UnaryLogging(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			logger.Error("gRPC unary call failed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
				"error", err,
			)
		} else {
			logger.Info("gRPC unary call",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
			)
		}
		return resp, err
	}
}

// StreamLogging returns a gRPC stream server interceptor that logs stream start/end.
func StreamLogging(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		logger.Info("gRPC stream started", "method", info.FullMethod)
		start := time.Now()
		err := handler(srv, ss)
		duration := time.Since(start)

		if err != nil {
			logger.Error("gRPC stream failed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
				"error", err,
			)
		} else {
			logger.Info("gRPC stream ended",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
			)
		}
		return err
	}
}
