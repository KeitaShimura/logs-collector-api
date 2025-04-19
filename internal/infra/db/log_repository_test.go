package db_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v8"

	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/repository"
	"github.com/KeitaShimura/logs-collector-api/internal/infra/db"
	"github.com/KeitaShimura/logs-collector-api/internal/infra/db/models"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

var (
	errInsert = errors.New("insert error")
	errQuery  = errors.New("db failure")
)

// setupDBTestWithLogger はモックDB・ロガー付きリポジトリ・クリーンアップ関数を返すテスト用のヘルパー関数
//
//nolint:ireturn // クリーンアーキテクチャのため命名を維持
func setupDBTestWithLogger(t *testing.T) (repository.LogRepository, sqlmock.Sqlmock, *testutil.MockLogger, func()) {
	t.Helper()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)

	cleanup := func() { dbConn.Close() }

	mockLogger := testutil.NewMockLogger()
	repo := db.NewLogRepository(dbConn, mockLogger)

	return repo, mock, mockLogger, cleanup
}

// TestLogRepository_SendLog_LogsInfo はログ保存成功時に Info ログが出力されることを検証するテスト
func TestLogRepository_SendLog_LogsInfo(t *testing.T) {
	t.Parallel()

	// モックリポジトリ・ロガー・DBを初期化
	repo, mock, logger, cleanup := setupDBTestWithLogger(t)
	defer cleanup()

	ctx := t.Context()

	// テスト用のログデータを準備
	logEntry := &model.Log{
		ID:        "log-123",
		TraceID:   "trace-abc",
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "auth-service",
		Message:   "test log",
		Metadata:  map[string]string{"ip": "127.0.0.1"},
	}

	// メタデータを JSON に変換
	metadataJSON, err := json.Marshal(logEntry.Metadata)
	require.NoError(t, err)

	// INSERT クエリの期待値を設定
	mock.ExpectExec(`INSERT INTO "logs"`).
		WithArgs(
			logEntry.ID,
			sqlmock.AnyArg(),
			logEntry.Timestamp,
			logEntry.Level,
			logEntry.Service,
			logEntry.Message,
			metadataJSON,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// ログ保存処理の実行
	err = repo.SendLog(ctx, logEntry)
	require.NoError(t, err)

	// Info ログが1件出力されていることを確認
	require.Len(t, logger.Infos, 1)
	require.Contains(t, logger.Infos[0].Msg, "Log successfully inserted")

	// モックの期待値がすべて満たされていることを確認
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestLogRepository_SendLog_WithEmptyMetadata はメタデータが空でも正常にログが保存されることを確認するテスト
func TestLogRepository_SendLog_WithEmptyMetadata(t *testing.T) {
	t.Parallel()

	// モックDB・ロガー・リポジトリを初期化
	repo, mock, logger, cleanup := setupDBTestWithLogger(t)
	defer cleanup()

	ctx := t.Context()

	// メタデータが空のログデータを準備
	logEntry := &model.Log{
		ID:        "log-empty-meta",
		TraceID:   "trace-empty",
		Timestamp: time.Now(),
		Level:     "DEBUG",
		Service:   "billing-service",
		Message:   "no metadata provided",
		Metadata:  map[string]string{}, // 空のメタデータ
	}

	// メタデータを JSON に変換
	metadataJSON, err := json.Marshal(logEntry.Metadata)
	require.NoError(t, err)

	// INSERT クエリの期待動作をモックで定義
	mock.ExpectExec(`INSERT INTO "logs"`).
		WithArgs(
			logEntry.ID,
			sqlmock.AnyArg(),
			logEntry.Timestamp,
			logEntry.Level,
			logEntry.Service,
			logEntry.Message,
			metadataJSON,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// ログ保存処理の実行
	err = repo.SendLog(ctx, logEntry)
	require.NoError(t, err)

	// Info ログが1件出力されていることを確認
	require.Len(t, logger.Infos, 1)
	require.Contains(t, logger.Infos[0].Msg, "Log successfully inserted")

	// モックの期待値がすべて満たされていることを確認
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestLogRepository_SendLog_InsertError はINSERT に失敗した場合にエラーが返ることを確認するテスト
func TestLogRepository_SendLog_InsertError(t *testing.T) {
	t.Parallel()

	// モックリポジトリと DB 接続を初期化
	repo, mock, _, cleanup := setupDBTestWithLogger(t)
	defer cleanup()

	ctx := t.Context()

	// テスト用のログデータを準備（空のメタデータ）
	logEntry := &model.Log{
		ID:        "log-insert-fail",
		TraceID:   "trace-xyz",
		Timestamp: time.Now(),
		Level:     "WARN",
		Service:   "user-service",
		Message:   "insert should fail",
		Metadata:  map[string]string{},
	}

	// メタデータを JSON に変換
	metadataJSON, err := json.Marshal(logEntry.Metadata)
	require.NoError(t, err)

	// INSERT クエリがエラーを返すことをモックで指定
	mock.ExpectExec(`INSERT INTO "logs"`).
		WithArgs(
			logEntry.ID,
			sqlmock.AnyArg(),
			logEntry.Timestamp,
			logEntry.Level,
			logEntry.Service,
			logEntry.Message,
			metadataJSON,
		).
		WillReturnError(fmt.Errorf("%w", errInsert))

	// SendLog 実行時にエラーが返ることを確認
	err = repo.SendLog(ctx, logEntry)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insert error")

	// モックの期待がすべて満たされていることを確認
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestLogRepository_GetLogs_Success はログの取得に成功した場合の結果と Info ログの出力を確認するテスト
func TestLogRepository_GetLogs_Success(t *testing.T) {
	t.Parallel()

	// モックDB・ロガー・リポジトリの初期化
	repo, mock, logger, cleanup := setupDBTestWithLogger(t)
	defer cleanup()

	ctx := t.Context()

	// メタデータ付きのログレコードを準備
	metadata := map[string]string{"key": "value"}
	metadataJSON, err := json.Marshal(metadata)
	require.NoError(t, err)

	// モックの DB レスポンス（1件のログを返す）
	rows := sqlmock.NewRows([]string{
		"id", "trace_id", "timestamp", "level", "service", "message", "metadata",
	}).AddRow("log-1", "trace-1", time.Now(), "INFO", "user-service", "test message", metadataJSON)

	// SELECT クエリの期待値を設定
	mock.ExpectQuery(`SELECT (.+) FROM "logs"`).WillReturnRows(rows)

	// GetLogs を実行して正常にログが取得できることを検証
	results, err := repo.GetLogs(ctx, "user-service", "INFO", 10, 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "log-1", results[0].ID)
	require.Equal(t, "value", results[0].Metadata["key"])

	// Info ログが出力されていることを検証
	require.Len(t, logger.Infos, 1)
	require.Contains(t, logger.Infos[0].Msg, "Logs retrieved from DB successfully")

	// モックの期待がすべて満たされていることを確認
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestLogRepository_GetLogs_NoMetadata はメタデータが nil のログが取得された場合でも正しく処理されることを確認するテスト
func TestLogRepository_GetLogs_NoMetadata(t *testing.T) {
	t.Parallel()

	// モックDB・ロガー・リポジトリの初期化
	repo, mock, logger, cleanup := setupDBTestWithLogger(t)
	defer cleanup()

	ctx := t.Context()

	// メタデータが nil のレコードをモックで返すように設定
	rows := sqlmock.NewRows([]string{
		"id", "trace_id", "timestamp", "level", "service", "message", "metadata",
	}).AddRow("log-2", "trace-2", time.Now(), "INFO", "user-service", "message with no metadata", nil)

	// SELECT クエリの期待値を設定
	mock.ExpectQuery(`SELECT (.+) FROM "logs"`).WillReturnRows(rows)

	// GetLogs を実行し、メタデータが空の map として扱われることを検証
	results, err := repo.GetLogs(ctx, "user-service", "INFO", 10, 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "log-2", results[0].ID)
	require.Empty(t, results[0].Metadata) // メタデータは空 map として扱われる

	// Info ログが出力されていることを確認
	require.Len(t, logger.Infos, 1)
	require.Contains(t, logger.Infos[0].Msg, "Logs retrieved from DB successfully")

	// モックの期待がすべて満たされていることを確認
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestLogRepository_GetLogs_UnmarshalError はメタデータの JSON が不正な場合にエラーが返ることを確認するテスト
func TestLogRepository_GetLogs_UnmarshalError(t *testing.T) {
	t.Parallel()

	// モックDB・ロガー・リポジトリの初期化
	repo, mock, _, cleanup := setupDBTestWithLogger(t)
	defer cleanup()

	ctx := t.Context()

	// 不正な JSON（末尾が欠落している）をメタデータに設定
	badJSON := []byte(`{"key":`) // 不正なJSON

	// 不正なメタデータを含むレコードを返すようにモックを設定
	rows := sqlmock.NewRows([]string{
		"id", "trace_id", "timestamp", "level", "service", "message", "metadata",
	}).AddRow("log-3", "trace-3", time.Now(), "ERROR", "user-service", "bad json", badJSON)

	// SELECT クエリの期待値を設定
	mock.ExpectQuery(`SELECT (.+) FROM "logs"`).WillReturnRows(rows)

	// GetLogs 実行時に JSON パースエラーが発生し、エラーが返ることを確認
	results, err := repo.GetLogs(ctx, "user-service", "ERROR", 10, 0)
	require.Error(t, err)
	require.Nil(t, results)

	// モックの期待がすべて満たされていることを確認
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestLogRepository_GetLogs_QueryError はDB クエリが失敗した場合にエラーが返ることを確認するテスト
func TestLogRepository_GetLogs_QueryError(t *testing.T) {
	t.Parallel()

	// モックDB・リポジトリを初期化（ロガーは使わないので省略）
	repo, mock, _, cleanup := setupDBTestWithLogger(t)
	defer cleanup()

	ctx := t.Context()

	// クエリ実行時にエラーを返すように設定
	mock.ExpectQuery(`SELECT (.+) FROM "logs"`).
		WillReturnError(fmt.Errorf("%w", errQuery))

	// GetLogs を実行し、クエリエラーが返ることを確認
	results, err := repo.GetLogs(ctx, "user-service", "ERROR", 10, 0)
	require.Error(t, err)
	require.Nil(t, results)

	// モックの期待がすべて満たされていることを確認
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestExtractMetadata_Valid は正しい JSON のメタデータが map に変換されることを確認するテスト
func TestExtractMetadata_Valid(t *testing.T) {
	t.Parallel()

	// 検証用のメタデータ map を定義し、JSON に変換
	meta := map[string]string{"env": "prod"}
	metaJSON, err := json.Marshal(meta)
	require.NoError(t, err)

	// 正常な Metadata を持つ Log モデルを準備
	//nolint:exhaustruct // LogL は型が生成されていないため初期化できない
	logEntry := &models.Log{
		ID:        "log-123",
		TraceID:   null.StringFrom("trace-123"),
		Timestamp: time.Now(),
		Level:     "INFO",
		Service:   "auth-service",
		Message:   "login success",
		Metadata: null.JSON{
			JSON:  metaJSON,
			Valid: true,
		},
		R: nil,
		// L: models.LogL{}, ← 外部パッケージからは参照できないので省略
	}

	// ExtractMetadata を実行し、map に正しく変換されることを確認
	result, err := db.ExtractMetadata(logEntry)
	require.NoError(t, err)
	require.Equal(t, meta, result)
}

// TestExtractMetadata_NotValid はMetadata.Valid が false の場合に空の map が返ることを確認するテスト
func TestExtractMetadata_NotValid(t *testing.T) {
	t.Parallel()

	// Metadata.Valid が false の状態で Log モデルを準備（JSON は無視される）
	//nolint:exhaustruct // LogL は型が生成されていないため初期化できない
	logEntry := &models.Log{
		ID:        "log-789",
		TraceID:   null.StringFrom("trace-789"),
		Timestamp: time.Now(),
		Level:     "DEBUG",
		Service:   "notification-service",
		Message:   "no metadata",
		Metadata: null.JSON{
			JSON:  nil,
			Valid: false,
		},
		R: nil,
		// L: models.LogL{}, ← 外部パッケージからは参照できないので省略
	}

	// メタデータは空の map として返ることを確認
	result, err := db.ExtractMetadata(logEntry)
	require.NoError(t, err)
	require.Empty(t, result)
}

// TestExtractMetadata_InvalidJSON は不正な JSON を含むメタデータがエラーになることを確認するテスト
func TestExtractMetadata_InvalidJSON(t *testing.T) {
	t.Parallel()

	// JSON の形式が崩れている（末尾の値がない）不正なメタデータを準備
	badJSON := []byte(`{"env":`) // 不完全なJSON

	//nolint:exhaustruct // LogL は型が生成されていないため初期化できない
	logEntry := &models.Log{
		ID:        "log-456",
		TraceID:   null.StringFrom("trace-456"),
		Timestamp: time.Now(),
		Level:     "ERROR",
		Service:   "billing-service",
		Message:   "invalid json",
		Metadata: null.JSON{
			JSON:  badJSON,
			Valid: true,
		},
		R: nil,
		// L: models.LogL{}, ← 外部パッケージからは参照できないので省略
	}

	// メタデータ変換時にエラーが発生し、nil が返ることを確認
	result, err := db.ExtractMetadata(logEntry)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "logID=log-456")
}
