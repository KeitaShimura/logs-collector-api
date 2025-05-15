package restmw

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// LoggingMiddleware は Echo 用の構造化リクエストログ出力ミドルウェア
func LoggingMiddleware(log logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(echoCtx echo.Context) error {
			// 処理開始時刻を記録（後で処理時間を測定するため）
			start := time.Now()

			// 実際のリクエスト処理（次のハンドラー）を実行
			err := next(echoCtx)

			// 処理完了までの経過時間を取得
			duration := time.Since(start)

			// HTTPメソッドとリクエストパスを組み合わせて、操作内容を表現（例: "GET /api/v1/users"）
			method := echoCtx.Request().Method + " " + echoCtx.Path()

			// レスポンスステータスコードを文字列で取得（例: "200", "404"）
			statusCode := strconv.Itoa(echoCtx.Response().Status)

			// LoggingHandler を使って構造化ログを出力
			// trace_id や user_id などは context から LoggingHandler 側で取得
			middleware.LoggingHandler(
				echoCtx.Request().Context(),
				log,
				method,
				statusCode,
				duration,
				err,
			)

			// ハンドラの結果をそのまま返す
			return err
		}
	}
}
