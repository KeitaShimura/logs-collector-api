package rest_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// 共通エラー定義
var (
	errValidation = errors.New("validation error")
	errDB         = errors.New("db error")
	errUnexpected = errors.New("unexpected error")
)

// --- Imports and Dummy Setup ---

// dummyValidator は Echo の Validator インターフェースを模倣する簡易バリデータ
type dummyValidator struct {
	forceError bool // true の場合、常にバリデーションエラーを返す
}

func (v *dummyValidator) Validate(_ interface{}) error {
	if v.forceError {
		return errValidation
	}

	return nil
}

// --- setup ---

// setupRestSendLogTest は REST ハンドラーと必要なモック、Echo サーバ、HTTPリクエストを準備するヘルパー関数
func setupRestSendLogTest(t *testing.T, reqBody rest.SendLogRequest) (
	*rest.LogHandler,
	*testutil.MockLogUseCase,
	*testutil.MockLogger,
	*http.Request,
	*httptest.ResponseRecorder,
	*echo.Echo,
) {
	t.Helper()

	// Echo サーバーを作成し、ダミーバリデータを設定
	echoServer := echo.New()
	echoServer.Validator = &dummyValidator{forceError: false}

	// モックユースケースとモックロガーを作成
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()

	// テスト対象のハンドラーを初期化
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// リクエストボディを JSON にシリアライズ
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// POST リクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// レスポンスレコーダを作成
	resp := httptest.NewRecorder()

	return handler, mockUC, mockLogger, req, resp, echoServer
}

// setupParseQueryParamsTest は ParseQueryParams 用のテストヘルパー関数
func setupParseQueryParamsTest(t *testing.T, query string) (
	*rest.LogHandler,
	*testutil.MockLogger,
	*http.Request,
	*httptest.ResponseRecorder,
	*echo.Echo,
) {
	t.Helper()

	// Echo サーバーを作成
	echoServer := echo.New()

	// モックロガーを作成（ユースケースは不要なので nil）
	mockLogger := testutil.NewMockLogger()
	handler := rest.NewLogHandler(nil, mockLogger)

	// GET リクエストを作成（クエリパラメータ付き）
	req := httptest.NewRequest(http.MethodGet, "/logs?"+query, nil)

	// レスポンスレコーダを作成
	rec := httptest.NewRecorder()

	return handler, mockLogger, req, rec, echoServer
}

// setupParseAndValidateTest は ParseAndValidateRequest のテスト用ヘルパー関数
func setupParseAndValidateTest(t *testing.T, body []byte, forceValidationError bool) (
	*rest.LogHandler,
	*testutil.MockLogger,
	*httptest.ResponseRecorder,
	*http.Request,
	*echo.Echo,
) {
	t.Helper()

	// Echo サーバーを作成し、カスタムバリデーターを設定（forceValidationError が true の場合は常にエラーを返す）
	echoServer := echo.New()
	echoServer.Validator = &dummyValidator{forceError: forceValidationError}

	// モックロガーを準備（ユースケースは必要ないので nil を渡す）
	mockLogger := testutil.NewMockLogger()
	handler := rest.NewLogHandler(nil, mockLogger)

	// テスト用の HTTP リクエストとレスポンスレコーダを作成

	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// Echo コンテキストを作成
	resp := httptest.NewRecorder()

	// ハンドラーとコンテキストを返却
	return handler, mockLogger, resp, req, echoServer
}

// --- SendLog Tests ---

// TestSendLog_Success は REST API の SendLog が正常終了する場合のテスト
func TestSendLog_Success(t *testing.T) {
	t.Parallel()

	// 正常なリクエストボディを準備
	reqBody := rest.SendLogRequest{
		ID:        "",
		TraceID:   "",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  nil,
	}

	// テスト環境をセットアップ
	handler, mockUC, mockLogger, req, resp, echoServer := setupRestSendLogTest(t, reqBody)

	ctx := echoServer.NewContext(req, resp)

	// モックユースケースを設定
	mockUC.On("SendLog", mock.Anything, mock.Anything).Return(nil)

	// 実行
	err := handler.SendLog(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	// 成功ログが出力されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	require.Contains(t, mockLogger.Infos[0].Msg, "Log entry saved successfully")
}

// TestSendLog_BadRequest は不正なJSONリクエストで400が返ることを確認するテスト
func TestSendLog_BadRequest(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// 不正なJSON文字列を送信
	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader([]byte("invalid-json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// 実行
	err := handler.SendLog(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Failed to bind request body")
}

// TestSendLog_ValidationError はバリデーションエラーで400が返ることを確認するテスト
func TestSendLog_ValidationError(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	echoServer.Validator = &dummyValidator{forceError: true}

	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// バリデーションに引っかかるリクエストボディを準備
	reqBody := rest.SendLogRequest{
		ID:        "",
		TraceID:   "",
		Message:   "test message",
		Level:     "invalid-level", // バリデーションエラーを引き起こす値
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  nil,
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// 実行
	err = handler.SendLog(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Validation failed")
}

// TestSendLog_TimestampParseError は不正なタイムスタンプが指定された場合に400が返ることを確認するテスト
func TestSendLog_TimestampParseError(t *testing.T) {
	t.Parallel()

	// タイムスタンプを不正な値に設定
	reqBody := rest.SendLogRequest{
		ID:        "",
		TraceID:   "",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: "invalid-timestamp", // 不正値
		Metadata:  nil,
	}

	// テスト環境をセットアップ
	handler, _, mockLogger, req, resp, echoServer := setupRestSendLogTest(t, reqBody)

	// Context を生成
	ctx := echoServer.NewContext(req, resp)

	// 実行
	err := handler.SendLog(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Timestamp parsing failed")
}

// TestSendLog_InternalError は内部エラー（ユースケース層）で500が返ることを確認するテスト
func TestSendLog_InternalError(t *testing.T) {
	t.Parallel()

	// 正常なリクエストボディを準備
	reqBody := rest.SendLogRequest{
		ID:        "",
		TraceID:   "",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  nil,
	}

	// テスト環境をセットアップ
	handler, mockUC, mockLogger, req, resp, echoServer := setupRestSendLogTest(t, reqBody)

	// Context を生成
	ctx := echoServer.NewContext(req, resp)

	// ユースケース層が失敗を返すよう設定
	mockUC.On("SendLog", mock.Anything, mock.Anything).Return(errDB)

	// 実行
	err := handler.SendLog(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.Code)

	// エラーログが出力されていることを確認
	require.Len(t, mockLogger.Errors, 1)
	require.Contains(t, mockLogger.Errors[0].Msg, "Failed to save log entry")
}

// TestSendLog_CompleteID はID未指定時にサーバー側でIDが補完されることを確認するテスト
func TestSendLog_CompleteID(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	echoServer.Validator = &dummyValidator{forceError: false}

	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// ID を空にしておく（サーバー側で自動補完されることを期待）
	reqBody := rest.SendLogRequest{
		ID:        "", // 空ID
		TraceID:   "",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  nil,
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// ユースケース内でIDが自動補完されたことを確認
	mockUC.On("SendLog", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		logArg, ok := args.Get(1).(*model.Log)
		require.True(t, ok, "expected argument to be of type *model.Log")
		require.NotEmpty(t, logArg.ID, "ID should be auto-generated")
	}).Return(nil)

	// 実行
	err = handler.SendLog(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	// 成功ログが出力されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	require.Contains(t, mockLogger.Infos[0].Msg, "Log entry saved successfully")
}

// TestSendLog_CompleteMetadata はMetadata未指定時にサーバー側で空mapが補完されることを確認するテスト
func TestSendLog_CompleteMetadata(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	echoServer.Validator = &dummyValidator{forceError: false}

	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// Metadata を nil 指定（サーバー側で空マップが補完されることを期待）
	reqBody := rest.SendLogRequest{
		ID:        "1",
		TraceID:   "",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  nil, // nil 指定
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// ユースケース内でMetadataが補完されているか確認
	mockUC.On("SendLog", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		logArg, ok := args.Get(1).(*model.Log)
		require.True(t, ok, "expected argument to be of type *model.Log")
		require.NotNil(t, logArg.Metadata, "Metadata should be initialized as empty map")
		require.Empty(t, logArg.Metadata, "Metadata should be empty")
	}).Return(nil)

	// 実行
	err = handler.SendLog(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	// 成功ログが出力されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	require.Contains(t, mockLogger.Infos[0].Msg, "Log entry saved successfully")
}

// TestGetLogs_Success は GetLogs エンドポイントが正常にログ一覧を返すケースをテスト
func TestGetLogs_Success(t *testing.T) {
	t.Parallel()

	// Echo サーバーと必要なモックをセットアップ
	echoServer := echo.New()
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// ユースケース層が正常なレスポンスを返すよう設定
	mockUC.On("GetLogs", mock.Anything, "", "", 100, 0).Return([]model.Log{
		{
			ID:        "1",
			TraceID:   "",
			Timestamp: time.Now(),
			Level:     "info",
			Service:   "test-service",
			Message:   "test",
			Metadata:  map[string]string{},
		},
	}, nil)

	// HTTP リクエストとレスポンスを準備
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// 実行
	err := handler.GetLogs(ctx)

	// 結果検証
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)
}

// TestGetLogs_InternalError はユースケース層がエラーを返した場合に500エラーが返ることをテスト
func TestGetLogs_InternalError(t *testing.T) {
	t.Parallel()

	// Echo サーバーと必要なモックをセットアップ
	echoServer := echo.New()
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// ユースケース層がエラーを返すよう設定
	mockUC.On("GetLogs", mock.Anything, "", "", 100, 0).Return(nil, assert.AnError)

	// HTTP リクエストとレスポンスを準備
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// 実行
	err := handler.GetLogs(ctx)

	// 結果検証
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.Code)

	// エラーログが出力されていることを確認
	require.Len(t, mockLogger.Errors, 1)
	require.Contains(t, mockLogger.Errors[0].Msg, "Failed to fetch logs")
}

// TestRespondJSON_Success は正常な Context で JSON レスポンスが返されることを確認するテスト
func TestRespondJSON_Success(t *testing.T) {
	t.Parallel()

	// Echoサーバーとレスポンスレコーダーをセットアップ
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

	// テスト対象関数を実行
	err := rest.RespondJSON(c, http.StatusOK, map[string]string{"message": "ok"})

	// 結果を検証
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"message":"ok"`)
}

// TestRespondJSON_PanicOnNilContext は nil の Context を渡した場合にpanicすることを確認するテスト
func TestRespondJSON_PanicOnNilContext(t *testing.T) {
	t.Parallel()

	// panicが発生するか確認
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic but code did not panic")
		}
	}()

	// nil Contextを渡す（ここでpanicが発生するはず）
	_ = rest.RespondJSON(nil, http.StatusOK, map[string]string{"message": "ok"})
}

// TestParseTimestamp_Empty は空文字列が渡された場合に現在時刻が返ることを確認するテスト
func TestParseTimestamp_Empty(t *testing.T) {
	t.Parallel()

	// 空文字列入力
	result, err := rest.ParseTimestamp("")
	require.NoError(t, err)

	// 返り値が現在時刻付近であることを確認
	require.WithinDuration(t, time.Now(), result, time.Second)
}

// TestParseTimestamp_Valid は有効なRFC3339形式の文字列が正しくパースされることを確認するテスト
func TestParseTimestamp_Valid(t *testing.T) {
	t.Parallel()

	// 有効なRFC3339形式の文字列
	input := "2025-04-06T12:00:00Z"
	result, err := rest.ParseTimestamp(input)
	require.NoError(t, err)

	// 期待される結果と一致するか確認
	expected, _ := time.Parse(time.RFC3339, input)
	require.Equal(t, expected, result)
}

// TestParseTimestamp_Invalid は無効な文字列が渡された場合にエラーが返ることを確認するテスト
func TestParseTimestamp_Invalid(t *testing.T) {
	t.Parallel()

	// 不正なタイムスタンプ文字列
	_, err := rest.ParseTimestamp("invalid-timestamp")

	// エラーが返ることを確認
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid timestamp")
}

// TestParseAndValidateRequest_Success は全て正しいリクエストが通る場合のテスト
func TestParseAndValidateRequest_Success(t *testing.T) {
	t.Parallel()

	// 正常なリクエストボディを準備
	reqBody := rest.SendLogRequest{
		ID:        "1",
		TraceID:   "trace-1",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  map[string]string{"key": "value"},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// テスト環境をセットアップ
	handler, _, resp, req, echoServer := setupParseAndValidateTest(t, body, false)
	ctx := echoServer.NewContext(req, resp)

	// 実行
	result, parsedTime, err := handler.ParseAndValidateRequest(ctx)

	// 結果を検証
	require.NoError(t, err)
	require.Equal(t, reqBody.Message, result.Message)
	require.False(t, parsedTime.IsZero(), "parsed timestamp should not be zero")
}

// TestParseAndValidateRequest_BindError は不正なJSONでバインドエラーになるケースをテスト
func TestParseAndValidateRequest_BindError(t *testing.T) {
	t.Parallel()

	// 不正なJSON文字列
	body := []byte("invalid-json")
	handler, mockLogger, resp, req, echoServer := setupParseAndValidateTest(t, body, false)
	ctx := echoServer.NewContext(req, resp)

	// 実行
	_, _, err := handler.ParseAndValidateRequest(ctx)

	// エラー内容を検証
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Failed to bind request body")
}

// TestParseAndValidateRequest_ValidationError はバリデーションエラーが発生するケースをテスト
func TestParseAndValidateRequest_ValidationError(t *testing.T) {
	t.Parallel()

	// 正常なリクエストボディ（だが dummyValidator が常にエラーを返す設定）
	reqBody := rest.SendLogRequest{
		ID:        "1",
		TraceID:   "trace-1",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  map[string]string{"key": "value"},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// テスト環境をセットアップ
	handler, mockLogger, resp, req, echoServer := setupParseAndValidateTest(t, body, true)
	ctx := echoServer.NewContext(req, resp)

	// 実行
	_, _, err = handler.ParseAndValidateRequest(ctx)

	// エラー内容を検証
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation error")

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Validation failed")
}

// TestParseAndValidateRequest_TimestampParseError はタイムスタンプが不正な場合のパースエラーをテスト
func TestParseAndValidateRequest_TimestampParseError(t *testing.T) {
	t.Parallel()

	// タイムスタンプを不正な文字列に設定
	reqBody := rest.SendLogRequest{
		ID:        "1",
		TraceID:   "trace-1",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: "invalid-timestamp",
		Metadata:  map[string]string{"key": "value"},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// テスト環境をセットアップ
	handler, mockLogger, resp, req, echoServer := setupParseAndValidateTest(t, body, false)
	ctx := echoServer.NewContext(req, resp)

	// 実行
	_, _, err = handler.ParseAndValidateRequest(ctx)

	// エラー内容を検証
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid timestamp format")

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Timestamp parsing failed")
}

// TestParseQueryParams_Success は全てのクエリパラメータが正しく指定された場合のテスト
func TestParseQueryParams_Success(t *testing.T) {
	t.Parallel()

	// テスト環境を準備（全パラメータ指定）
	handler, _, req, rec, server := setupParseQueryParamsTest(t, "service=test-service&level=info&limit=50&offset=10")
	ctx := server.NewContext(req, rec)

	// 実行
	service, level, limit, offset, err := handler.ParseQueryParams(ctx)

	// 結果検証
	require.NoError(t, err)
	require.Equal(t, "test-service", service)
	require.Equal(t, "info", level)
	require.Equal(t, 50, limit)
	require.Equal(t, 10, offset)
}

// TestParseQueryParams_DefaultLimit は limit が指定されない場合にデフォルト値が使われることを確認するテスト
func TestParseQueryParams_DefaultLimit(t *testing.T) {
	t.Parallel()

	// テスト環境を準備（limit 未指定）
	handler, _, req, rec, server := setupParseQueryParamsTest(t, "service=test-service&level=info&offset=5")
	ctx := server.NewContext(req, rec)

	// 実行
	service, level, limit, offset, err := handler.ParseQueryParams(ctx)
	require.NoError(t, err)

	_ = service
	_ = level
	_ = offset

	// 結果検証（limit はデフォルト100になるはず）
	require.NoError(t, err)
	require.Equal(t, 100, limit)
}

// TestParseQueryParams_DefaultOffset は offset が指定されない場合にデフォルト値が使われることを確認するテスト
func TestParseQueryParams_DefaultOffset(t *testing.T) {
	t.Parallel()

	// テスト環境を準備（offset 未指定）
	handler, _, req, rec, server := setupParseQueryParamsTest(t, "service=test-service&level=info&limit=20")
	ctx := server.NewContext(req, rec)

	// 実行
	service, level, limit, offset, err := handler.ParseQueryParams(ctx)
	require.NoError(t, err)

	_ = service
	_ = level
	_ = limit

	// 結果検証（offset はデフォルト0になるはず）
	require.NoError(t, err)
	require.Equal(t, 0, offset)
}

// TestParseQueryParams_InvalidLimit は limit に数値以外の文字列が渡された場合のエラーを確認するテスト
func TestParseQueryParams_InvalidLimit(t *testing.T) {
	t.Parallel()

	// テスト環境を準備（limit が不正な文字列）
	handler, mockLogger, req, rec, server := setupParseQueryParamsTest(t, "limit=abc")
	ctx := server.NewContext(req, rec)

	// 実行
	service, level, limit, offset, err := handler.ParseQueryParams(ctx)

	_ = service
	_ = level
	_ = limit
	_ = offset

	// 結果検証（エラーが発生するはず）
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid limit parameter")

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Invalid limit parameter")
}

// TestParseQueryParams_LimitOutOfRange は limit が範囲外（>1000）の場合のエラーを確認するテスト
func TestParseQueryParams_LimitOutOfRange(t *testing.T) {
	t.Parallel()

	// テスト環境を準備（limit が範囲外の数値）
	handler, mockLogger, req, rec, server := setupParseQueryParamsTest(t, "limit=2000")
	ctx := server.NewContext(req, rec)

	// 実行
	service, level, limit, offset, err := handler.ParseQueryParams(ctx)

	_ = service
	_ = level
	_ = limit
	_ = offset

	// 結果検証（エラーが発生するはず）
	require.Error(t, err)
	require.Contains(t, err.Error(), "limit must be between")

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Limit parameter out of range")
}

// TestParseQueryParams_InvalidOffset は offset に数値以外の文字列が渡された場合のエラーを確認するテスト
func TestParseQueryParams_InvalidOffset(t *testing.T) {
	t.Parallel()

	// テスト環境を準備（offset が不正な文字列）
	handler, mockLogger, req, rec, server := setupParseQueryParamsTest(t, "offset=xyz")

	ctx := server.NewContext(req, rec)

	// 実行
	service, level, limit, offset, err := handler.ParseQueryParams(ctx)

	_ = service
	_ = level
	_ = limit
	_ = offset

	// 結果検証（エラーが発生するはず）
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid offset parameter")

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Invalid offset parameter")
}

// TestParseQueryParams_OffsetNegative は offset に負の値が渡された場合のエラーを確認するテスト
func TestParseQueryParams_OffsetNegative(t *testing.T) {
	t.Parallel()

	// テスト環境を準備（offset が負の値）
	handler, mockLogger, req, rec, server := setupParseQueryParamsTest(t, "offset=-5")
	ctx := server.NewContext(req, rec)

	// 実行
	service, level, limit, offset, err := handler.ParseQueryParams(ctx)

	_ = service
	_ = level
	_ = limit
	_ = offset

	// 結果検証（エラーが発生するはず）
	require.Error(t, err)
	require.Contains(t, err.Error(), "offset must be >= 0")

	// Warnログが出力されていることを確認
	require.Len(t, mockLogger.Warns, 1)
	require.Contains(t, mockLogger.Warns[0].Msg, "Offset parameter is negative")
}

// --- TestAppErrorToHTTPStatus Tests ---

// TestAppErrorToHTTPStatus は AppErrorToHTTPStatus 関数のマッピング動作を確認するテスト
func TestAppErrorToHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string // サブテスト名
		err      error  // 入力エラー
		expected int    // 期待するHTTPステータスコード
	}{
		{
			name:     "ValidationFailure",
			err:      usecase.ErrValidationFailure,
			expected: http.StatusBadRequest,
		},
		{
			name:     "RepositoryFailure",
			err:      usecase.ErrRepositoryFailure,
			expected: http.StatusInternalServerError,
		},
		{
			name:     "NoLogsFound",
			err:      usecase.ErrNoLogsFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "UnknownError",
			err:      errUnexpected,
			expected: http.StatusInternalServerError,
		},
	}

	// 各ケースを順に検証
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			status := rest.AppErrorToHTTPStatus(testCase.err)
			require.Equal(t, testCase.expected, status,
				"error %v should map to HTTP status %d", testCase.err, testCase.expected)
		})
	}
}
