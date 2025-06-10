package integration_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	testhelper "github.com/KeitaShimura/logs-collector-api/internal/testutil/helper"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// TestGRPC_SendLog は、SendLog RPC の正常系を検証します。
func TestGRPC_SendLog(t *testing.T) {
	t.Parallel()

	// gRPC クライアント、モックProducer/Searcherをセットアップ（DBはクローズしない）
	client, _, _, mockProducer, mockSearcher := testhelper.SetupGRPCTestHandler(t, false)

	// テスト用リクエスト生成
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        uuid.NewString(),
			Service:   "test-service",
			Level:     "INFO",
			Message:   "integration test log",
			TraceId:   "trace-123",
			Timestamp: timestamppb.Now(),
			Metadata:  map[string]string{"env": "test"},
		},
	}

	// RPC 呼び出しとレスポンス検証
	resp, err := client.SendLog(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.GetSuccess())

	// モックNATSへのパブリッシュ検証
	require.Len(t, mockProducer.PublishedMessages, 1)
	published := mockProducer.PublishedMessages[0]
	require.Equal(t, req.GetLog().GetId(), published.ID)

	// モックElasticsearch検索呼び出し検証
	require.Len(t, mockSearcher.Calls, 1)
	require.Equal(t, "logs-index", mockSearcher.Calls[0].Index)
	require.Contains(t, mockSearcher.Calls[0].LogData["message"], "integration test log")
}

// TestGRPC_SendLog_DuplicateID は、重複IDによるエラーを検証します。
func TestGRPC_SendLog_DuplicateID(t *testing.T) {
	t.Parallel()

	client, _, _, mockProducer, mockSearcher := testhelper.SetupGRPCTestHandler(t, false)

	// 同一IDで2回リクエストを投げる
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        uuid.NewString(),
			Service:   "test-service",
			Level:     "INFO",
			Message:   "first message",
			TraceId:   "trace-123",
			Timestamp: timestamppb.Now(),
			Metadata:  map[string]string{},
		},
	}

	// 1回目は成功
	resp, err := client.SendLog(t.Context(), req)
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())

	// 2回目は重複エラー
	_, err = client.SendLog(t.Context(), req)
	require.Error(t, err)

	// モック呼び出し回数は1回のみであることを確認
	// モックNATSへのパブリッシュ検証
	require.Len(t, mockProducer.PublishedMessages, 1)
	published := mockProducer.PublishedMessages[0]
	require.Equal(t, req.GetLog().GetId(), published.ID)

	// モックElasticsearch検索呼び出し検証
	require.Len(t, mockSearcher.Calls, 1)
	require.Equal(t, "logs-index", mockSearcher.Calls[0].Index)
	require.Contains(t, mockSearcher.Calls[0].LogData["message"], "first message")
}

// TestGRPC_SendLog_DBConnectionFailure は、DB接続障害時に Internal エラーとなることを検証します。
func TestGRPC_SendLog_DBConnectionFailure(t *testing.T) {
	t.Parallel()

	// DBを即時クローズしてセットアップ
	client, _, _, mockProducer, mockSearcher := testhelper.SetupGRPCTestHandler(t, true)

	// リクエスト生成
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        uuid.NewString(),
			Service:   "test-service",
			Level:     "INFO",
			Message:   "should fail",
			TraceId:   "trace-err",
			Timestamp: timestamppb.Now(),
			Metadata:  map[string]string{"env": "test"},
		},
	}

	// RPC 呼び出し
	resp, err := client.SendLog(t.Context(), req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())
	require.Nil(t, resp)

	// モック呼び出しは行われていないことを確認
	require.Empty(t, mockProducer.PublishedMessages)
	require.Empty(t, mockSearcher.Calls)
}

// TestGRPC_SendLog_MissingService は、Service 欄が空の場合に InvalidArgument エラーとなることを検証します。
func TestGRPC_SendLog_MissingService(t *testing.T) {
	t.Parallel()

	client, _, _, mockProducer, mockSearcher := testhelper.SetupGRPCTestHandler(t, false)

	// Service が空文字のリクエスト
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        uuid.NewString(),
			Service:   "", // Service 未指定
			Level:     "INFO",
			Message:   "missing service",
			TraceId:   "trace-123",
			Timestamp: timestamppb.Now(),
			Metadata:  map[string]string{},
		},
	}

	// RPC 呼び出し
	resp, err := client.SendLog(t.Context(), req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Nil(t, resp)

	// モック呼び出しは行われていないことを確認
	require.Empty(t, mockProducer.PublishedMessages)
	require.Empty(t, mockSearcher.Calls)
}

// TestGRPC_SendLog_MetadataNil は、Metadata が nil の場合でも正常に動作することを検証します。
func TestGRPC_SendLog_MetadataNil(t *testing.T) {
	t.Parallel()

	client, _, _, mockProducer, mockSearcher := testhelper.SetupGRPCTestHandler(t, false)

	// Metadata を nil に設定したリクエスト
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        uuid.NewString(),
			Service:   "test-service",
			Level:     "INFO",
			Message:   "metadata is nil",
			TraceId:   "trace-xyz",
			Timestamp: timestamppb.Now(),
			Metadata:  nil,
		},
	}

	// RPC 呼び出しと結果検証
	resp, err := client.SendLog(t.Context(), req)
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())

	// モックNATSへのパブリッシュ検証
	require.Len(t, mockProducer.PublishedMessages, 1)
	published := mockProducer.PublishedMessages[0]
	require.Equal(t, req.GetLog().GetId(), published.ID)

	// モックElasticsearch検索呼び出し検証
	require.Len(t, mockSearcher.Calls, 1)
	require.Equal(t, "logs-index", mockSearcher.Calls[0].Index)
	require.Contains(t, mockSearcher.Calls[0].LogData["message"], "metadata is nil")
}
