package logger_test

import (
	"bytes"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

var errSomethingBroke = errors.New("something broke")

// TestLogger_InfoWarnErrorDebugOutput は各ログレベルの出力内容を検証するテスト
func TestLogger_DebugInfoWarnErrorOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.NewLogger(logger.WithWriter(&buf), logger.WithLevel(logger.LevelDebug))

	log.Debug("debug message", "debugKey", "debugVal")
	log.Info("info message", "infoKey", "infoVal")
	log.Warn("warn message", "warnKey", true)
	log.Error("error message", nil, "errorCode", 500)

	output := buf.String()

	require.Contains(t, output, `"msg":"debug message"`)
	require.Contains(t, output, `"debugKey":"debugVal"`)

	require.Contains(t, output, `"msg":"info message"`)
	require.Contains(t, output, `"infoKey":"infoVal"`)

	require.Contains(t, output, `"msg":"warn message"`)
	require.Contains(t, output, `"warnKey":true`)

	require.Contains(t, output, `"msg":"error message"`)
	require.Contains(t, output, `"errorCode":500`)
}

// TestLogger_ErrorWithNil は err が nil の場合に "error" フィールドが含まれないことを確認する
func TestLogger_ErrorWithNil(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.NewLogger(logger.WithWriter(&buf))

	log.Error("error without error", nil, "ctx", "test")
	require.Contains(t, buf.String(), `"msg":"error without error"`)
	require.NotContains(t, buf.String(), `"error":`)
	require.Contains(t, buf.String(), `"ctx":"test"`)
}

// TestLogger_ErrorWithNonNilError は error が非 nil のときに "error" フィールドが出力されることを検証する
func TestLogger_ErrorWithNonNilError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.NewLogger(logger.WithWriter(&buf), logger.WithLevel(logger.LevelDebug))

	log.Error("some error message", errSomethingBroke, "foo", "bar")

	output := buf.String()
	require.Contains(t, output, `"msg":"some error message"`)
	require.Contains(t, output, `"error":"something broke"`)
	require.Contains(t, output, `"foo":"bar"`)
}

// TestLogger_LevelFiltering はログレベルが Warn のときに Debug や Info が出力されないことを検証する
func TestLogger_LevelFiltering(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.NewLogger(
		logger.WithWriter(&buf),
		logger.WithLevel(slog.LevelWarn), // Info, Debug は出力されない
	)

	log.Debug("debug message")      // 出力されないはず
	log.Info("info message")        // 出力されないはず
	log.Warn("warn message")        // 出力される
	log.Error("error message", nil) // 出力される

	output := buf.String()
	require.NotContains(t, output, "debug message")
	require.NotContains(t, output, "info message")
	require.Contains(t, output, "warn message")
	require.Contains(t, output, "error message")
}

// TestLogger_SetLevel は SetLevel によるログレベル変更が反映されることを検証する
func TestLogger_SetLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.NewLogger(logger.WithWriter(&buf))
	log.SetLevel(slog.LevelError) // 実行時にログレベルを Error に変更

	log.Info("info should be hidden")   // 出力されないはず
	log.Error("error should show", nil) // 出力される

	out := buf.String()
	require.NotContains(t, out, "info should be hidden")
	require.Contains(t, out, "error should show")
}

// TestLogger_SetLevel_Twice は SetLevel を複数回呼び出した際に、ログレベルの変更が都度正しく反映されることを検証する
func TestLogger_SetLevel_Twice(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.NewLogger(logger.WithWriter(&buf))

	log.SetLevel(slog.LevelError)  // 1回目: Error に設定
	log.Debug("should not appear") // Debug ログ → 出力されないはず
	log.SetLevel(slog.LevelDebug)  // 2回目: Debug に設定
	log.Debug("should appear")     // Debug ログ → 出力されるはず

	out := buf.String()
	require.NotContains(t, out, "should not appear")
	require.Contains(t, out, "should appear")
}

// TestLogger_ConcurrentSetLevel は複数の goroutine から同時に SetLevel を呼び出してもクラッシュしないことを確認する
func TestLogger_ConcurrentSetLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.NewLogger(logger.WithWriter(&buf))

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(lvl slog.Level) {
			defer wg.Done()
			log.SetLevel(lvl) // 並行で SetLevel を呼び出す
		}(slog.LevelDebug)
	}
	wg.Wait()
}
