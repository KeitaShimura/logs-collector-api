package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/pkg/server"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// TestSwaggerRoute はSwaggerルートが登録されていることを確認するテスト
func TestSwaggerRoute(t *testing.T) {
	t.Parallel()

	// モックのユースケースとロガーを準備
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger))

	// Swaggerドキュメント用のGETリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rec := httptest.NewRecorder()

	// リクエストをサーバーに送信
	echoServer.ServeHTTP(rec, req)

	// 404でないことを確認（Swaggerルートが存在することを確認）
	require.NotEqual(t, http.StatusNotFound, rec.Code, "Swagger route should be registered")
}

// TestPostLogsRoute_Success はPOST /api/logsルートが正常に登録され、モックが呼ばれることを確認するテスト
func TestPostLogsRoute_Success(t *testing.T) {
	t.Parallel()

	// モックユースケースのSendLogを成功レスポンスに設定
	mockUC := new(testutil.MockLogUseCase)
	mockUC.On("SendLog", mock.Anything, mock.AnythingOfType("*model.Log")).Return(nil)

	mockLogger := testutil.NewMockLogger()
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger))

	// テスト用のPOSTリクエストボディを作成
	reqBody := rest.SendLogRequest{
		ID:        "",
		TraceID:   "",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Metadata:  nil,
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// POSTリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/api/logs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()

	// リクエストをサーバーに送信
	echoServer.ServeHTTP(rec, req)

	// 404ではないこと、および200 OKであることを確認
	require.NotEqual(t, http.StatusNotFound, rec.Code, "POST /api/logs route should be registered")
	require.Equal(t, http.StatusOK, rec.Code, "POST /api/logs should return 200 OK")

	// モックの期待が満たされていることを確認
	mockUC.AssertExpectations(t)
}

// TestPostLogsRoute_NotFound は存在しないPOSTルートが404を返すことを確認するテスト
func TestPostLogsRoute_NotFound(t *testing.T) {
	t.Parallel()

	// モックのユースケースとロガーを準備
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger))

	// 存在しないPOSTルートへのリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/api/unknown", nil)
	rec := httptest.NewRecorder()

	// リクエストをサーバーに送信
	echoServer.ServeHTTP(rec, req)

	// 404 Not Found を期待
	require.Equal(t, http.StatusNotFound, rec.Code, "Unknown route should return 404")
}

// TestGetLogsRoute_Success はGET /api/logsルートが正常に登録され200を返すことを確認するテスト
func TestGetLogsRoute_Success(t *testing.T) {
	t.Parallel()

	// モックユースケースのGetLogsを成功レスポンスに設定
	mockUC := new(testutil.MockLogUseCase)
	mockUC.On("GetLogs", mock.Anything, "", "", 100, 0).Return([]model.Log{}, nil)

	mockLogger := testutil.NewMockLogger()
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger))

	// GETリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/api/logs", nil)
	rec := httptest.NewRecorder()

	// リクエストをサーバーに送信
	echoServer.ServeHTTP(rec, req)

	// 404ではないこと、および200 OKであることを確認
	require.NotEqual(t, http.StatusNotFound, rec.Code, "GET /api/logs route should be registered")
	require.Equal(t, http.StatusOK, rec.Code, "GET /api/logs should return 200 OK")

	// モックの期待が満たされていることを確認
	mockUC.AssertExpectations(t)
}

// TestGetLogsRoute_NotFound は存在しないGETルートが404を返すことを確認するテスト
func TestGetLogsRoute_NotFound(t *testing.T) {
	t.Parallel()

	// モックのユースケースとロガーを準備
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger))

	// 存在しないGETルートへのリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/api/invalid", nil)
	rec := httptest.NewRecorder()

	// リクエストをサーバーに送信
	echoServer.ServeHTTP(rec, req)

	// 404 Not Found を期待
	require.Equal(t, http.StatusNotFound, rec.Code, "Unknown GET route should return 404")
}
