package middleware_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
)

// 共通エラー定義
var (
	errShouldNotBeCalled = errors.New("timeout fallback should not be called")
	errHandlerFailure    = errors.New("handler error")
	errTimeoutOccurred   = errors.New("timeout occurred")
)

// TestWithTimeout_Success は handler がタイムアウト前に正常終了する場合の挙動を検証する。
// このケースでは onTimeout は呼び出されず、Warn ログも出力されない。
func TestWithTimeout_Success(t *testing.T) {
	t.Parallel()

	mockLogger := appmock.NewLogger()

	err := middleware.WithTimeout(
		t.Context(),
		mockLogger,
		func(_ context.Context) error {
			time.Sleep(10 * time.Millisecond) // 処理はすぐに完了する

			return nil
		},
		func() error {
			return errShouldNotBeCalled // 呼ばれてはいけない
		},
	)

	require.NoError(t, err)
	require.Empty(t, mockLogger.Warns, "no warning logs should be emitted")
}

// TestWithTimeout_TimeoutOccurs は handler が処理に時間がかかり、タイムアウトに到達するケースを検証する。
// onTimeout が呼び出され、その返り値が最終的なエラーとして返却される。
func TestWithTimeout_TimeoutOccurs(t *testing.T) {
	t.Parallel()

	mockLogger := appmock.NewLogger()
	expectedTimeoutErr := errTimeoutOccurred

	// 明示的にタイムアウト付きコンテキストを作成
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	err := middleware.WithTimeout(
		ctx,
		mockLogger,
		func(_ context.Context) error {
			time.Sleep(50 * time.Millisecond) // コンテキストの deadline を超える

			return nil
		},
		func() error {
			return expectedTimeoutErr
		},
	)

	require.ErrorIs(t, err, expectedTimeoutErr)

	require.Len(t, mockLogger.Warns, 1)
	require.Equal(t, "timeout exceeded", mockLogger.Warns[0].Msg)
}

// TestWithTimeout_HandlerError は handler が即座にエラーを返すケースを検証する。
// onTimeout は呼び出されず、返されたエラーがそのまま返却される。
func TestWithTimeout_HandlerError(t *testing.T) {
	t.Parallel()

	mockLogger := appmock.NewLogger()
	expectedErr := errHandlerFailure

	err := middleware.WithTimeout(
		t.Context(),
		mockLogger,
		func(_ context.Context) error {
			return expectedErr // 処理内でエラーを返す
		},
		func() error {
			return errShouldNotBeCalled // 呼ばれてはいけない
		},
	)

	require.ErrorIs(t, err, expectedErr)
	require.Empty(t, mockLogger.Warns, 0)
}

// TestWithTimeout_ContextCanceled は、親 context が事前にキャンセルされた場合の動作を検証する。
// handler や onTimeout は呼ばれず、context canceled エラーが返される。
func TestWithTimeout_ContextCanceled(t *testing.T) {
	t.Parallel()

	mockLogger := appmock.NewLogger()
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // 直ちにキャンセルする

	err := middleware.WithTimeout(
		ctx,
		mockLogger,
		func(_ context.Context) error {
			return nil // 実行されない想定
		},
		func() error {
			return errShouldNotBeCalled // 呼ばれてはいけない
		},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "context canceled")
	require.Empty(t, mockLogger.Warns, "no warning logs should be emitted")
}
