package grpc_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/grpc"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// 共通エラー定義
var (
	errSaveFailed  = errors.New("save failed")
	errGetLogsFail = errors.New("get logs failed")
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
	require.Contains(t, mockLogger.Infos[0].Msg, "log saved successfully")
}

// TestSendLog_Failure は gRPC の SendLog メソッドがログ保存に失敗した場合のテスト
func TestSendLog_Failure(t *testing.T) {
	t.Parallel()

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
	require.Contains(t, mockLogger.Errors[0].Msg, "failed to save log")
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
		Service:   grpc.StringPtr("test-service"),
		Level:     grpc.StringPtr("info"),
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
	require.Contains(t, mockLogger.Infos[0].Msg, "logs retrieved successfully")
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
		Service:   grpc.StringPtr("test-service"),
		Level:     grpc.StringPtr("info"),
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
	require.Contains(t, mockLogger.Errors[0].Msg, "failed to get logs")
}
