package grpcmw

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// LoggingInterceptor は gRPC 呼び出し全体の構造化ログを出力する
func LoggingInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 呼び出し開始時刻を記録（処理時間計測のため）
		start := time.Now()

		// 実際の gRPC ハンドラを呼び出す
		resp, err := handler(ctx, req)

		// 処理完了後の経過時間を取得
		duration := time.Since(start)

		// gRPC ステータスコードを取得（例：OK, Internal, NotFound など）
		code := status.Code(err)

		// LoggingHandler を使って構造化ログを出力
		// trace_id や user_id などは context から LoggingHandler 側で取得
		middleware.LoggingHandler(
			ctx,
			log,
			info.FullMethod,
			code.String(),
			duration,
			err,
		)

		// ハンドラの結果をそのまま返す
		return resp, err
	}
}
