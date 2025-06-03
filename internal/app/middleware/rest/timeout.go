package restmw

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// TimeoutMiddleware は Echo 用のリクエストタイムアウトミドルウェア。
// 指定された duration の間にリクエスト処理が完了しなかった場合、HTTP 504 を返す。
// 通常のリクエストハンドラは middleware.WithTimeout を経由して制御される。
func TimeoutMiddleware(duration time.Duration, log logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(echoCtx echo.Context) error {
			// 元の HTTP リクエストからタイムアウト付きの Context を生成
			ctx, cancel := context.WithTimeout(echoCtx.Request().Context(), duration)
			defer cancel()

			// Echo Context に新しいタイムアウト付き Context を持つリクエストをセット
			// EchoCtx 自体は goroutine に渡さないように、ここでセットする
			newReq := echoCtx.Request().WithContext(ctx)
			echoCtx.SetRequest(newReq)

			// 共通の WithTimeout 関数を用いて、タイムアウト処理を適用
			return middleware.WithTimeout(
				ctx,
				log,
				func(_ context.Context) error {
					// 通常のハンドラー実行
					return next(echoCtx)
				},
				func() error {
					// タイムアウト時の処理：ログ出力と HTTP 504 レスポンス
					log.Warn("REST timeout", "path", echoCtx.Path())

					return echoCtx.JSON(http.StatusGatewayTimeout, map[string]string{
						"error": "request timed out",
					})
				},
			)
		}
	}
}
