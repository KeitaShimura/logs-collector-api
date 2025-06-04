package server

import (
	"time"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	// Swagger docs を読み込むための blank import（swag により生成される）
	_ "github.com/KeitaShimura/logs-collector-api/docs"
	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
	restmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/rest"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
	customValidator "github.com/KeitaShimura/logs-collector-api/internal/pkg/validator"
)

// NewRouter は Echo サーバーのルーターを初期化し、エンドポイントを登録する
func NewRouter(logHandler *rest.LogHandler, logger logger.Logger, restTimeout time.Duration) *echo.Echo {
	// Echo インスタンスを作成
	echoServer := echo.New()

	// 共通ミドルウェア
	echoServer.Use(restmw.LoggingMiddleware(logger))
	echoServer.Use(restmw.TimeoutMiddleware(restTimeout, logger))
	echoServer.Use(restmw.RecoveryMiddleware(logger))

	// カスタムバリデーターを設定（Echo のバリデーション用）
	echoServer.Validator = customValidator.NewValidator()

	// Swagger UI エンドポイントを登録
	echoServer.GET("/swagger/*", echoSwagger.WrapHandler)

	// API グループ
	api := echoServer.Group("/api")

	// 各エンドポイント
	api.POST("/logs", logHandler.SendLog, restmw.ValidationMiddlewareSendLog(logger))
	api.GET("/logs", logHandler.GetLogs, restmw.ValidationMiddlewareGetLogs(logger))

	return echoServer
}
