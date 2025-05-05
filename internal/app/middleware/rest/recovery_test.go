package restmw_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	restmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/rest"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// 共通エラー定義
var ErrHandler = errors.New("handler error")

func TestRecoveryMiddleware_NormalFlow(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	mockLogger := testutil.NewMockLogger()

	// Echo サーバに RecoveryMiddleware を登録
	echoServer.Use(restmw.RecoveryMiddleware(mockLogger))

	// 正常に "OK" を返すハンドラー
	echoServer.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Echo 全体を通してリクエストを実行
	echoServer.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "OK", rec.Body.String())
	require.Empty(t, mockLogger.Errors)
}

// TestRecoveryMiddleware_Panic はハンドラー内で panic が発生した場合の回復テスト
func TestRecoveryMiddleware_Panic(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	mockLogger := testutil.NewMockLogger()

	// RecoveryMiddleware を登録
	echoServer.Use(restmw.RecoveryMiddleware(mockLogger))

	// panic を発生させるハンドラー
	echoServer.GET("/panic", func(_ echo.Context) error {
		panic("simulated panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	// Echo 全体で ServeHTTP を使うことでミドルウェアを通過させる
	echoServer.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "internal server error")

	// Error ログが記録されていることを確認
	require.Len(t, mockLogger.Errors, 1)
	require.Contains(t, mockLogger.Errors[0].Msg, "panic recovered")
}
