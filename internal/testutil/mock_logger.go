package testutil

import (
	"log/slog"
)

// LogEntry はモックロガーで記録されたログ情報
type LogEntry struct {
	Msg  string
	Args []any
}

// MockLogger は Logger を模倣するモック構造体
type MockLogger struct {
	Infos  []LogEntry
	Warns  []LogEntry
	Errors []LogEntry
}

// SetLevel は動的にログレベルを変更する（実際には使用しない）
func (m *MockLogger) SetLevel(_ slog.Level) {}

// Debug はデバッグ用の詳細ログを出力する
func (m *MockLogger) Debug(msg string, args ...any) {
	m.Infos = append(m.Infos, LogEntry{Msg: msg, Args: args})
}

// Info は情報ログを出力する
func (m *MockLogger) Info(msg string, args ...any) {
	m.Infos = append(m.Infos, LogEntry{Msg: msg, Args: args})
}

// Warn は警告ログを出力する
func (m *MockLogger) Warn(msg string, args ...any) {
	m.Warns = append(m.Warns, LogEntry{Msg: msg, Args: args})
}

// Error はエラーログを出力する（nil エラーも考慮）
func (m *MockLogger) Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, slog.String("error", err.Error()))
	}

	m.Errors = append(m.Errors, LogEntry{Msg: msg, Args: args})
}

// NewMockLogger はモックロガーを初期化して返す
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Infos:  []LogEntry{},
		Warns:  []LogEntry{},
		Errors: []LogEntry{},
	}
}
