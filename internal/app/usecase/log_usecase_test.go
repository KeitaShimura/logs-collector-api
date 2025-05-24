package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// 共通エラー定義
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
func setup() (*mockLogRepository, *testutil.MockLogger, *usecase.LogUseCaseImpl) {
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
	require.Len(t, logger.Infos, 1)
	require.Contains(t, logger.Infos[0].Msg, "Log entry saved successfully")
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

	require.Error(t, err)

	// エラーが repository failure であることを確認
	require.ErrorIs(t, err, usecase.ErrRepositoryFailure)

	// エラーログが1件記録されていることを確認
	require.Len(t, logger.Errors, 1)
	require.Contains(t, logger.Errors[0].Msg, "Failed to save log entry")
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
	require.Equal(t, expected, logs)

	// ログが2回記録されていることを確認（取得開始と取得成功）
	require.Len(t, logger.Infos, 1)
	require.Contains(t, logger.Infos[0].Msg, "Logs retrieved successfully")
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

	// エラーが repository failure であることを確認
	require.ErrorIs(t, err, usecase.ErrRepositoryFailure)

	// エラーログが1件記録されていることを確認
	require.Len(t, logger.Errors, 1)
	require.Contains(t, logger.Errors[0].Msg, "Failed to get logs")
}

// TestLogUseCase_GetLogs_NotFound はログが見つからなかった場合のテスト
func TestLogUseCase_GetLogs_NotFound(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	mockRepo, _, logUseCase := setup()

	// GetLogs が空のログリストを返すようにモック
	mockRepo.On("GetLogs", ctx, "user3", "INFO", 10, 0).Return([]model.Log{}, nil)

	// ログが見つからなかった場合、not found エラーが返されることを確認
	_, err := logUseCase.GetLogs(ctx, "user3", "INFO", 10, 0)
	require.Error(t, err)

	// エラーが not found failure であることを確認
	require.ErrorIs(t, err, usecase.ErrNoLogsFound)
}
