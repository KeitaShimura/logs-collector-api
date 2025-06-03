package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// WithTimeout は、指定された Context 内で handler を実行し、
// タイムアウトまたはキャンセルが発生した場合に onTimeout を呼び出す共通処理。
// - handler: 通常のリクエスト処理（REST / gRPC 両対応）
// - onTimeout: タイムアウト時に呼ばれる処理（レスポンス返却やエラーログ記録など）
// 戻り値は handler または onTimeout の返却値。
func WithTimeout(
	ctx context.Context,
	log logger.Logger,
	handler func(ctx context.Context) error,
	onTimeout func() error,
) error {
	// handler を実行（この時点で ctx.DeadlineExceeded ではない前提）
	err := handler(ctx)
	if err != nil {
		return err // handler 側でエラーがあればそのまま返す
	}

	// handler 完了後に context の状態を確認（キャンセルやタイムアウトが発生していないか）
	if ctx.Err() != nil {
		// タイムアウトで終了していた場合
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Warn("timeout exceeded")

			return onTimeout() // タイムアウト処理を実行
		}

		// その他の Context エラー（キャンセルなど）はそのまま返却
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	// 正常終了
	return nil
}
