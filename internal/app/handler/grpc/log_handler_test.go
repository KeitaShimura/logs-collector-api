package grpc_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/grpc"
	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// 共通エラー定義
var (
	errSaveFailed  = errors.New("save failed")
	errGetLogsFail = errors.New("get logs failed")
	errUnexpected  = errors.New("unexpected error")
)

// --- setup ---

// setupSendLogTest は gRPC ハンドラーと必要なモック、およびテスト用リクエストを準備するヘルパー関数
func setupSendLogTest() (*grpc.LogHandler, *testutil.MockLogUseCase, *testutil.MockLogger, *pb.SendLogRequest) {
	// モックユースケースとモックロガーを作成
	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()

	// テスト対象の gRPC ハンドラーを初期化
	handler := grpc.NewLogHandler(mockUC, mockLogger)

	// テスト用の gRPC リクエストを準備
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "1",
			TraceId:   "trace-1",
			Timestamp: timestamppb.Now(),
			Level:     "info",
			Service:   "test-service",
			Message:   "test message",
			Metadata:  map[string]string{"key": "value"},
		},
	}

	return handler, mockUC, mockLogger, req
}

// --- SendLog Tests ---

// TestSendLog_Success は gRPC の SendLog メソッドが正常にログを保存できる場合のテスト
func TestSendLog_Success(t *testing.T) {
	t.Parallel()

	// テスト環境をセットアップ
	handler, mockUC, mockLogger, req := setupSendLogTest()

	// モックユースケースに正常な応答を設定
	mockUC.On("SendLog", mock.Anything, mock.Anything).Return(nil)

	// SendLog を実行
	resp, err := handler.SendLog(t.Context(), req)

	// エラーが発生しないことを確認
	require.NoError(t, err)

	// 成功フラグが true で、エラーメッセージが空であることを確認
	require.True(t, resp.GetSuccess())
	require.Empty(t, resp.GetErrorMessage())

	// 成功ログが1件記録されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	require.Contains(t, mockLogger.Infos[0].Msg, "Log saved successfully")
}

// TestSendLog_Failure は gRPC の SendLog メソッドがログ保存に失敗した場合のテスト
func TestSendLog_Failure(t *testing.T) {
	t.Parallel()

	// テスト環境をセットアップ
	handler, mockUC, mockLogger, req := setupSendLogTest()

	// モックユースケースに失敗を返すよう設定
	mockUC.On("SendLog", mock.Anything, mock.Anything).Return(errSaveFailed)

	// SendLog を実行
	resp, err := handler.SendLog(t.Context(), req)

	// エラーが返ることを確認
	require.Error(t, err)

	// 成功フラグが false で、エラーメッセージが存在することを確認
	require.False(t, resp.GetSuccess())
	require.NotNil(t, resp.GetErrorMessage())

	// エラーログが1件出力されていることを確認
	require.Len(t, mockLogger.Errors, 1)
	require.Contains(t, mockLogger.Errors[0].Msg, "Failed to save log")
}

// TestSendLog_CompleteID は gRPC の SendLog メソッドが ID 未指定の場合に自動補完されることを確認するテスト
func TestSendLog_CompleteID(t *testing.T) {
	t.Parallel()

	// テスト環境をセットアップ
	handler, mockUC, mockLogger, req := setupSendLogTest()

	// IDを空にしておく（サーバー側で補完されることを期待）
	req.Log.Id = ""

	// モックユースケースが呼び出された際に、IDが自動生成されているか確認
	mockUC.On("SendLog", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		logArg, ok := args.Get(1).(*model.Log)
		require.True(t, ok, "expected argument to be of type *model.Log")
		require.NotEmpty(t, logArg.ID, "ID should be auto-generated")
	}).Return(nil)

	// SendLog を実行
	resp, err := handler.SendLog(t.Context(), req)

	// エラーが発生しないことを確認
	require.NoError(t, err)

	// 成功フラグとエラーメッセージを確認
	require.True(t, resp.GetSuccess())
	require.Empty(t, resp.GetErrorMessage())

	// 成功ログが1件記録されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	require.Contains(t, mockLogger.Infos[0].Msg, "Log saved successfully")
}

// TestSendLog_CompleteMetadata は gRPC の SendLog メソッドが Metadata 未指定の場合に空マップが補完されることを確認するテスト
func TestSendLog_CompleteMetadata(t *testing.T) {
	t.Parallel()

	// テスト環境をセットアップ
	handler, mockUC, mockLogger, req := setupSendLogTest()

	// Metadataをnilにしておく（サーバー側で空マップ補完されることを期待）
	req.Log.Metadata = nil

	// モックユースケースが呼び出された際に、Metadataが空マップになっているか確認
	mockUC.On("SendLog", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		logArg, ok := args.Get(1).(*model.Log)
		require.True(t, ok, "expected argument to be of type *model.Log")
		require.NotNil(t, logArg.Metadata, "Metadata should be initialized as empty map")
		require.Empty(t, logArg.Metadata, "Metadata should be an empty map")
	}).Return(nil)

	// SendLog を実行
	resp, err := handler.SendLog(t.Context(), req)

	// エラーが発生しないことを確認
	require.NoError(t, err)

	// 成功フラグとエラーメッセージを確認
	require.True(t, resp.GetSuccess())
	require.Empty(t, resp.GetErrorMessage())

	// 成功ログが1件記録されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	require.Contains(t, mockLogger.Infos[0].Msg, "Log saved successfully")
}

// --- GetLogs Tests ---

// TestGetLogs_Success は gRPC の GetLogs メソッドが正常にログを取得できる場合のテスト
func TestGetLogs_Success(t *testing.T) {
	t.Parallel()

	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	handler := grpc.NewLogHandler(mockUC, mockLogger)

	// モックログデータを準備
	mockLogs := []model.Log{
		{
			ID:        "1",
			TraceID:   "trace-1",
			Timestamp: timestamppb.Now().AsTime(),
			Level:     "info",
			Service:   "test-service",
			Message:   "test message",
			Metadata:  map[string]string{"key": "value"},
		},
	}

	// モックユースケースに成功レスポンスを設定
	mockUC.On("GetLogs", mock.Anything, "test-service", "info", 10, 0).Return(mockLogs, nil)

	req := &pb.GetLogsRequest{
		Service:   testutil.StringPtr("test-service"),
		Level:     testutil.StringPtr("info"),
		Limit:     10,
		Offset:    0,
		StartTime: nil,
		EndTime:   nil,
	}

	// GetLogs を実行
	resp, err := handler.GetLogs(t.Context(), req)

	// エラーがないことを確認
	require.NoError(t, err)

	// レスポンス内のログ件数と内容を確認
	require.Len(t, resp.GetLogs(), 1)
	require.Equal(t, "1", resp.GetLogs()[0].GetId())

	// Info ログが1件出力されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	require.Contains(t, mockLogger.Infos[0].Msg, "Logs retrieved successfully")
}

// TestGetLogs_Failure は gRPC の GetLogs メソッドがログ取得に失敗した場合のテスト
func TestGetLogs_Failure(t *testing.T) {
	t.Parallel()

	mockUC := new(testutil.MockLogUseCase)
	mockLogger := testutil.NewMockLogger()
	handler := grpc.NewLogHandler(mockUC, mockLogger)

	// モックユースケースに失敗を返すよう設定
	mockUC.On("GetLogs", mock.Anything, "test-service", "info", 10, 0).Return(nil, errGetLogsFail)

	req := &pb.GetLogsRequest{
		Service:   testutil.StringPtr("test-service"),
		Level:     testutil.StringPtr("info"),
		Limit:     10,
		Offset:    0,
		StartTime: nil,
		EndTime:   nil,
	}

	// GetLogs を実行
	resp, err := handler.GetLogs(t.Context(), req)

	// エラーが返ることを確認
	require.Error(t, err)

	// レスポンスは nil であることを確認
	require.Nil(t, resp)

	// エラーログが1件出力されていることを確認
	require.Len(t, mockLogger.Errors, 1)
	require.Contains(t, mockLogger.Errors[0].Msg, "Failed to get logs")
}

// --- TestAppErrorToGRPCCode Tests ---

// TestAppErrorToGRPCCode は AppErrorToGRPCCode 関数のマッピング動作を確認するテスト
func TestAppErrorToGRPCCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string     // サブテスト名
		err      error      // 入力エラー
		expected codes.Code // 期待するgRPCステータスコード
	}{
		{
			name:     "ValidationFailure",
			err:      usecase.ErrValidationFailure,
			expected: codes.InvalidArgument,
		},
		{
			name:     "RepositoryFailure",
			err:      usecase.ErrRepositoryFailure,
			expected: codes.Internal,
		},
		{
			name:     "NoLogsFound",
			err:      usecase.ErrNoLogsFound,
			expected: codes.NotFound,
		},
		{
			name:     "UnknownError",
			err:      errUnexpected,
			expected: codes.Unknown,
		},
	}

	// 各ケースを順に検証
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			code := grpc.AppErrorToGRPCCode(testCase.err)
			require.Equal(t, testCase.expected, code,
				"error %v should map to gRPC code %v", testCase.err, testCase.expected)
		})
	}
}
