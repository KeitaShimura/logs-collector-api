package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/pkg/server"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// TestSwaggerRoute はSwaggerルートが登録されていることを確認するテスト
func TestSwaggerRoute(t *testing.T) {
	t.Parallel()

	// モックのユースケースとロガーを準備
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger), mockLogger)

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
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger), mockLogger)

	// テスト用のPOSTリクエストボディを作成
	reqBody := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "11111111-1111-1111-1111-111111111111",
			TraceId:   "22222222-2222-2222-2222-222222222222",
			Message:   "test message",
			Level:     "INFO",
			Service:   "test-service",
			Timestamp: timestamppb.Now(),
			Metadata:  map[string]string{},
		},
	}

	// JSON（protojson）にエンコード
	body, err := protojson.Marshal(reqBody)
	require.NoError(t, err)

	// POSTリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/api/logs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()

	// リクエストをサーバーに送信
	echoServer.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Logf("POST response: %s", rec.Body.String())
	}

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
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger), mockLogger)

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
	mockUC.On("GetLogs",
		mock.Anything,
		"test-service",
		"INFO",
		100,
		0,
	).Return([]model.Log{}, nil)

	mockLogger := testutil.NewMockLogger()
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger), mockLogger)

	// GETリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/api/logs?limit=100&offset=0&service=test-service&level=INFO", nil)
	rec := httptest.NewRecorder()

	// リクエストをサーバーに送信
	echoServer.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Logf("GET response: %s", rec.Body.String())
	}

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
	echoServer := server.NewRouter(rest.NewLogHandler(mockUC, mockLogger), mockLogger)

	// 存在しないGETルートへのリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/api/invalid", nil)
	rec := httptest.NewRecorder()

	// リクエストをサーバーに送信
	echoServer.ServeHTTP(rec, req)

	// 404 Not Found を期待
	require.Equal(t, http.StatusNotFound, rec.Code, "Unknown GET route should return 404")
}
