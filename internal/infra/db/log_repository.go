package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/repository"
	"github.com/KeitaShimura/logs-collector-api/internal/infra/db/models"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// LogRepository はログの永続化を担うリポジトリ実装
// domain層のLogRepositoryインターフェースを実装している
type LogRepository struct {
	db     boil.ContextExecutor
	logger logger.Logger
}

// NewLogRepository は新しいログリポジトリを作成するファクトリー関数
//
//nolint:ireturn // クリーンアーキテクチャ上の意図により命名を維持
func NewLogRepository(db boil.ContextExecutor, logger logger.Logger) repository.LogRepository {
	return &LogRepository{
		db:     db,
		logger: logger,
	}
}

// SendLog はログをデータベースに保存する
func (r *LogRepository) SendLog(ctx context.Context, log *model.Log) error {
	// ログのメタデータをJSONに変換
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf(
			"failed to marshal log metadata (logID=%s, service=%s, traceID=%s, level=%s): %w",
			log.ID, log.Service, log.TraceID, log.Level, err)
	}

	// domainモデルをDBモデルに変換
	//nolint:exhaustruct // LogL は型が生成されていないため初期化できない
	logDB := models.Log{
		ID:        log.ID,
		TraceID:   null.StringFrom(log.TraceID),
		Timestamp: log.Timestamp,
		Level:     log.Level,
		Service:   log.Service,
		Message:   log.Message,
		Metadata:  null.JSONFrom(metadataJSON),
		R:         nil,
	}

	// データベースに挿入処理を実行
	if err := logDB.Insert(ctx, r.db, boil.Infer()); err != nil {
		return fmt.Errorf(
			"failed to insert log into DB (logID=%s, service=%s, traceID=%s, level=%s): %w",
			log.ID, log.Service, log.TraceID, log.Level, err)
	}

	r.logger.Info("Log successfully inserted",
		"logID", log.ID,
		"service", log.Service,
		"traceID", log.TraceID,
		"level", log.Level,
	)

	return nil
}

// GetLogs は指定された条件に合致するログをデータベースから取得する
func (r *LogRepository) GetLogs(
	ctx context.Context,
	service string,
	level string,
	limit int,
	offset int,
) ([]model.Log, error) {
	const initialQueryCap = 3 // service, level, pagination を想定

	// クエリ条件を構築
	query := make([]qm.QueryMod, 0, initialQueryCap)

	if service != "" {
		query = append(query, qm.Where("service = ?", service))
	}

	if level != "" {
		query = append(query, qm.Where("level = ?", level))
	}

	query = append(query, qm.Limit(limit), qm.Offset(offset))

	// クエリ実行
	logs, err := models.Logs(query...).All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to execute log query (service=%s, level=%s, limit=%d, offset=%d): %w",
			service, level, limit, offset, err)
	}

	results := make([]model.Log, 0, len(logs))

	// DBモデルをドメインモデルに変換
	for _, logEntry := range logs {
		metadata, err := ExtractMetadata(logEntry)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to unmarshal metadata (logID=%s, service=%s, traceID=%s, level=%s): %w",
				logEntry.ID, logEntry.Service, logEntry.TraceID.String, logEntry.Level, err)
		}

		results = append(results, model.Log{
			ID:        logEntry.ID,
			TraceID:   logEntry.TraceID.String,
			Timestamp: logEntry.Timestamp,
			Level:     logEntry.Level,
			Service:   logEntry.Service,
			Message:   logEntry.Message,
			Metadata:  metadata,
		})
	}

	r.logger.Info("Logs retrieved from DB successfully",
		"retrievedCount", len(results),
		"service", service,
		"level", level,
		"limit", limit,
		"offset", offset,
	)

	return results, nil
}

// ExtractMetadata はLogエントリからメタデータを抽出してmapに変換する
func ExtractMetadata(logEntry *models.Log) (map[string]string, error) {
	if !logEntry.Metadata.Valid {
		return map[string]string{}, nil
	}

	var metadata map[string]string
	if err := json.Unmarshal(logEntry.Metadata.JSON, &metadata); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal metadata (logID=%s, service=%s, traceID=%s, level=%s): %w",
			logEntry.ID, logEntry.Service, logEntry.TraceID.String, logEntry.Level, err)
	}

	return metadata, nil
}
