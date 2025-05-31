package restmw_test

import (
	"bytes"
	"encoding/json"
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
var errMockRead = errors.New("mock read error")

// --- setup ---

// errorReader は Read 時に常にエラーを返すテスト用の io.Reader 実装。
type errorReader struct{}

// Read は毎回エラーを返し、読み取り失敗をシミュレートする。
func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errMockRead
}

// --- ValidationMiddlewareSendLog Tests ---

// TestValidationMiddlewareSendLog_ValidRequest は、正しいリクエストを送信した場合に OK を返し、エラーログや警告ログが出力されないことを検証する
func TestValidationMiddlewareSendLog_ValidRequest(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()

	// 有効な JSON リクエストボディを作成
	reqBody := map[string]interface{}{
		"log": map[string]interface{}{
			"id":        "log-id",
			"trace_id":  "trace-123",
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     "INFO",
			"service":   "svc",
			"message":   "test message",
			"metadata":  map[string]string{},
		},
	}
	data, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// リクエスト作成
	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(data))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// モックロガーとミドルウェア適用
	mockLogger := appmock.NewLogger()
	mw := restmw.ValidationMiddlewareSendLog(mockLogger)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// ハンドラ実行
	err = handler(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "OK", resp.Body.String())

	// ログ出力なしを確認（正常系）
	require.Empty(t, mockLogger.Warns)
	require.Empty(t, mockLogger.Errors)
}

// TestValidationMiddlewareSendLog_ReadBodyError は、リクエストボディの読み込みに失敗した場合に 400 を返し、エラーログが出力されることを検証する
func TestValidationMiddlewareSendLog_ReadBodyError(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()

	// 読み込み失敗を発生させるリクエストボディを差し込む
	req := httptest.NewRequest(http.MethodPost, "/logs", &errorReader{})
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// モックロガー
	mockLogger := appmock.NewLogger()

	mw := restmw.ValidationMiddlewareSendLog(mockLogger)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "SHOULD NOT HAPPEN")
	})

	// 実行
	err := handler(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Contains(t, resp.Body.String(), "failed to read request")

	// エラーログ出力確認
	require.Len(t, mockLogger.Errors, 1)
	require.Contains(t, mockLogger.Errors[0].Msg, "failed to read body")
}

// TestValidationMiddlewareSendLog_InvalidJSON は、不正な JSON を送信した場合に 400 を返し、エラーログが出力されることを検証する
func TestValidationMiddlewareSendLog_InvalidJSON(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()

	// 不正な JSON をボディに設定
	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader([]byte(`invalid json`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// ミドルウェア適用
	mockLogger := appmock.NewLogger()
	mw := restmw.ValidationMiddlewareSendLog(mockLogger)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// 実行して 400 を期待
	err := handler(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Contains(t, resp.Body.String(), "invalid request")

	// エラーログ出力を確認
	require.Len(t, mockLogger.Errors, 1)
	require.Contains(t, mockLogger.Errors[0].Msg, "invalid protobuf json")
}

// TestValidationMiddlewareSendLog_InvalidRequest は、trace_id が空の不正なリクエストに対し、400 を返し、Warn ログが出力されることを検証する
func TestValidationMiddlewareSendLog_InvalidRequest(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()

	// trace_id が空 → ValidateSendLogRequest によるバリデーションエラーを発生させる
	reqBody := map[string]interface{}{
		"log": map[string]interface{}{
			"id":        "log-id",
			"trace_id":  "", // 空 → エラーになる
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     "INFO",
			"service":   "svc",
			"message":   "msg",
		},
	}
	data, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(data))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	mockLogger := appmock.NewLogger()
	mw := restmw.ValidationMiddlewareSendLog(mockLogger)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// 実行：バリデーションにより 400 を期待
	err = handler(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Contains(t, resp.Body.String(), "trace_id must not be empty")

	// Warn ログが出力されていることを確認（ValidateSendLogRequest 経由）
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "SendLog validation failed")
}

// --- ValidationMiddlewareGetLogs Tests ---

// TestValidationMiddlewareGetLogs_Valid は、GET /logs のクエリが有効な場合に OK を返し、ログが出力されないことを検証する
func TestValidationMiddlewareGetLogs_Valid(t *testing.T) {
	t.Parallel()

	e := echo.New()

	// 有効なクエリ（limit, offset 範囲内）
	req := httptest.NewRequest(http.MethodGet, "/logs?service=svc&level=INFO&limit=10&offset=0", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	mockLogger := appmock.NewLogger()
	mw := restmw.ValidationMiddlewareGetLogs(mockLogger)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// 実行して 200 OK を期待
	err := handler(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "OK", resp.Body.String())

	// 正常系なのでログは出力されない
	require.Empty(t, mockLogger.Warns)
	require.Empty(t, mockLogger.Errors)
}

// TestValidationMiddlewareGetLogs_InvalidLimit は、limit が不正な場合に 400 として処理され、警告ログが出力されることを検証する
func TestValidationMiddlewareGetLogs_InvalidLimit(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()

	// limit が負 → バリデーションエラー
	req := httptest.NewRequest(http.MethodGet, "/logs?service=svc&level=INFO&limit=-1&offset=0", nil)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	mockLogger := appmock.NewLogger()
	mw := restmw.ValidationMiddlewareGetLogs(mockLogger)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "SHOULD NOT HAPPEN")
	})

	// 実行
	err := handler(ctx)

	// エラーが返らず、レスポンスに 400 が書き込まれているはず
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	// レスポンスボディに "limit" を含む JSON が返っていること
	require.Contains(t, resp.Body.String(), "limit")

	// 警告ログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Validation failed")
}
