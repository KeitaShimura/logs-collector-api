package restmw_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	restmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/rest"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
)

// 共通エラー定義
var errRESTHandlerFailure = errors.New("handler failure")

// TestTimeoutMiddleware_Success は、タイムアウト時間内に正常終了するケースを検証する。
func TestTimeoutMiddleware_Success(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	mockLogger := appmock.NewLogger()

	// タイムアウト時間を長めに設定して成功させる
	mw := restmw.TimeoutMiddleware(100*time.Millisecond, mockLogger)

	// タイムアウト対象ハンドラ（短時間で完了）
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, rec)

	err := handler(ctx)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
	require.Empty(t, mockLogger.Warns, "no warning logs should be emitted")
}

// TestTimeoutMiddleware_Timeout は、タイムアウトが発生した場合の動作（504 とログ出力）を検証する。
func TestTimeoutMiddleware_Timeout(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	mockLogger := appmock.NewLogger()

	// タイムアウト時間を短くして、意図的にタイムアウトさせる
	mw := restmw.TimeoutMiddleware(10*time.Millisecond, mockLogger)

	handler := mw(func(_ echo.Context) error {
		time.Sleep(50 * time.Millisecond)

		return nil // レスポンスは書かず、純粋にタイムアウトをテスト
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, rec)

	err := handler(ctx)

	require.NoError(t, err)
	require.Equal(t, http.StatusGatewayTimeout, rec.Code)
	require.Contains(t, rec.Body.String(), "request timed out")

	// Warn ログが出力されていること
	require.Len(t, mockLogger.Warns, 2)
	require.Equal(t, "timeout exceeded", mockLogger.Warns[0].Msg)
	require.Equal(t, "REST timeout", mockLogger.Warns[1].Msg)
}

// TestTimeoutMiddleware_HandlerError は、タイムアウトしていないが handler 自体がエラーを返すケースを検証する。
func TestTimeoutMiddleware_HandlerError(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	mockLogger := appmock.NewLogger()

	mw := restmw.TimeoutMiddleware(100*time.Millisecond, mockLogger)

	expectedErr := errRESTHandlerFailure

	handler := mw(func(_ echo.Context) error {
		return expectedErr
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, rec)

	err := handler(ctx)

	require.ErrorIs(t, err, expectedErr)
	require.Empty(t, mockLogger.Warns, "no warning logs should be emitted")
}
