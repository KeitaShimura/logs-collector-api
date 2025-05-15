package restmw_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	restmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/rest"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// TestLoggingMiddleware_InfoLogEmitted は正常系リクエストで Info ログが出力されることを確認する
func TestLoggingMiddleware_InfoLogEmitted(t *testing.T) {
	t.Parallel()

	// モックロガーを使ってログ出力を確認
	mockLogger := testutil.NewMockLogger()

	echoServer := echo.New()
	echoServer.Use(restmw.LoggingMiddleware(mockLogger))

	// エンドポイントを設定
	echoServer.GET("/hello", func(echoCtx echo.Context) error {
		// 必要な context メタデータを埋め込む
		ctx := echoCtx.Request().Context()
		ctx = context.WithValue(ctx, middleware.ContextKeyTraceID, "trace-xyz")
		ctx = context.WithValue(ctx, middleware.ContextKeyRequestID, "req-123")
		ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "user-456")
		ctx = context.WithValue(ctx, middleware.ContextKeyClientIP, "127.0.0.1")
		echoCtx.SetRequest(echoCtx.Request().WithContext(ctx))

		return echoCtx.String(http.StatusOK, "ok")
	})

	// テストリクエストを実行
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// ステータスコード検証
	require.Equal(t, http.StatusOK, rec.Code)

	// ログ出力検証
	require.Len(t, mockLogger.Infos, 1)
	entry := mockLogger.Infos[0]
	require.Equal(t, "request completed", entry.Msg)

	// 含まれるフィールドを確認（構造化ログ）
	require.Contains(t, entry.Args, "trace_id")
	require.Contains(t, entry.Args, "trace-xyz")

	require.Contains(t, entry.Args, "request_id")
	require.Contains(t, entry.Args, "req-123")

	require.Contains(t, entry.Args, "method")
	require.Contains(t, entry.Args, "GET /hello")

	require.Contains(t, entry.Args, "status_code")
	require.Contains(t, entry.Args, "200")

	require.Contains(t, entry.Args, "duration_ms")
	testutil.AssertDurationMsFieldExists(t, entry.Args)

	require.Contains(t, entry.Args, "user_id")
	require.Contains(t, entry.Args, "user-456")

	require.Contains(t, entry.Args, "client_ip")
	require.Contains(t, entry.Args, "127.0.0.1")
}

// TestLoggingMiddleware_ErrorLogEmitted は異常系リクエストで Error ログが出力されることを確認する
func TestLoggingMiddleware_ErrorLogEmitted(t *testing.T) {
	t.Parallel()

	// モックロガーを準備
	mockLogger := testutil.NewMockLogger()

	echoServer := echo.New()
	echoServer.Use(restmw.LoggingMiddleware(mockLogger))

	// 異常系エンドポイント（明示的にエラーを返す）
	echoServer.GET("/error", func(echoCtx echo.Context) error {
		ctx := echoCtx.Request().Context()
		ctx = context.WithValue(ctx, middleware.ContextKeyTraceID, "trace-err")
		ctx = context.WithValue(ctx, middleware.ContextKeyRequestID, "req-err")
		ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "user-err")
		ctx = context.WithValue(ctx, middleware.ContextKeyClientIP, "192.168.1.100")
		echoCtx.SetRequest(echoCtx.Request().WithContext(ctx))

		echoCtx.Response().WriteHeader(http.StatusInternalServerError)

		return echo.NewHTTPError(http.StatusInternalServerError, "internal failure")
	})

	// テストリクエスト送信
	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// ステータス確認
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	// ログ出力（Error）を検証
	require.Len(t, mockLogger.Errors, 1)
	entry := mockLogger.Errors[0]
	require.Equal(t, "request failed", entry.Msg)
	require.Error(t, entry.Err)
	require.Contains(t, entry.Err.Error(), "internal failure")

	// 含まれるフィールドを確認
	require.Contains(t, entry.Args, "trace_id")
	require.Contains(t, entry.Args, "trace-err")

	require.Contains(t, entry.Args, "request_id")
	require.Contains(t, entry.Args, "req-err")

	require.Contains(t, entry.Args, "method")
	require.Contains(t, entry.Args, "GET /error")

	require.Contains(t, entry.Args, "status_code")
	require.Contains(t, entry.Args, "500")

	require.Contains(t, entry.Args, "duration_ms")
	testutil.AssertDurationMsFieldExists(t, entry.Args)

	require.Contains(t, entry.Args, "user_id")
	require.Contains(t, entry.Args, "user-err")

	require.Contains(t, entry.Args, "client_ip")
	require.Contains(t, entry.Args, "192.168.1.100")
}
