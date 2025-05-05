package grpcmw

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// RecoveryInterceptor は panic 発生時に gRPC サーバーを回復させるインターセプター
func RecoveryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		var resp interface{}

		var err error

		// done チャネルで goroutine の完了を待つ
		done := make(chan struct{})

		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					// panic を捕捉してログに記録
					middleware.RecoveryHandler(log, map[string]interface{}{
						"panic":  recovered,
						"method": info.FullMethod,
					})
					// エラーに Internal ステータスを設定
					err = status.Errorf(codes.Internal, "panic recovered: %v", recovered)
				}

				close(done) // goroutine 完了通知
			}()

			// handler 実行結果をセット（panic が起きた場合 defer が回収）
			resp, err = handler(ctx, req)
		}()
		<-done // goroutine の終了を待つ

		// Internal 以外のエラーを Internal にラップ
		if err != nil && status.Code(err) != codes.Internal {
			return nil, status.Errorf(codes.Internal, "internal error: %v", err)
		}

		return resp, err
	}
}
