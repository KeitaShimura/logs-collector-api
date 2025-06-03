package grpcmw_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpcmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/grpc"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
)

// 共通エラー定義
var errGRPCHandlerFailure = errors.New("handler failed")

// TestTimeoutInterceptor_Success は、タイムアウトせず handler が正常に完了するケース
func TestTimeoutInterceptor_Success(t *testing.T) {
	t.Parallel()

	mockLogger := appmock.NewLogger()
	interceptor := grpcmw.TimeoutInterceptor(100*time.Millisecond, mockLogger)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)

		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/QuickMethod",
		Server:     nil,
	}

	resp, err := interceptor(t.Context(), nil, info, handler)

	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.Empty(t, mockLogger.Warns, "no warning logs should be emitted")
}

// TestTimeoutInterceptor_Timeout は、gRPC タイムアウト発生時に正しくエラーとログが出力されることを検証する
func TestTimeoutInterceptor_Timeout(t *testing.T) {
	t.Parallel()

	mockLogger := appmock.NewLogger()
	interceptor := grpcmw.TimeoutInterceptor(10*time.Millisecond, mockLogger)

	// 長時間かかる handler を設定し、タイムアウトさせる
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		select {
		case <-time.After(50 * time.Millisecond):
			return "delayed-response", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/SlowMethod",
		Server:     nil,
	}

	resp, err := interceptor(t.Context(), nil, info, handler)

	// タイムアウト時はエラーとして返る
	require.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.DeadlineExceeded, st.Code())
	require.Contains(t, st.Message(), "server-side timeout")

	// Warnログが1つ出力されている
	require.Len(t, mockLogger.Warns, 1)
	require.Equal(t, "gRPC timeout", mockLogger.Warns[0].Msg)
}

// TestTimeoutInterceptor_HandlerError は、handler 自体がエラーを返す場合の動作を検証する
func TestTimeoutInterceptor_HandlerError(t *testing.T) {
	t.Parallel()

	mockLogger := appmock.NewLogger()
	interceptor := grpcmw.TimeoutInterceptor(100*time.Millisecond, mockLogger)

	expectedErr := errGRPCHandlerFailure
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/FailingMethod",
		Server:     nil,
	}

	resp, err := interceptor(t.Context(), nil, info, handler)

	require.Nil(t, resp)
	require.ErrorIs(t, err, expectedErr)
	require.Empty(t, mockLogger.Warns, "no warning logs should be emitted")
}
