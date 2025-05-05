package server

import (
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	// Swagger docs を読み込むための blank import（swag により生成される）
	_ "github.com/KeitaShimura/logs-collector-api/docs"
	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
	customValidator "github.com/KeitaShimura/logs-collector-api/internal/pkg/validator"
)

// NewRouter は Echo サーバーのルーターを初期化し、エンドポイントを登録する
func NewRouter(logHandler *rest.LogHandler) *echo.Echo {
	// Echo インスタンスを作成
	echoServer := echo.New()

	// カスタムバリデーターを設定（Echo のバリデーション用）
	echoServer.Validator = customValidator.NewValidator()

	// Swagger UI エンドポイントを登録
	echoServer.GET("/swagger/*", echoSwagger.WrapHandler)

	api := echoServer.Group("/api")

	api.POST("/logs", logHandler.SendLog)
	api.GET("/logs", logHandler.GetLogs)

	return echoServer
}
