package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	testhelper "github.com/KeitaShimura/logs-collector-api/internal/testutil/helper"
)

// TestGetLogs_Success は、REST API の /logs エンドポイントで、正常にログ一覧が返却されることを検証します。
func TestGetLogs_Success(t *testing.T) {
	t.Parallel()

	// Echo サーバと DB、モックをセットアップ
	echoServer, sqlDB, _, _ := testhelper.SetupRestTestHandler(t)
	defer sqlDB.Close() // テスト終了後に DB 接続をクローズ

	// テスト用データを事前に挿入
	require.NoError(t, testhelper.InsertTestLog(t.Context(), sqlDB))

	// HTTP リクエストを作成（クエリパラメータ付き）
	req := httptest.NewRequest(http.MethodGet, "/api/logs?service=test-service&level=INFO&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	// サーバにリクエストを送信
	echoServer.ServeHTTP(rec, req)

	// レスポンスボディを JSON としてパース
	var logs []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &logs)
	require.NoError(t, err)

	// 期待通り 1 件のログが返却され、Service フィールドが一致すること
	require.Len(t, logs, 1)
	require.Equal(t, "test-service", logs[0]["Service"])
}

// TestGetLogs_EmptyResult は、該当ログが存在しない場合に、HTTP 404 が返却されることを検証します。
func TestGetLogs_EmptyResult(t *testing.T) {
	t.Parallel()

	echoServer, sqlDB, _, _ := testhelper.SetupRestTestHandler(t)
	defer sqlDB.Close()

	// ログを挿入せずに GET リクエスト
	req := httptest.NewRequest(http.MethodGet, "/api/logs?service=not-found&level=DEBUG&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// ステータスコードが 404 Not Found であること
	require.Equal(t, http.StatusNotFound, rec.Code)
}

// TestGetLogs_InvalidQueryParam は、クエリパラメータが不正な場合に、HTTP 400 が返却されることを検証します。
func TestGetLogs_InvalidQueryParam(t *testing.T) {
	t.Parallel()

	echoServer, sqlDB, _, _ := testhelper.SetupRestTestHandler(t)
	defer sqlDB.Close()

	// limit に文字列を指定して不正リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/api/logs?limit=abc", nil)
	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// ステータスコードが 400 Bad Request であること
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestGetLogs_DBFailure は、DB 接続障害時に、HTTP 500 が返却されることを検証します。
func TestGetLogs_DBFailure(t *testing.T) {
	t.Parallel()

	echoServer, sqlDB, _, _ := testhelper.SetupRestTestHandler(t)
	// 意図的に DB 接続をクローズして障害を発生
	sqlDB.Close()

	// サービスとレベルのみ指定してリクエスト
	req := httptest.NewRequest(http.MethodGet, "/api/logs?service=test-service&level=INFO", nil)
	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// ステータスコードが 500 Internal Server Error であること
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// TestGetLogs_OnlyServiceQuery は、service のみ指定した場合でも、正常にログが返却されることを検証します。
func TestGetLogs_OnlyServiceQuery(t *testing.T) {
	t.Parallel()

	echoServer, sqlDB, _, _ := testhelper.SetupRestTestHandler(t)
	defer sqlDB.Close()

	// テストログを事前に挿入
	require.NoError(t, testhelper.InsertTestLog(t.Context(), sqlDB))

	// service パラメータのみ指定してリクエスト
	req := httptest.NewRequest(http.MethodGet, "/api/logs?service=test-service", nil)
	rec := httptest.NewRecorder()
	echoServer.ServeHTTP(rec, req)

	// レスポンスをパースし、1 件のログが返却されることを検証
	var logs []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &logs)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, "test-service", logs[0]["Service"])
}
