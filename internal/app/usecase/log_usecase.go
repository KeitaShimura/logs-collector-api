package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/repository"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// 共通エラー定義
var (
	ErrLogEntryNil          = errors.New("log entry is nil")
	ErrEmptyMessage         = errors.New("message must not be empty")
	ErrEmptyLevel           = errors.New("level must not be empty")
	ErrEmptyService         = errors.New("service must not be empty")
	ErrInvalidIDFormat      = errors.New("invalid ID format")
	ErrInvalidTraceIDFormat = errors.New("invalid TraceID format")
	ErrInvalidTimeZone      = errors.New("invalid time zone")
	ErrInvalidLimit         = errors.New("limit must be greater than 0")
	ErrInvalidOffset        = errors.New("offset must be >= 0")
	ErrNoLogsFound          = errors.New("no logs found")
	ErrRepositoryFailure    = errors.New("repository failure")
	ErrValidationFailure    = errors.New("validation failure")
)

// LogUseCase はログに関連するユースケースのインターフェースを定義する
type LogUseCase interface {
	SendLog(ctx context.Context, log *model.Log) error
	GetLogs(ctx context.Context, service string, level string, limit, offset int) ([]model.Log, error)
}

// LogUseCaseImpl は LogUseCase インターフェースの具体的な実装
type LogUseCaseImpl struct {
	logRepo repository.LogRepository
	logger  logger.Logger
}

// NewLogUseCase は LogUseCase のインスタンスを生成する
func NewLogUseCase(repo repository.LogRepository, log logger.Logger) *LogUseCaseImpl {
	return &LogUseCaseImpl{
		logRepo: repo,
		logger:  log,
	}
}

// SendLog はログを永続化するユースケースを実行する
func (uc *LogUseCaseImpl) SendLog(ctx context.Context, logEntry *model.Log) error {
	// 入力バリデーション（ID, TraceID, Message など）
	if err := ValidateLog(logEntry, time.LoadLocation); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailure, err)
	}

	// 永続化
	if err := uc.logRepo.SendLog(ctx, logEntry); err != nil {
		uc.logger.Error("Failed to save log entry", err,
			"ID", logEntry.ID,
			"TraceID", logEntry.TraceID,
			"Timestamp", logEntry.Timestamp,
			"Service", logEntry.Service,
			"Level", logEntry.Level,
			"Message", logEntry.Message,
			"Metadata", logEntry.Metadata,
		)

		return fmt.Errorf("%w: %w", ErrRepositoryFailure, err)
	}

	// 保存成功ログ
	uc.logger.Info("Log entry saved successfully",
		"ID", logEntry.ID,
		"TraceID", logEntry.TraceID,
		"Timestamp", logEntry.Timestamp,
		"Service", logEntry.Service,
		"Level", logEntry.Level,
		"Message", logEntry.Message,
		"Metadata", logEntry.Metadata,
	)

	return nil
}

// GetLogs は指定された条件に一致するログを取得する
func (uc *LogUseCaseImpl) GetLogs(
	ctx context.Context,
	service string,
	level string,
	limit int,
	offset int,
) ([]model.Log, error) {
	// 入力パラメータ検証
	if err := ValidateLogQueryParams(service, level, limit, offset); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailure, err)
	}

	// 永続層から取得
	logs, err := uc.logRepo.GetLogs(ctx, service, level, limit, offset)
	if err != nil {
		uc.logger.Error("Failed to get logs", err,
			"Service", service,
			"Level", level,
			"Limit", limit,
			"Offset", offset,
		)

		return nil, fmt.Errorf("%w: %w", ErrRepositoryFailure, err)
	}

	// 取得結果が0件
	if len(logs) == 0 {
		uc.logger.Warn("No logs found",
			"Service", service,
			"Level", level,
			"Limit", limit,
			"Offset", offset,
		)

		return nil, fmt.Errorf("%w", ErrNoLogsFound)
	}

	// 成功ログ
	uc.logger.Info("Logs retrieved successfully",
		"RetrievedCount", len(logs),
		"Service", service,
		"Level", level,
	)

	return logs, nil
}

// ValidateLog はログエントリの入力検証を行う
func ValidateLog(log *model.Log, timeLoadLocation func(string) (*time.Location, error)) error {
	if log == nil {
		return ErrLogEntryNil
	}

	if log.Message == "" {
		return ErrEmptyMessage
	}

	if log.Level == "" {
		return ErrEmptyLevel
	}

	if log.Service == "" {
		return ErrEmptyService
	}

	if _, err := uuid.Parse(log.ID); err != nil {
		return ErrInvalidIDFormat
	}

	if _, err := uuid.Parse(log.TraceID); err != nil {
		return ErrInvalidTraceIDFormat
	}

	// Timestamp がゼロ値の場合 JST で補完
	if log.Timestamp.IsZero() {
		jst, err := timeLoadLocation("Asia/Tokyo")
		if err != nil {
			return ErrInvalidTimeZone
		}

		log.Timestamp = time.Now().In(jst) // 日本時間（Asia/Tokyo）で補完
	}

	return nil
}

// ValidateLogQueryParams はログ検索条件の検証を行う
func ValidateLogQueryParams(service string, level string, limit int, offset int) error {
	if service == "" {
		return ErrEmptyService
	}

	if level == "" {
		return ErrEmptyLevel
	}

	if limit <= 0 {
		return ErrInvalidLimit
	}

	if offset < 0 {
		return ErrInvalidOffset
	}

	return nil
}
