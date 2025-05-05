package restmw

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// RecoveryMiddleware は Echo 用の panic リカバリー用ミドルウェア
func RecoveryMiddleware(log logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					// 共通のリカバリーロガーを呼び出し、panic 内容を記録
					middleware.RecoveryHandler(log, map[string]interface{}{
						"panic":  r,
						"path":   ctx.Request().URL.Path,
						"method": ctx.Request().Method,
					})

					// クライアントに HTTP 500 を返却
					if err := ctx.JSON(http.StatusInternalServerError, map[string]string{
						"error": "internal server error",
					}); err != nil {
						log.Error("failed to send error response", err)
					}
				}
			}()

			// 通常のハンドラー処理を実行
			return next(ctx)
		}
	}
}
