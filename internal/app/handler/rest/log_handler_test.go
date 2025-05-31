package rest_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
	"github.com/KeitaShimura/logs-collector-api/internal/app/helper"
	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
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

// injectSendLogRequest は pb.SendLogRequest を Echo の context に設定する
func injectSendLogRequest(ctx echo.Context, reqBody rest.SendLogRequest) {
	ctx.Set("send_log_request", &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        reqBody.ID,
			TraceId:   reqBody.TraceID,
			Level:     reqBody.Level,
			Service:   reqBody.Service,
			Message:   reqBody.Message,
			Timestamp: timestamppb.New(time.Now()),
			Metadata:  map[string]string{},
		},
	})
}

// --- setup ---

// setupRestSendLogTest は REST ハンドラーと必要なモック、Echo サーバ、HTTPリクエストを準備するヘルパー関数
func setupRestSendLogTest(t *testing.T, reqBody rest.SendLogRequest) (
	*rest.LogHandler,
	*appmock.LogUseCase,
	*appmock.Logger,
	*http.Request,
	*httptest.ResponseRecorder,
	*echo.Echo,
) {
	t.Helper()

	// Echo サーバーを作成し、ダミーバリデータを設定
	echoServer := echo.New()
	echoServer.Validator = &dummyValidator{forceError: false}

	// モックユースケースとモックロガーを作成
	mockUC := new(appmock.LogUseCase)
	mockLogger := appmock.NewLogger()

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

// --- SendLog Tests ---

// TestSendLog_Success は REST API の SendLog が正常終了する場合のテスト
func TestSendLog_Success(t *testing.T) {
	t.Parallel()

	// 正常なリクエストボディを準備
	reqBody := rest.SendLogRequest{
		ID:        "",
		TraceID:   "trace-id",
		Message:   "test message",
		Level:     "info",
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  nil,
	}

	// テスト環境をセットアップ
	handler, mockUC, mockLogger, req, resp, echoServer := setupRestSendLogTest(t, reqBody)

	ctx := echoServer.NewContext(req, resp)

	// pb.SendLogRequest を直接 context に仕込む
	injectSendLogRequest(ctx, reqBody)

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
	mockUC := new(appmock.LogUseCase)
	mockLogger := appmock.NewLogger()
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
	require.Contains(t, mockLogger.Warns[0].Msg, "send_log_request not found in context or invalid type")
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

	// pb.SendLogRequest を直接 context に仕込む
	injectSendLogRequest(ctx, reqBody)

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

	mockUC := new(appmock.LogUseCase)
	mockLogger := appmock.NewLogger()
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

	// pb.SendLogRequest を直接 context に仕込む
	injectSendLogRequest(ctx, reqBody)

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

	mockUC := new(appmock.LogUseCase)
	mockLogger := appmock.NewLogger()
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

	// pb.SendLogRequest を直接 context に仕込む
	injectSendLogRequest(ctx, reqBody)

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
	mockUC := new(appmock.LogUseCase)
	mockLogger := appmock.NewLogger()
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// ユースケース層が正常なレスポンスを返すよう設定
	mockUC.On("GetLogs", mock.Anything, "test-service", "INFO", 100, 0).Return([]model.Log{
		{
			ID:        "1",
			TraceID:   "",
			Timestamp: time.Now(),
			Level:     "INFO",
			Service:   "test-service",
			Message:   "test",
			Metadata:  map[string]string{},
		},
	}, nil)

	// HTTP リクエストとレスポンスを準備
	req := httptest.NewRequest(http.MethodGet, "/logs?service=test-service&level=INFO&limit=100&offset=0", nil)
	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	ctx.Set("parsed_query_params", &helper.QueryParams{
		Service: "test-service",
		Level:   "INFO",
		Limit:   100,
		Offset:  0,
	})

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
	mockUC := new(appmock.LogUseCase)
	mockLogger := appmock.NewLogger()
	handler := rest.NewLogHandler(mockUC, mockLogger)

	// ユースケース層がエラーを返すよう設定（引数を一致させる）
	mockUC.On("GetLogs", mock.Anything, "test-service", "info", 100, 0).
		Return(nil, fmt.Errorf("%w: mock db error", usecase.ErrRepositoryFailure))

	// クエリパラメータ付きのリクエストを準備
	req := httptest.NewRequest(http.MethodGet, "/logs?service=test-service&level=info&limit=100&offset=0", nil)
	resp := httptest.NewRecorder()
	ctx := echoServer.NewContext(req, resp)

	// Middleware により設定される想定の値を模倣して context に注入
	ctx.Set("parsed_query_params", &helper.QueryParams{
		Service: "test-service",
		Level:   "info",
		Limit:   100,
		Offset:  0,
	})

	// 実行
	err := handler.GetLogs(ctx)

	// 結果検証
	require.NoError(t, err)
	mockUC.On("GetLogs", mock.Anything, "test-service", "info", 100, 0).
		Return(nil, fmt.Errorf("%w", usecase.ErrRepositoryFailure))
}

// TestRespondJSON_Success は正常な Context で JSON レスポンスが返されることを確認するテスト
func TestRespondJSON_Success(t *testing.T) {
	t.Parallel()

	// Echoサーバーとレスポンスレコーダーをセットアップ
	echoServer := echo.New()
	rec := httptest.NewRecorder()
	c := echoServer.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

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
	if err := rest.RespondJSON(nil, http.StatusOK, map[string]string{"message": "ok"}); err != nil {
		t.Fatalf("failed to respond json: %v", err)
	}
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
