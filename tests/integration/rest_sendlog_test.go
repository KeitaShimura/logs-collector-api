package integration_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	testhelper "github.com/KeitaShimura/logs-collector-api/internal/testutil/helper"
)

// REST POST /api/logs の統合テスト

// TestPostLogs_ValidRequest は、正しいログ送信リクエストを送った場合のフローを検証します。
//  1. HTTP 200 を返す
//  2. NATS / Elasticsearch のモックが 1 回ずつ呼ばれる
func TestPostLogs_ValidRequest(t *testing.T) {
	t.Parallel()

	// 前処理: Echo サーバーとモックを初期化
	echoServer, sqlDB, mockProducer, mockSearcher := testhelper.SetupRestTestHandler(t)
	defer sqlDB.Close()

	// 入力データ生成
	body := `{
        "log": {
            "trace_id": "trace-123",
            "timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `",
            "level": "INFO",
            "service": "test-service",
            "message": "integration test log",
            "metadata": {"env": "test"}
        }
    }`

	// HTTP リクエスト送信
	req := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// 検証
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, mockProducer.PublishedMessages, 1)
	require.Len(t, mockSearcher.Calls, 1)
}

// TestPostLogs_EmptyService は service が空の場合を検証します。
//   - 期待: HTTP 400, NATS / Elasticsearch のモック呼び出し 0 回
func TestPostLogs_EmptyService(t *testing.T) {
	t.Parallel()

	// 前処理
	echoServer, sqlDB, mockProducer, mockSearcher := testhelper.SetupRestTestHandler(t)
	defer sqlDB.Close()

	// 入力データ（service が空）
	body := `{
        "log": {
            "trace_id": "trace-123",
            "timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `",
            "level": "INFO",
            "service": "",
            "message": "no service",
            "metadata": {}
        }
    }`

	// リクエスト送信
	req := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// 検証
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, mockProducer.PublishedMessages)
	require.Empty(t, mockSearcher.Calls)
}

// TestPostLogs_InvalidJSON は JSON が壊れている場合を検証します。
//   - 期待: HTTP 400, NATS / Elasticsearch のモック呼び出し 0 回
func TestPostLogs_InvalidJSON(t *testing.T) {
	t.Parallel()

	// 前処理
	echoServer, sqlDB, mockProducer, mockSearcher := testhelper.SetupRestTestHandler(t)
	defer sqlDB.Close()

	// 不正な JSON
	body := `{"log": {"trace_id": "abc", "timestamp": "bad"}`

	// リクエスト送信
	req := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// 検証
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, mockProducer.PublishedMessages)
	require.Empty(t, mockSearcher.Calls)
}

// TestPostLogs_FutureTimestamp は timestamp が将来日付の場合を検証します。
//   - 期待: HTTP 400, NATS / Elasticsearch のモック呼び出し 0 回
func TestPostLogs_FutureTimestamp(t *testing.T) {
	t.Parallel()

	// 前処理
	echoServer, sqlDB, mockProducer, mockSearcher := testhelper.SetupRestTestHandler(t)
	defer sqlDB.Close()

	// 未来日時を設定
	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	body := `{
        "log": {
            "trace_id": "trace-123",
            "timestamp": "` + future + `",
            "level": "INFO",
            "service": "test-service",
            "message": "future timestamp",
            "metadata": {}
        }
    }`

	// リクエスト送信
	req := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// 検証
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, mockProducer.PublishedMessages)
	require.Empty(t, mockSearcher.Calls)
}

// TestPostLogs_DBFailure は DB 接続エラー時を検証します。
//   - 期待: HTTP 500, NATS / Elasticsearch のモック呼び出し 0 回
func TestPostLogs_DBFailure(t *testing.T) {
	t.Parallel()

	// 前処理
	echoServer, sqlDB, mockProducer, mockSearcher := testhelper.SetupRestTestHandler(t)

	// DB を故意に閉じる
	sqlDB.Close()

	// 入力データ
	body := `{
        "log": {
            "trace_id": "trace-err",
            "timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `",
            "level": "INFO",
            "service": "test-service",
            "message": "should fail",
            "metadata": {}
        }
    }`

	// リクエスト送信
	req := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// 検証
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Empty(t, mockProducer.PublishedMessages)
	require.Empty(t, mockSearcher.Calls)
}
