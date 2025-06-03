package grpcmw

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// TimeoutInterceptor は gRPC サーバー側のタイムアウト制御を行う Unary インターセプター。
// 指定された duration 内にリクエスト処理が完了しなければ、codes.DeadlineExceeded を返す。
func TimeoutInterceptor(duration time.Duration, log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, // gRPC のリクエストに関連する context
		req interface{}, // クライアントからのリクエスト
		info *grpc.UnaryServerInfo, // メソッド情報（メソッド名など）
		handler grpc.UnaryHandler, // 実際のリクエスト処理関数
	) (interface{}, error) {
		// タイムアウト付き context を生成（元 context を拡張）
		ctxWithTimeout, cancel := context.WithTimeout(ctx, duration)
		defer cancel()

		var resp interface{} // レスポンス保持用

		errChan := make(chan error, 1) // 非同期で処理結果を受け取るチャネル

		// 処理を別 goroutine で実行し、タイムアウトを監視する構造
		go func() {
			err := middleware.WithTimeout(
				ctxWithTimeout,
				log,
				func(innerCtx context.Context) error {
					var hErr error
					resp, hErr = handler(innerCtx, req) // ハンドラー実行

					return hErr
				},
				func() error {
					// WithTimeout 内部で ctx.Err() に応じて呼ばれるが、
					// この goroutine 内で呼ばれるわけではないため副作用なし
					return nil
				},
			)
			errChan <- err
		}()

		// タイムアウト or handler の完了を待機
		select {
		case <-ctxWithTimeout.Done():
			// タイムアウトが発生した場合のログとエラー返却
			log.Warn("gRPC timeout", "method", info.FullMethod)

			return nil, status.Error(codes.DeadlineExceeded, "server-side timeout")

		case err := <-errChan:
			// 通常終了または handler 側でのエラーを返却
			return resp, err
		}
	}
}
