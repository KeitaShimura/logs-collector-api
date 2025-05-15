package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// AssertDurationMsFieldExists は duration_ms フィールドが存在し、int64 型かつ 0 以上であることを検証する
func AssertDurationMsFieldExists(t *testing.T, args []any) {
	t.Helper()

	var durationFound bool

	// 引数 args を "キー → 値" のペアとして順番に確認する
	for i := 0; i < len(args)-1; i += 2 {
		key := args[i]

		// duration_ms キーを検出した場合にのみ検証を実施
		if key == "duration_ms" {
			val := args[i+1]

			// 値が int64 型であることを検証
			ms, ok := val.(int64)
			require.True(t, ok, "duration_ms value should be int64")

			// 値が 0 以上であることを検証
			require.GreaterOrEqual(t, ms, int64(0))

			durationFound = true

			break
		}
	}

	// duration_ms キーが存在していたことを検証
	require.True(t, durationFound, "duration_ms field should be present in log")
}
