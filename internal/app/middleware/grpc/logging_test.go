package grpcmw_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	grpcmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/grpc"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// TestLoggingInterceptor_InfoLog は gRPC 呼び出しが正常に完了したときに Info ログが出力されることを確認する
func TestLoggingInterceptor_InfoLog(t *testing.T) {
	t.Parallel()

	// モックロガーとインターセプターを準備
	mockLogger := testutil.NewMockLogger()
	interceptor := grpcmw.LoggingInterceptor(mockLogger)

	// context にメタデータを設定（LoggingHandler 内で使用される）
	ctx := t.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyTraceID, "trace-123")
	ctx = context.WithValue(ctx, middleware.ContextKeyRequestID, "req-456")
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "user-789")
	ctx = context.WithValue(ctx, middleware.ContextKeyClientIP, "192.168.1.1")

	// 正常レスポンスを返すハンドラーを定義
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	// インターセプターを実行
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/TestMethod",
		Server:     nil,
	}, handler)

	// 結果を検証
	require.NoError(t, err)
	require.Equal(t, "ok", resp)

	// Info ログが1件記録されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	entry := mockLogger.Infos[0]
	require.Equal(t, "request completed", entry.Msg)

	// ログに含まれるフィールドを検証
	require.Contains(t, entry.Args, "trace_id")
	require.Contains(t, entry.Args, "trace-123")

	require.Contains(t, entry.Args, "request_id")
	require.Contains(t, entry.Args, "req-456")

	require.Contains(t, entry.Args, "method")
	require.Contains(t, entry.Args, "/test.TestService/TestMethod")

	require.Contains(t, entry.Args, "status_code")
	require.Contains(t, entry.Args, "OK")

	require.Contains(t, entry.Args, "duration_ms")
	testutil.AssertDurationMsFieldExists(t, entry.Args)

	require.Contains(t, entry.Args, "user_id")
	require.Contains(t, entry.Args, "user-789")

	require.Contains(t, entry.Args, "client_ip")
	require.Contains(t, entry.Args, "192.168.1.1")
}

// TestLoggingInterceptor_ErrorLog は gRPC 呼び出しがエラーで終了したときに Error ログが出力されることを確認する
func TestLoggingInterceptor_ErrorLog(t *testing.T) {
	t.Parallel()

	// モックロガーとインターセプターを準備
	mockLogger := testutil.NewMockLogger()
	interceptor := grpcmw.LoggingInterceptor(mockLogger)

	// context にメタデータを設定（LoggingHandler 内で使用される）
	ctx := t.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyTraceID, "trace-err")
	ctx = context.WithValue(ctx, middleware.ContextKeyRequestID, "req-err")
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "user-err")
	ctx = context.WithValue(ctx, middleware.ContextKeyClientIP, "10.0.0.1")

	// エラーを返すハンドラーを定義
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, status.Error(codes.Internal, "internal error")
	}

	// インターセプターを実行
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/ErrorMethod",
		Server:     nil,
	}, handler)

	// 結果を検証
	require.Nil(t, resp)
	require.Error(t, err)

	// Error ログが1件記録されていることを確認
	require.Len(t, mockLogger.Errors, 1)
	entry := mockLogger.Errors[0]
	require.Equal(t, "request failed", entry.Msg)
	require.Error(t, entry.Err)
	require.Contains(t, entry.Err.Error(), "internal error")

	// ログに含まれるフィールドを検証
	require.Contains(t, entry.Args, "trace_id")
	require.Contains(t, entry.Args, "trace-err")

	require.Contains(t, entry.Args, "request_id")
	require.Contains(t, entry.Args, "req-err")

	require.Contains(t, entry.Args, "method")
	require.Contains(t, entry.Args, "/test.TestService/ErrorMethod")

	require.Contains(t, entry.Args, "status_code")
	require.Contains(t, entry.Args, "Internal")

	require.Contains(t, entry.Args, "duration_ms")
	testutil.AssertDurationMsFieldExists(t, entry.Args)

	require.Contains(t, entry.Args, "user_id")
	require.Contains(t, entry.Args, "user-err")

	require.Contains(t, entry.Args, "client_ip")
	require.Contains(t, entry.Args, "10.0.0.1")
}
