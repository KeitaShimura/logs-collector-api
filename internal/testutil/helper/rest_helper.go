package testhelper

import (
	"database/sql"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
	restmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/rest"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
	"github.com/KeitaShimura/logs-collector-api/internal/pkg/validator"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
)

// テスト用の REST タイムアウトを定義
const restTimeout = 3 * time.Second

// SetupRestTestHandler は、REST API テスト用の環境を初期化し、
// Echo サーバ、SQL DB、モックプロデューサ、モックサーチャを返します。
func SetupRestTestHandler(t *testing.T) (*echo.Echo, *sql.DB, *appmock.Producer, *appmock.LogSearcher) {
	t.Helper()

	// ユースケースとモックを初期化
	uc, sqlDB, mockProducer, mockSearcher := initTestUseCase(t)

	// ロガーを生成し、ハンドラに注入
	log := logger.NewLogger()
	logHandler := rest.NewLogHandler(uc, log)

	// Echo サーバインスタンスを生成
	echoServer := echo.New()
	// ミドルウェアを順次登録
	echoServer.Use(restmw.LoggingMiddleware(log))              // ログ出力
	echoServer.Use(restmw.TimeoutMiddleware(restTimeout, log)) // タイムアウト制御
	echoServer.Use(restmw.RecoveryMiddleware(log))             // パニックリカバリ

	// リクエストバリデーション設定
	echoServer.Validator = validator.NewValidator()

	// Swagger UI エンドポイントを登録
	echoServer.GET("/swagger/*", echoSwagger.WrapHandler)

	// API グループ
	api := echoServer.Group("/api")

	// API エンドポイントと対応ハンドラを登録
	api.POST("/logs", logHandler.SendLog, restmw.ValidationMiddlewareSendLog(log))
	api.GET("/logs", logHandler.GetLogs, restmw.ValidationMiddlewareGetLogs(log))

	// 初期化したリソースを返却
	return echoServer, sqlDB, mockProducer, mockSearcher
}
