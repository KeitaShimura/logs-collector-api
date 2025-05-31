// internal/interface/grpcmw/validation.go
package grpcmw

import (
	"context"
	"fmt"

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
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		switch typedReq := req.(type) {
		case *pb.SendLogRequest:
			log.Info("validation: received SendLogRequest", "method", info.FullMethod)

			if err := middleware.ValidateSendLogRequest(typedReq); err != nil {
				log.Warn("validation: SendLogRequest validation failed", "method", info.FullMethod, "error", err)

				return nil, status.Error(codes.InvalidArgument, err.Error())
			}

		case *pb.GetLogsRequest:
			log.Info("validation: received GetLogsRequest", "method", info.FullMethod)

			err := middleware.ValidateGetLogsRequest(&helper.QueryParams{
				Service: typedReq.GetService(),
				Level:   typedReq.GetLevel(),
				Limit:   int(typedReq.GetLimit()),
				Offset:  int(typedReq.GetOffset()),
			})
			if err != nil {
				log.Warn("validation: GetLogsRequest validation failed", "method", info.FullMethod, "error", err)

				return nil, status.Error(codes.InvalidArgument, err.Error())
			}

		default:
			log.Warn("validation: unknown request type", "method", info.FullMethod, "type", fmt.Sprintf("%T", req))
		}

		return handler(ctx, req)
	}
}
