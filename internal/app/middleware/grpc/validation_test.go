// internal/interface/grpcmw/validation_test.go
package grpcmw_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	grpcmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/grpc"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// TestValidationInterceptor_UnknownType は、バリデーション対象外の型が渡された場合にハンドラがそのまま実行されることを検証する
func TestValidationInterceptor_UnknownType(t *testing.T) {
	t.Parallel()

	logger := appmock.NewLogger()
	interceptor := grpcmw.ValidationInterceptor(logger)

	// 対象外の型 → handlerがそのまま呼ばれる
	resp, err := interceptor(t.Context(), "some string", nil, func(_ context.Context, _ interface{}) (interface{}, error) {
		return "passed", nil
	})

	require.NoError(t, err)
	require.Equal(t, "passed", resp)
	require.Empty(t, logger.Warns) // 警告ログが出ていないことも確認
}

// TestValidationInterceptor_SendLogRequest_Valid は、SendLogRequest が正しい場合にハンドラが正常に実行されることを検証する
func TestValidationInterceptor_SendLogRequest_Valid(t *testing.T) {
	t.Parallel()

	logger := appmock.NewLogger()
	interceptor := grpcmw.ValidationInterceptor(logger)

	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "log-id",
			TraceId:   "trace-123",
			Timestamp: timestamppb.Now(),
			Level:     "INFO",
			Service:   "svc",
			Message:   "msg",
			Metadata:  map[string]string{},
		},
	}

	resp, err := interceptor(t.Context(), req, nil, func(_ context.Context, _ interface{}) (interface{}, error) {
		return "OK", nil
	})

	require.NoError(t, err)
	require.Equal(t, "OK", resp)
}

// TestValidationInterceptor_SendLogRequest_Invalid は、SendLogRequest の TraceId が空の場合にバリデーションエラーとログ出力が発生することを検証する
func TestValidationInterceptor_SendLogRequest_Invalid(t *testing.T) {
	t.Parallel()

	logger := appmock.NewLogger()
	interceptor := grpcmw.ValidationInterceptor(logger)

	// TraceIdが空 → バリデーションエラーになるはず
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "",
			Timestamp: timestamppb.Now(),
			Level:     "INFO",
			Service:   "svc",
			Message:   "msg",
			Metadata:  map[string]string{},
		},
	}

	resp, err := interceptor(t.Context(), req, nil, nil)
	require.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Contains(t, st.Message(), "trace_id")

	// ログ出力の検証
	require.Len(t, logger.Warns, 1)
	require.Contains(t, logger.Warns[0].Msg, "SendLog validation failed")
}

// TestValidationInterceptor_GetLogsRequest_Valid は、GetLogsRequest が正しい内容である場合にハンドラが正常に実行されることを検証する
func TestValidationInterceptor_GetLogsRequest_Valid(t *testing.T) {
	t.Parallel()

	logger := appmock.NewLogger()
	interceptor := grpcmw.ValidationInterceptor(logger)

	req := &pb.GetLogsRequest{
		Service:   testutil.StringPtr("svc"),
		Level:     testutil.StringPtr("INFO"),
		Limit:     50,
		Offset:    0,
		StartTime: nil,
		EndTime:   nil,
	}

	resp, err := interceptor(t.Context(), req, nil, func(_ context.Context, _ interface{}) (interface{}, error) {
		return "OK", nil
	})

	require.NoError(t, err)
	require.Equal(t, "OK", resp)
	require.Empty(t, logger.Warns) // 警告ログが出ていないことも確認
}

// TestValidationInterceptor_GetLogsRequest_Invalid は、GetLogsRequest の Limit が不正な場合にバリデーションエラーとログ出力が発生することを検証する
func TestValidationInterceptor_GetLogsRequest_Invalid(t *testing.T) {
	t.Parallel()

	logger := appmock.NewLogger()
	interceptor := grpcmw.ValidationInterceptor(logger)

	// Limit が負 → バリデーションエラー
	req := &pb.GetLogsRequest{
		Service:   testutil.StringPtr("svc"),
		Level:     testutil.StringPtr("INFO"),
		Limit:     -1,
		Offset:    0,
		StartTime: nil,
		EndTime:   nil,
	}

	resp, err := interceptor(t.Context(), req, nil, nil)
	require.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Contains(t, st.Message(), "limit")

	// ログ出力の検証
	require.Len(t, logger.Warns, 1)
	require.Contains(t, logger.Warns[0].Msg, "GetLogs validation failed")
}
