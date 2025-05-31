package grpcmw_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpcmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/grpc"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
)

// 共通エラー定義
var ErrHandler = errors.New("handler error")

// TestRecoveryInterceptor_NormalFlow は panic やエラーがない通常フローの確認用テスト
func TestRecoveryInterceptor_NormalFlow(t *testing.T) {
	t.Parallel()

	// モックロガーを作成し、RecoveryInterceptor を取得
	mockLogger := appmock.NewLogger()
	interceptor := grpcmw.RecoveryInterceptor(mockLogger)

	// 正常に値を返すハンドラーを用意
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "success", nil
	}

	// インターセプターを実行
	resp, err := interceptor(t.Context(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/TestMethod",
		Server:     nil,
	}, handler)

	// エラーがないことを確認
	require.NoError(t, err)
	// レスポンスが期待通りであることを確認
	require.Equal(t, "success", resp)
	// panic や error がないため、Error ログは記録されないことを確認
	require.Empty(t, mockLogger.Errors)
}

// TestRecoveryInterceptor_Panic はハンドラー内で panic が発生した場合の回復テスト
func TestRecoveryInterceptor_Panic(t *testing.T) {
	t.Parallel()

	// モックロガーを作成し、RecoveryInterceptor を取得
	mockLogger := appmock.NewLogger()
	interceptor := grpcmw.RecoveryInterceptor(mockLogger)

	// panic を発生させるハンドラーを用意
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		panic("simulated panic")
	}

	// インターセプターを実行
	resp, err := interceptor(t.Context(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/TestMethod",
		Server:     nil,
	}, handler)

	// panic 回復後、レスポンスは nil でエラーが返ることを確認
	require.Nil(t, resp)
	require.Error(t, err)

	// エラーが gRPC の Internal ステータスでラップされていることを確認
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())

	// エラーログが1件記録され、内容に panic 情報が含まれていることを確認
	require.Len(t, mockLogger.Errors, 1)
	require.Contains(t, mockLogger.Errors[0].Msg, "panic recovered")
}

// TestRecoveryInterceptor_HandlerReturnsError はハンドラーがエラーを返した場合のテスト
func TestRecoveryInterceptor_HandlerReturnsError(t *testing.T) {
	t.Parallel()

	// モックロガーを作成し、RecoveryInterceptor を取得
	mockLogger := appmock.NewLogger()
	interceptor := grpcmw.RecoveryInterceptor(mockLogger)

	// エラーを返すハンドラーを用意
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, ErrHandler
	}

	// インターセプターを実行
	resp, err := interceptor(t.Context(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/TestMethod",
		Server:     nil,
	}, handler)

	// エラーが返ることを確認
	require.Nil(t, resp)
	require.Error(t, err)

	// エラーが gRPC の Internal ステータスでラップされていることを確認
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())

	// panic ではないため、Error ログは記録されていないことを確認
	require.Empty(t, mockLogger.Errors)
}
