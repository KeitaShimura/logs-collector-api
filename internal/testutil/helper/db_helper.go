package testhelper

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // テスト用のインメモリDBで使用する SQLite ドライバ

	"github.com/KeitaShimura/logs-collector-api/internal/infra/db/models"
)

// newTestBoilExecutor は、SQLite のインメモリ DB をセットアップし、テーブルスキーマを作成したうえで、
// SQLBoiler 用の ContextExecutor と *sql.DB を返します。
func newTestBoilExecutor() (*sql.DB, *sql.DB, error) {
	// インメモリ DB に接続
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open in-memory DB: %w", err)
	}

	// テーブルスキーマ定義
	schema := `
		CREATE TABLE logs (
		id        TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		trace_id  TEXT,
		timestamp DATETIME NOT NULL,
		level     TEXT NOT NULL CHECK (length(level) <= 10),
		service   TEXT NOT NULL,
		message   TEXT NOT NULL,
		metadata  JSON
		);
	`

	// スキーマを実行してテーブルを作成
	if _, err := dbConn.Exec(schema); err != nil {
		return nil, nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// boil.ContextExecutor として dbConn を返却
	return dbConn, dbConn, nil
}

// InsertTestLog は、テスト用のログレコードを 1 件生成し、
// 指定された ContextExecutor で DB に挿入します。
func InsertTestLog(ctx context.Context, exec boil.ContextExecutor) error {
	//nolint:exhaustruct // LogL は型が生成されていないため初期化できない
	log := &models.Log{
		ID:        uuid.NewString(),                        // 一意のIDを生成
		TraceID:   null.StringFrom("trace-123"),            // nullable 型
		Timestamp: time.Now(),                              // 現在時刻を設定
		Service:   "test-service",                          // サービス名
		Level:     "INFO",                                  // ログレベル
		Message:   "for test",                              // メッセージ内容
		Metadata:  null.JSONFrom([]byte(`{"env":"test"}`)), // JSON メタデータ
		R:         nil,                                     // 関連リレーションは省略
		// L: models.LogL{}, ← 非公開型のため初期化を省略
	}

	// レコードを挿入
	if err := log.Insert(ctx, exec, boil.Infer()); err != nil {
		return fmt.Errorf("failed to insert test log: %w", err)
	}

	return nil
}
