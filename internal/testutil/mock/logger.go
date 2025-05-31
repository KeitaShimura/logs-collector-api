package mock

import (
	"log/slog"
)

// LogEntry はモックロガーで記録されたログ情報
type LogEntry struct {
	Msg  string
	Err  error
	Args []any
}

// Logger は Logger を模倣するモック構造体
type Logger struct {
	Debugs []LogEntry
	Infos  []LogEntry
	Warns  []LogEntry
	Errors []LogEntry
}

// SetLevel は動的にログレベルを変更する（実際には使用しない）
func (m *Logger) SetLevel(_ slog.Level) {}

// Debug はデバッグ用の詳細ログを出力する
func (m *Logger) Debug(msg string, args ...any) {
	m.Debugs = append(m.Debugs, LogEntry{
		Msg:  msg,
		Err:  nil,
		Args: args,
	})
}

// Info は情報ログを出力する
func (m *Logger) Info(msg string, args ...any) {
	m.Infos = append(m.Infos, LogEntry{
		Msg:  msg,
		Err:  nil,
		Args: args,
	})
}

// Warn は警告ログを出力する
func (m *Logger) Warn(msg string, args ...any) {
	m.Warns = append(m.Warns, LogEntry{
		Msg:  msg,
		Err:  nil,
		Args: args,
	})
}

// Error はエラーログを出力する（nil エラーも考慮）
func (m *Logger) Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, slog.String("error", err.Error()))
	}

	m.Errors = append(m.Errors, LogEntry{
		Msg:  msg,
		Err:  err,
		Args: args,
	})
}

// NewLogger はモックロガーを初期化して返す
func NewLogger() *Logger {
	return &Logger{
		Debugs: []LogEntry{},
		Infos:  []LogEntry{},
		Warns:  []LogEntry{},
		Errors: []LogEntry{},
	}
}
