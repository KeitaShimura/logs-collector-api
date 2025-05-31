package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/repository"
	"github.com/KeitaShimura/logs-collector-api/internal/infra/queue"
	"github.com/KeitaShimura/logs-collector-api/internal/infra/search"
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
	logRepo  repository.LogRepository
	producer queue.LogProducer
	searcher search.LogIndexer
	logger   logger.Logger
}

// NewLogUseCase は LogUseCase のインスタンスを生成する
func NewLogUseCase(
	repo repository.LogRepository,
	producer queue.LogProducer,
	searcher search.LogIndexer,
	logger logger.Logger,
) *LogUseCaseImpl {
	return &LogUseCaseImpl{
		logRepo:  repo,
		producer: producer,
		searcher: searcher,
		logger:   logger,
	}
}

// SendLog はログを永続化するユースケースを実行する
func (uc *LogUseCaseImpl) SendLog(ctx context.Context, logEntry *model.Log) error {
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

	// NATSへ送信
	msg := queue.LogMessage{
		ID:        logEntry.ID,
		TraceID:   logEntry.TraceID,
		Timestamp: logEntry.Timestamp.Format(time.RFC3339),
		Level:     logEntry.Level,
		Service:   logEntry.Service,
		Message:   logEntry.Message,
		Metadata:  logEntry.Metadata,
	}

	if err := uc.producer.Publish("logs.send", msg); err != nil {
		uc.logger.Error("Failed to publish log to NATS", err, "ID", msg.ID)
	}

	// Elasticsearchへインデックス
	esDoc := map[string]interface{}{
		"id":        logEntry.ID,
		"trace_id":  logEntry.TraceID,
		"timestamp": logEntry.Timestamp.Format(time.RFC3339),
		"level":     logEntry.Level,
		"service":   logEntry.Service,
		"message":   logEntry.Message,
		"metadata":  logEntry.Metadata,
	}

	if err := uc.searcher.IndexLog("logs-index", esDoc); err != nil {
		uc.logger.Error("Failed to index log to Elasticsearch", err, "ID", logEntry.ID)
	}

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
