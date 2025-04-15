package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

var (
	errMockDB              = errors.New("db error")
	errMockLogsNil         = errors.New("mock GetLogs returned nil logs")
	errMockLogsTypeInvalid = errors.New("mock GetLogs type assertion failed")
	ErrInvalidTimeZone     = errors.New("invalid time zone")
)

// --- Mocks ---

// mockLogRepository は LogRepository を模倣するモック構造体
type mockLogRepository struct{ mock.Mock }

// SendLog はログの永続化処理を模倣する
func (m *mockLogRepository) SendLog(ctx context.Context, log *model.Log) error {
	args := m.Called(ctx, log)

	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock SendLog error: %w", err)
	}

	return nil
}

// GetLogs はログ検索処理を模倣する
func (m *mockLogRepository) GetLogs(
	ctx context.Context,
	service, level string,
	limit, offset int,
) ([]model.Log, error) {
	args := m.Called(ctx, service, level, limit, offset)

	if args.Get(0) == nil {
		return nil, fmt.Errorf("%w", errMockLogsNil)
	}

	logs, ok := args.Get(0).([]model.Log)
	if !ok {
		return nil, fmt.Errorf("%w: got=%T", errMockLogsTypeInvalid, args.Get(0))
	}

	if args.Error(1) != nil {
		return nil, fmt.Errorf("mock GetLogs error: %w", args.Error(1))
	}

	return logs, nil
}

// --- setup ---

// setup はモックリポジトリ、モックロガー、ユースケースを初期化して返す
func setup() (*mockLogRepository, *testutil.MockLogger, *usecase.LogUseCase) {
	mockRepo := new(mockLogRepository)
	mockLogger := testutil.NewMockLogger()
	uc := usecase.NewLogUseCase(mockRepo, mockLogger)

	return mockRepo, mockLogger, uc
}

// --- SendLog Tests ---

// TestLogUseCase_SendLog_Success はログが正常に保存される場合のテスト
func TestLogUseCase_SendLog_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	mockRepo, logger, logUseCase := setup()

	// テスト用の正常なログデータを準備
	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "auth",
		Message:   "log message",
		Metadata:  map[string]string{"key": "value"},
	}

	// SendLog メソッドが正常に呼び出されることを期待
	mockRepo.On("SendLog", ctx, log).Return(nil).Once()

	// SendLog を実行し、エラーがないことを確認
	err := logUseCase.SendLog(ctx, log)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// ログが保存前後で2回記録されていることを確認
	// 保存前のログと保存後のログがそれぞれ1回ずつ記録される
	assert.Len(t, logger.Infos, 2)
	assert.Contains(t, logger.Infos[0].Msg, "Saving log entry")
	assert.Contains(t, logger.Infos[1].Msg, "Log entry saved successfully")
}

// TestLogUseCase_SendLog_ValidationError はログエントリにバリデーションエラーがある場合のテスト
func TestLogUseCase_SendLog_ValidationError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	_, _, logUseCase := setup()

	// バリデーションエラーを含むログデータを準備（Message と Level が空）
	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Now(),
		Service:   "auth",
		Level:     "",
		Message:   "",
		Metadata:  map[string]string{},
	}

	// SendLog メソッドを実行し、バリデーションエラーが発生することを確認
	err := logUseCase.SendLog(ctx, log)
	require.Error(t, err)

	// エラーステータスコードが InvalidArgument であることを確認
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestLogUseCase_SendLog_RepositoryError はリポジトリでエラーが発生した場合のテスト
func TestLogUseCase_SendLog_RepositoryError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	mockRepo, logger, logUseCase := setup()

	// エラーをシミュレートするためのログデータを準備
	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "auth",
		Message:   "should fail",
		Metadata:  map[string]string{},
	}

	// SendLog メソッドがエラーを返すようにモック
	mockRepo.On("SendLog", ctx, log).Return(errMockDB).Once()

	// SendLog を実行し、エラーが発生することを確認
	err := logUseCase.SendLog(ctx, log)

	// エラーが Internal コードで返されることを確認
	require.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	// エラーログが1件記録されていることを確認
	// "Failed to save log entry" のメッセージが含まれているか確認
	assert.Len(t, logger.Errors, 1)
	assert.Contains(t, logger.Errors[0].Msg, "Failed to save log entry")
}

// --- GetLogs Tests ---

// TestLogUseCase_GetLogs_Success はログが正常に取得される場合のテスト
func TestLogUseCase_GetLogs_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	mockRepo, logger, logUseCase := setup()

	// 期待するログデータ
	expected := []model.Log{{
		ID:        "id-1",
		TraceID:   "trace-expected",
		Service:   "user",
		Level:     "INFO",
		Message:   "test message",
		Timestamp: time.Now(),
		Metadata:  map[string]string{},
	}}

	// GetLogs が期待したログを返すようにモック
	mockRepo.On("GetLogs", ctx, "user", "INFO", 10, 0).Return(expected, nil)

	// GetLogs メソッドを実行し、エラーが発生しないことを確認
	logs, err := logUseCase.GetLogs(ctx, "user", "INFO", 10, 0)
	require.NoError(t, err)
	// 返されるログが期待した値であることを確認
	assert.Equal(t, expected, logs)

	// ログが2回記録されていることを確認（取得開始と取得成功）
	assert.Len(t, logger.Infos, 2)
}

// TestLogUseCase_GetLogs_InvalidArgs は無効な引数が渡された場合のテスト
func TestLogUseCase_GetLogs_InvalidArgs(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	_, _, uc := setup()

	// 無効な引数（limit が0、offset が負）を渡して GetLogs を実行
	_, err := uc.GetLogs(ctx, "user", "INFO", 0, -1)
	require.Error(t, err)

	// エラーのステータスコードが InvalidArgument であることを確認
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestLogUseCase_GetLogs_RepositoryError はリポジトリでエラーが発生した場合のテスト
func TestLogUseCase_GetLogs_RepositoryError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	mockRepo, logger, logUseCase := setup()

	// リポジトリでエラーを返すようにモック
	mockRepo.On("GetLogs", ctx, "user2", "INFO", 10, 0).Return([]model.Log(nil), errMockDB)

	// GetLogs メソッドを実行し、エラーが発生することを確認
	_, err := logUseCase.GetLogs(ctx, "user2", "INFO", 10, 0)
	require.Error(t, err)

	// エラーが Internal コードで返されることを確認
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	// エラーログが1件記録されていることを確認
	assert.Len(t, logger.Errors, 1)
	assert.Contains(t, logger.Errors[0].Msg, "Failed to get logs")
}

// TestLogUseCase_GetLogs_NotFound はログが見つからなかった場合のテスト
func TestLogUseCase_GetLogs_NotFound(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	mockRepo, _, logUseCase := setup()

	// GetLogs が空のログリストを返すようにモック
	mockRepo.On("GetLogs", ctx, "user3", "INFO", 10, 0).Return([]model.Log{}, nil)

	// ログが見つからなかった場合、NotFound エラーが返されることを確認
	_, err := logUseCase.GetLogs(ctx, "user3", "INFO", 10, 0)
	require.Error(t, err)

	// エラーのステータスコードが NotFound であることを確認
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

// --- validateLog Tests ---

// TestValidateLog_Success は正常系のテストケースです。
func TestValidateLog_Success(t *testing.T) {
	t.Parallel()

	// 正常なモック関数（タイムゾーンとして UTC を返す）
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return time.UTC, nil // 正常に UTC を返す
	}

	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "auth",
		Message:   "log message",
		Metadata:  map[string]string{"key": "value"},
	}

	// ValidateLog 関数を呼び出して、エラーが発生しないことを確認
	err := usecase.ValidateLog(log, timeLoadLocation)
	require.NoError(t, err)
}

// TestValidateLog_NilLog はログエントリが nil の場合の異常系テスト
func TestValidateLog_NilLog(t *testing.T) {
	t.Parallel()

	// 正常なモック関数
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return time.UTC, nil // 正常に UTC を返す
	}

	var log *model.Log

	// ValidateLog を呼び出して、nil ログがエラーになることを確認
	err := usecase.ValidateLog(log, timeLoadLocation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "log entry is nil")
}

// TestValidateLog_InvalidMessage はログメッセージが空である場合の異常系テスト
func TestValidateLog_InvalidMessage(t *testing.T) {
	t.Parallel()

	// 正常なモック関数
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return time.UTC, nil // 正常に UTC を返す
	}

	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "auth",
		Message:   "",
		Metadata:  map[string]string{"key": "value"},
	}

	// メッセージが空である場合、エラーが発生することを確認
	err := usecase.ValidateLog(log, timeLoadLocation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message must not be empty")
}

// TestValidateLog_InvalidLevel はログレベルが空である場合の異常系テスト
func TestValidateLog_InvalidLevel(t *testing.T) {
	t.Parallel()

	// 正常なモック関数
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return time.UTC, nil // 正常に UTC を返す
	}

	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Now(),
		Level:     "",
		Service:   "auth",
		Message:   "log message",
		Metadata:  map[string]string{"key": "value"},
	}

	// ログレベルが空の場合、エラーが発生することを確認
	err := usecase.ValidateLog(log, timeLoadLocation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "level must not be empty")
}

// TestValidateLog_InvalidService はサービス名が空である場合の異常系テスト
func TestValidateLog_InvalidService(t *testing.T) {
	t.Parallel()

	// 正常なモック関数
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return time.UTC, nil // 正常に UTC を返す
	}

	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "",
		Message:   "log message",
		Metadata:  map[string]string{"key": "value"},
	}

	// サービス名が空の場合、エラーが発生することを確認
	err := usecase.ValidateLog(log, timeLoadLocation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service must not be empty")
}

// TestValidateLog_InvalidIDFormat はログIDが無効な場合の異常系テスト
func TestValidateLog_InvalidIDFormat(t *testing.T) {
	t.Parallel()

	// 正常なモック関数
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return time.UTC, nil // 正常に UTC を返す
	}

	log := &model.Log{
		ID:        "invalid-uuid", // 無効なUUID形式
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "auth",
		Message:   "log message",
		Metadata:  map[string]string{"key": "value"},
	}

	// 無効なID形式の場合、エラーが発生することを確認
	err := usecase.ValidateLog(log, timeLoadLocation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ID format")
}

// TestValidateLog_InvalidTraceIDFormat は TraceID が無効な場合の異常系テスト
func TestValidateLog_InvalidTraceIDFormat(t *testing.T) {
	t.Parallel()

	// 正常なモック関数
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return time.UTC, nil // 正常に UTC を返す
	}

	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "invalid-uuid", // 無効なUUID形式
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "auth",
		Message:   "log message",
		Metadata:  map[string]string{"key": "value"},
	}

	// 無効なTraceID形式の場合、エラーが発生することを確認
	err := usecase.ValidateLog(log, timeLoadLocation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid TraceID format")
}

// TestValidateLog_ValidTimestamp は Timestamp がゼロ値の場合に補完されることを確認するテスト
func TestValidateLog_ValidTimestamp(t *testing.T) {
	t.Parallel()

	// 正常なモック関数
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return time.UTC, nil // 正常に UTC を返す
	}

	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Time{}, // ゼロ値
		Level:     "INFO",
		Service:   "auth",
		Message:   "log message",
		Metadata:  map[string]string{"key": "value"},
	}

	// Timestamp がゼロ値の場合、補完されることを確認
	err := usecase.ValidateLog(log, timeLoadLocation)
	require.NoError(t, err)
	assert.False(t, log.Timestamp.IsZero(), "Timestamp should not be zero after validation")
}

// TestValidateLog_InvalidTimeZone は無効なタイムゾーンの場合の異常系テスト
func TestValidateLog_InvalidTimeZone(t *testing.T) {
	t.Parallel()

	// 無効なタイムゾーンを返すモック関数
	timeLoadLocation := func(_ string) (*time.Location, error) {
		return nil, ErrInvalidTimeZone // 事前に定義されたエラーを返す
	}

	// ゼロ値の Timestamp をセットして無効なログエントリを作成
	log := &model.Log{
		ID:        "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee45",
		TraceID:   "a4dcd4a8-2fb7-4c6b-bb02-54a5beedee46",
		Timestamp: time.Time{}, // ゼロ値
		Level:     "INFO",
		Service:   "auth",
		Message:   "log message",
		Metadata:  map[string]string{"key": "value"},
	}

	// バリデーションを実行
	err := usecase.ValidateLog(log, timeLoadLocation)

	// エラーが発生することを確認
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid time zone")
}

// --- validateLogQueryParams Tests ---

// TestValidateLogQueryParams_Success は正常系のテストケース
func TestValidateLogQueryParams_Success(t *testing.T) {
	t.Parallel()

	err := usecase.ValidateLogQueryParams("auth", "INFO", 10, 0)
	require.NoError(t, err)
}

// TestValidateLogQueryParams_InvalidService はサービス名が空である場合の異常系テスト
func TestValidateLogQueryParams_InvalidService(t *testing.T) {
	t.Parallel()

	err := usecase.ValidateLogQueryParams("", "INFO", 10, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service must not be empty")
}

// TestValidateLogQueryParams_InvalidLevel はレベルが空である場合の異常系テスト
func TestValidateLogQueryParams_InvalidLevel(t *testing.T) {
	t.Parallel()

	err := usecase.ValidateLogQueryParams("auth", "", 10, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "level must not be empty")
}

// TestValidateLogQueryParams_InvalidLimit はリミットが0または負の値である場合の異常系テスト
func TestValidateLogQueryParams_InvalidLimit(t *testing.T) {
	t.Parallel()

	err := usecase.ValidateLogQueryParams("auth", "INFO", -1, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be greater than 0")
}

// TestValidateLogQueryParams_InvalidOffset はオフセットが負の値である場合の異常系テスト
func TestValidateLogQueryParams_InvalidOffset(t *testing.T) {
	t.Parallel()

	err := usecase.ValidateLogQueryParams("auth", "INFO", 10, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "offset must be >= 0")
}
