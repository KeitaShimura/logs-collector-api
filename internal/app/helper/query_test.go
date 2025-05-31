package helper_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/helper"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
)

// setupHelperTest は ParseQueryParams のテスト用に Echo インスタンス、リクエスト、レスポンスなどを準備するヘルパー関数
func setupHelperTest(t *testing.T, query string) (
	*appmock.Logger,
	*httptest.ResponseRecorder,
	*http.Request,
	*echo.Echo,
) {
	t.Helper()

	echoServer := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/logs?"+query, nil)
	rec := httptest.NewRecorder()
	mockLogger := appmock.NewLogger()

	return mockLogger, rec, req, echoServer
}

// --- ParseQueryParams Tests ---

// TestParseQueryParams_Success はすべてのクエリパラメータが正しく指定された場合のテスト
func TestParseQueryParams_Success(t *testing.T) {
	t.Parallel()

	logger, rec, req, echoServer := setupHelperTest(t, "service=test&level=info&limit=10&offset=2")
	ctx := echoServer.NewContext(req, rec)

	params, err := helper.ParseQueryParams(ctx, logger)

	require.NoError(t, err)
	require.Equal(t, "test", params.Service)
	require.Equal(t, "info", params.Level)
	require.Equal(t, 10, params.Limit)
	require.Equal(t, 2, params.Offset)
}

// TestParseQueryParams_DefaultLimit は limit が未指定のときに 100 が使われることを確認
func TestParseQueryParams_DefaultLimit(t *testing.T) {
	t.Parallel()

	logger, rec, req, echoServer := setupHelperTest(t, "service=test-service&level=info&offset=5")
	ctx := echoServer.NewContext(req, rec)

	params, err := helper.ParseQueryParams(ctx, logger)

	require.NoError(t, err)
	require.Equal(t, 100, params.Limit)
	require.Equal(t, 5, params.Offset)
}

// TestParseQueryParams_DefaultOffset は offset が未指定のときに 0 が使われることを確認
func TestParseQueryParams_DefaultOffset(t *testing.T) {
	t.Parallel()

	logger, rec, req, echoServer := setupHelperTest(t, "service=test-service&level=info&limit=20")
	ctx := echoServer.NewContext(req, rec)

	params, err := helper.ParseQueryParams(ctx, logger)

	require.NoError(t, err)
	require.Equal(t, 0, params.Offset)
	require.Equal(t, 20, params.Limit)
}

// TestParseQueryParams_InvalidLimit は limit が数値でない場合にエラーとなることを確認
func TestParseQueryParams_InvalidLimit(t *testing.T) {
	t.Parallel()

	logger, rec, req, echoServer := setupHelperTest(t, "limit=abc")
	ctx := echoServer.NewContext(req, rec)

	_, err := helper.ParseQueryParams(ctx, logger)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid limit parameter")
	require.Len(t, logger.Warns, 1)
	require.Contains(t, logger.Warns[0].Msg, "Invalid limit")
}

// TestParseQueryParams_InvalidOffset は offset が数値でない場合にエラーとなることを確認
func TestParseQueryParams_InvalidOffset(t *testing.T) {
	t.Parallel()

	logger, rec, req, echoServer := setupHelperTest(t, "offset=xyz")
	ctx := echoServer.NewContext(req, rec)

	_, err := helper.ParseQueryParams(ctx, logger)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid offset parameter")
	require.Len(t, logger.Warns, 1)
	require.Contains(t, logger.Warns[0].Msg, "Invalid offset")
}
