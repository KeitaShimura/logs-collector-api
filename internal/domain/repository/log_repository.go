package repository

import (
	"context"

	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
)

// LogRepository はログデータの永続化を担当
type LogRepository interface {
	SendLog(ctx context.Context, log *model.Log) error
	GetLogs(ctx context.Context, service string, level string, limit int, offset int) ([]model.Log, error)
}
