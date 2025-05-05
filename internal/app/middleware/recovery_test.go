package middleware_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// TestRecoveryHandler は RecoveryHandler 関数の挙動をテストする
func TestRecoveryHandler(t *testing.T) {
	t.Parallel()

	// モックロガーを初期化
	mockLogger := testutil.NewMockLogger()

	// テスト用の追加コンテキスト情報を準備
	contextInfo := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	// RecoveryHandler を実行（panic回収とログ記録を模擬）
	stack := middleware.RecoveryHandler(mockLogger, contextInfo)

	// スタックトレースが空ではないことを確認
	require.NotEmpty(t, stack)

	// Error ログが1件出力されていることを確認
	require.Len(t, mockLogger.Errors, 1)

	logEntry := mockLogger.Errors[0]

	// ログメッセージが期待通りであることを確認
	require.Equal(t, "panic recovered", logEntry.Msg)

	// 渡されたフィールドに stack, key1, key2 が含まれていることを確認
	foundStack := false
	foundKey1 := false
	foundKey2 := false

	for i := 0; i < len(logEntry.Args); i += 2 {
		key := logEntry.Args[i]
		value := logEntry.Args[i+1]

		if key == "stack" {
			foundStack = true

			require.Contains(t, value, "goroutine") // スタックトレースの断片を簡易チェック
		}

		if key == "key1" && value == "value1" {
			foundKey1 = true
		}

		if key == "key2" && value == 123 {
			foundKey2 = true
		}
	}

	require.True(t, foundStack, "stack field should be present")
	require.True(t, foundKey1, "key1 field should be present")
	require.True(t, foundKey2, "key2 field should be present")

	// 他のログレベルには出力がないことを確認
	require.Empty(t, mockLogger.Infos)
	require.Empty(t, mockLogger.Warns)
}
