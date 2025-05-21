package middleware_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// TestValidateSendLogRequest_Valid は、全ての項目が正しい場合にバリデーションエラーが発生しないことを検証する
func TestValidateSendLogRequest_Valid(t *testing.T) {
	t.Parallel()

	timestamp := timestamppb.New(time.Now())

	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "trace-123",
			Timestamp: timestamp,
			Level:     "INFO",
			Service:   "auth-service",
			Message:   "login success",
			Metadata:  map[string]string{},
		},
	}

	err := middleware.ValidateSendLogRequest(req)
	require.NoError(t, err)
}

// TestValidateSendLogRequest_NilLog は、Log が nil の場合にエラーが返されることを検証する
func TestValidateSendLogRequest_NilLog(t *testing.T) {
	t.Parallel()

	err := middleware.ValidateSendLogRequest(&pb.SendLogRequest{Log: nil})
	require.EqualError(t, err, "log is required")
}

// TestValidateSendLogRequest_EmptyService は、Service が空の場合にエラーが返されることを検証する
func TestValidateSendLogRequest_EmptyService(t *testing.T) {
	t.Parallel()

	timestamp := timestamppb.New(time.Now())
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "abc",
			Timestamp: timestamp,
			Level:     "INFO",
			Service:   "",
			Message:   "msg",
			Metadata:  map[string]string{},
		},
	}

	err := middleware.ValidateSendLogRequest(req)
	require.EqualError(t, err, "log.service must not be empty")
}

// TestValidateSendLogRequest_EmptyMessage は、Message が空の場合にエラーが返されることを検証する
func TestValidateSendLogRequest_EmptyMessage(t *testing.T) {
	t.Parallel()

	timestamp := timestamppb.New(time.Now())
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "abc",
			Timestamp: timestamp,
			Level:     "INFO",
			Service:   "svc",
			Message:   "",
			Metadata:  map[string]string{},
		},
	}

	err := middleware.ValidateSendLogRequest(req)
	require.EqualError(t, err, "log.message must not be empty")
}

// TestValidateSendLogRequest_EmptyLevel は、Level が空の場合にエラーが返されることを検証する
func TestValidateSendLogRequest_EmptyLevel(t *testing.T) {
	t.Parallel()

	timestamp := timestamppb.New(time.Now())
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "abc",
			Timestamp: timestamp,
			Level:     "",
			Service:   "svc",
			Message:   "msg",
			Metadata:  map[string]string{},
		},
	}

	err := middleware.ValidateSendLogRequest(req)
	require.EqualError(t, err, "log.level must not be empty")
}

// TestValidateSendLogRequest_InvalidLevel は、不正なログレベルが指定された場合にエラーが返されることを検証する
func TestValidateSendLogRequest_InvalidLevel(t *testing.T) {
	t.Parallel()

	timestamp := timestamppb.New(time.Now())
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "abc",
			Timestamp: timestamp,
			Level:     "TRACE",
			Service:   "svc",
			Message:   "msg",
			Metadata:  map[string]string{},
		},
	}

	err := middleware.ValidateSendLogRequest(req)
	require.EqualError(t, err, "invalid log.level: TRACE")
}

// TestValidateSendLogRequest_EmptyTraceID は、TraceId が空の場合にエラーが返されることを検証する
func TestValidateSendLogRequest_EmptyTraceID(t *testing.T) {
	t.Parallel()

	timestamp := timestamppb.New(time.Now())
	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "",
			Timestamp: timestamp,
			Level:     "INFO",
			Service:   "svc",
			Message:   "msg",
			Metadata:  map[string]string{},
		},
	}

	err := middleware.ValidateSendLogRequest(req)
	require.EqualError(t, err, "log.trace_id must not be empty")
}

// TestValidateSendLogRequest_NilTimestamp は、Timestamp が nil の場合にエラーが返されることを検証する
func TestValidateSendLogRequest_NilTimestamp(t *testing.T) {
	t.Parallel()

	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "abc",
			Timestamp: nil,
			Level:     "INFO",
			Service:   "svc",
			Message:   "msg",
			Metadata:  map[string]string{},
		},
	}

	err := middleware.ValidateSendLogRequest(req)
	require.EqualError(t, err, "log.timestamp is required")
}

// TestValidateSendLogRequest_FutureTimestamp は、Timestamp が未来の日付の場合にエラーが返されることを検証する
func TestValidateSendLogRequest_FutureTimestamp(t *testing.T) {
	t.Parallel()

	req := &pb.SendLogRequest{
		Log: &pb.Log{
			Id:        "",
			TraceId:   "abc",
			Timestamp: timestamppb.New(time.Now().Add(2 * time.Minute)),
			Level:     "INFO",
			Service:   "svc",
			Message:   "msg",
			Metadata:  map[string]string{},
		},
	}

	err := middleware.ValidateSendLogRequest(req)
	require.EqualError(t, err, "log.timestamp cannot be in the future")
}

// TestValidateGetLogsRequest_Valid は、有効なリクエストがエラーなくバリデーションを通過することを検証する
func TestValidateGetLogsRequest_Valid(t *testing.T) {
	t.Parallel()

	err := middleware.ValidateGetLogsRequest("auth-service", "DEBUG", 10, 0)
	require.NoError(t, err)

	err = middleware.ValidateGetLogsRequest("svc", "", 50, 0) // level空でもOK
	require.NoError(t, err)
}

// TestValidateGetLogsRequest_EmptyService は、service が空の場合にエラーが返されることを検証する
func TestValidateGetLogsRequest_EmptyService(t *testing.T) {
	t.Parallel()

	err := middleware.ValidateGetLogsRequest("", "INFO", 10, 0)
	require.EqualError(t, err, "service must not be empty")
}

// TestValidateGetLogsRequest_InvalidLevel は、無効なログレベルが指定された場合にエラーが返されることを検証する
func TestValidateGetLogsRequest_InvalidLevel(t *testing.T) {
	t.Parallel()

	err := middleware.ValidateGetLogsRequest("svc", "TRACE", 10, 0)
	require.EqualError(t, err, "invalid log.level: TRACE")
}

// TestValidateGetLogsRequest_LimitTooLow は、limit が 0 以下の場合にエラーが返されることを検証する
func TestValidateGetLogsRequest_LimitTooLow(t *testing.T) {
	t.Parallel()

	err := middleware.ValidateGetLogsRequest("svc", "INFO", 0, 0)
	require.EqualError(t, err, "limit must be between 1 and 1000")
}

// TestValidateGetLogsRequest_LimitTooHigh は、limit が 1001 以上の場合にエラーが返されることを検証する
func TestValidateGetLogsRequest_LimitTooHigh(t *testing.T) {
	t.Parallel()

	err := middleware.ValidateGetLogsRequest("svc", "INFO", 1001, 0)
	require.EqualError(t, err, "limit must be between 1 and 1000")
}

// TestValidateGetLogsRequest_NegativeOffset は、offset が負の値の場合にエラーが返されることを検証する
func TestValidateGetLogsRequest_NegativeOffset(t *testing.T) {
	t.Parallel()

	err := middleware.ValidateGetLogsRequest("svc", "INFO", 10, -1)
	require.EqualError(t, err, "offset must be >= 0")
}

// TestIsValidLogLevel は、isValidLogLevel が有効・無効なレベルを正しく判定することを検証する
func TestIsValidLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level    string
		expected bool
	}{
		{"DEBUG", true},
		{"INFO", true},
		{"WARN", true},
		{"ERROR", true},
		{"TRACE", false},
		{"FATAL", false},
		{"", false},
		{"debug", false}, // 大文字小文字の違いにも反応する
	}

	for _, tt := range tests {
		actual := middleware.IsValidLogLevel(tt.level)
		require.Equalf(t, tt.expected, actual, "expected %v for level %q", tt.expected, tt.level)
	}
}
