package logger

import (
	"io"
	"log/slog"
	"os"
	"sync"
)

// 公開用ログレベル定義
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// LoggerInterface は Logger が満たすべきインターフェース
type Logger interface {
	SetLevel(level slog.Level)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, err error, args ...any)
}

// LoggerImpl は構造化ロギング用のカスタムロガー実装
//
//nolint:revive // クリーンアーキテクチャ上の意図により命名を維持
type LoggerImpl struct {
	slog   *slog.Logger
	writer io.Writer
	level  slog.Level
	mutex  sync.Mutex
}

// Option は LoggerImpl のオプション設定用関数
type Option func(*LoggerImpl)

// WithWriter はログの出力先を設定する
func WithWriter(w io.Writer) Option {
	return func(logger *LoggerImpl) {
		logger.writer = w
		logger.rebuildLogger()
	}
}

// WithLevel はログレベルを設定する
func WithLevel(level slog.Level) Option {
	return func(logger *LoggerImpl) {
		logger.level = level
		logger.rebuildLogger()
	}
}

// NewLogger はカスタマイズ可能なロガーを作成し、Logger インターフェースとして返す
//
//nolint:ireturn // クリーンアーキテクチャのため命名を維持
func NewLogger(options ...Option) Logger {
	logger := &LoggerImpl{
		writer: os.Stdout,
		level:  slog.LevelInfo,
		slog:   nil,
		mutex:  sync.Mutex{},
	}

	for _, opt := range options {
		opt(logger)
	}

	logger.rebuildLogger()

	return logger
}

// rebuildLogger は LoggerImpl の設定変更後に再構築する
func (logger *LoggerImpl) rebuildLogger() {
	opts := &slog.HandlerOptions{
		Level:       logger.level,
		AddSource:   false,
		ReplaceAttr: nil,
	}
	handler := slog.NewJSONHandler(logger.writer, opts)
	logger.slog = slog.New(handler)
}

// SetLevel は動的にログレベルを変更する
func (logger *LoggerImpl) SetLevel(level slog.Level) {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	logger.level = level
	logger.rebuildLogger()
}

// Debug はデバッグ用の詳細ログを出力する
func (logger *LoggerImpl) Debug(msg string, args ...any) {
	logger.slog.Debug(msg, args...)
}

// Info は情報ログを出力する
func (logger *LoggerImpl) Info(msg string, args ...any) {
	logger.slog.Info(msg, args...)
}

// Warn は警告ログを出力する
func (logger *LoggerImpl) Warn(msg string, args ...any) {
	logger.slog.Warn(msg, args...)
}

// Error はエラーログを出力する（nil エラーも考慮）
func (logger *LoggerImpl) Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, "error", err.Error())
	}

	logger.slog.Error(msg, args...)
}
