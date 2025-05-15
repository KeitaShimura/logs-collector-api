package middleware_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// 共通のエラーインスタンスを定義（err113回避のため）
var errSomethingFailed = errors.New("something failed")

// TestLoggingHandler_Info は LoggingHandler が正常系リクエストを Info ログとして記録することを確認する
func TestLoggingHandler_Info(t *testing.T) {
	t.Parallel()

	mockLogger := testutil.NewMockLogger()

	// コンテキストに全てのメタデータをセット
	ctx := t.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyTraceID, "trace-123")
	ctx = context.WithValue(ctx, middleware.ContextKeyRequestID, "req-456")
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "user-789")
	ctx = context.WithValue(ctx, middleware.ContextKeyClientIP, "192.168.1.1")

	method := "GET /logs"
	statusCode := "200 OK"
	duration := 120 * time.Millisecond

	// LoggingHandler を呼び出し
	middleware.LoggingHandler(ctx, mockLogger, method, statusCode, duration, nil)

	// Info ログが1件出力されていることを確認
	require.Len(t, mockLogger.Infos, 1)
	require.Empty(t, mockLogger.Errors)
	require.Empty(t, mockLogger.Warns)

	entry := mockLogger.Infos[0]
	require.Equal(t, "request completed", entry.Msg)

	// ログ出力内容に各フィールドが含まれていることを検証
	require.Contains(t, entry.Args, "trace_id")
	require.Contains(t, entry.Args, "trace-123")

	require.Contains(t, entry.Args, "request_id")
	require.Contains(t, entry.Args, "req-456")

	require.Contains(t, entry.Args, "method")
	require.Contains(t, entry.Args, method)

	require.Contains(t, entry.Args, "status_code")
	require.Contains(t, entry.Args, statusCode)

	require.Contains(t, entry.Args, "duration_ms")
	require.Contains(t, entry.Args, duration.Milliseconds())

	require.Contains(t, entry.Args, "user_id")
	require.Contains(t, entry.Args, "user-789")

	require.Contains(t, entry.Args, "client_ip")
	require.Contains(t, entry.Args, "192.168.1.1")
}

// TestLoggingHandler_Error は LoggingHandler がエラー発生時に Error ログとして記録することを確認する
func TestLoggingHandler_Error(t *testing.T) {
	t.Parallel()

	mockLogger := testutil.NewMockLogger()

	// すべてのメタデータを context にセット
	ctx := t.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyTraceID, "trace-error")
	ctx = context.WithValue(ctx, middleware.ContextKeyRequestID, "req-error")
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "user-error")
	ctx = context.WithValue(ctx, middleware.ContextKeyClientIP, "10.0.0.1")

	method := "POST /fail"
	statusCode := "500 Internal Server Error"
	duration := 300 * time.Millisecond
	err := errSomethingFailed

	middleware.LoggingHandler(ctx, mockLogger, method, statusCode, duration, err)

	require.Len(t, mockLogger.Errors, 1)
	require.Empty(t, mockLogger.Infos)
	require.Empty(t, mockLogger.Warns)

	entry := mockLogger.Errors[0]
	require.Equal(t, "request failed", entry.Msg)
	require.Equal(t, err, entry.Err)

	// ログ出力にすべてのフィールドが含まれていることを確認
	require.Contains(t, entry.Args, "trace_id")
	require.Contains(t, entry.Args, "trace-error")

	require.Contains(t, entry.Args, "request_id")
	require.Contains(t, entry.Args, "req-error")

	require.Contains(t, entry.Args, "method")
	require.Contains(t, entry.Args, method)

	require.Contains(t, entry.Args, "status_code")
	require.Contains(t, entry.Args, statusCode)

	require.Contains(t, entry.Args, "duration_ms")
	require.Contains(t, entry.Args, duration.Milliseconds())

	require.Contains(t, entry.Args, "user_id")
	require.Contains(t, entry.Args, "user-error")

	require.Contains(t, entry.Args, "client_ip")
	require.Contains(t, entry.Args, "10.0.0.1")
}

// TestLoggingHandler_EmptyFieldsAreIncluded は未設定のメタ情報も空文字でログに含まれることを確認する
func TestLoggingHandler_EmptyFieldsAreIncluded(t *testing.T) {
	t.Parallel()

	mockLogger := testutil.NewMockLogger()

	ctx := t.Context() // context に何もセットしない

	method := "GET /logs"
	statusCode := "200 OK"
	duration := 150 * time.Millisecond

	middleware.LoggingHandler(ctx, mockLogger, method, statusCode, duration, nil)

	require.Len(t, mockLogger.Infos, 1)
	entry := mockLogger.Infos[0]

	require.Equal(t, "request completed", entry.Msg)

	// 空文字として出力されるフィールドが含まれることを確認
	require.Contains(t, entry.Args, "trace_id")
	require.Contains(t, entry.Args, "")

	require.Contains(t, entry.Args, "request_id")
	require.Contains(t, entry.Args, "")

	require.Contains(t, entry.Args, "method")
	require.Contains(t, entry.Args, method)

	require.Contains(t, entry.Args, "status_code")
	require.Contains(t, entry.Args, statusCode)

	require.Contains(t, entry.Args, "duration_ms")
	require.Contains(t, entry.Args, duration.Milliseconds())

	require.Contains(t, entry.Args, "user_id")
	require.Contains(t, entry.Args, "")

	require.Contains(t, entry.Args, "client_ip")
	require.Contains(t, entry.Args, "")
}

// Test_GetStringFromContext_StringValueExists は文字列型の値が正しく取得できることを確認する
func Test_GetStringFromContext_StringValueExists(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(t.Context(), middleware.ContextKeyTraceID, "abc123")
	actual := middleware.GetStringFromContext(ctx, middleware.ContextKeyTraceID)
	require.Equal(t, "abc123", actual)
}

// Test_GetStringFromContext_ValueIsNotString はコンテキストの値が string 以外の場合に空文字が返ることを確認する
func Test_GetStringFromContext_ValueIsNotString(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(t.Context(), middleware.ContextKeyUserID, 123) // int型で格納
	actual := middleware.GetStringFromContext(ctx, middleware.ContextKeyUserID)
	require.Equal(t, "", actual)
}

// Test_GetStringFromContext_KeyNotFound は存在しないキーに対して空文字が返ることを確認する
func Test_GetStringFromContext_KeyNotFound(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	actual := middleware.GetStringFromContext(ctx, "missing") // 不正なキー
	require.Equal(t, "", actual)
}
