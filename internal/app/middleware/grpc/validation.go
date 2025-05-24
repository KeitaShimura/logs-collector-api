// internal/interface/grpcmw/validation.go
package grpcmw

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/KeitaShimura/logs-collector-api/internal/app/helper"
	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// ValidationInterceptor は gRPC リクエストに対する事前バリデーションインターセプター
func ValidationInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		switch typedReq := req.(type) {
		case *pb.SendLogRequest:
			if err := middleware.ValidateSendLogRequest(typedReq); err != nil {
				log.Warn("SendLog validation failed", "error", err)

				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		case *pb.GetLogsRequest:
			err := middleware.ValidateGetLogsRequest(&helper.QueryParams{
				Service: typedReq.GetService(),
				Level:   typedReq.GetLevel(),
				Limit:   int(typedReq.GetLimit()),
				Offset:  int(typedReq.GetOffset()),
			})
			if err != nil {
				log.Warn("GetLogs validation failed", "error", err)

				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}

		return handler(ctx, req)
	}
}
