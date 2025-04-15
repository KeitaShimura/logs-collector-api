package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/KeitaShimura/logs-collector-api/internal/config"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

func main() {
	log := setupLogger()

	// 環境変数から設定をロード
	cfg, err := config.NewConfig(log)
	if err != nil {
		log.Error("Failed to load configuration", err)
		os.Exit(1)
	}

	// PostgreSQLの接続文字列（DSN）を生成
	dsn := buildPostgresDSN(cfg)
	log.Info("Database migration DSN created", "dsn", dsn)

	// マイグレーションの実行
	if err := runMigration(dsn, cfg.DBName, log); err != nil {
		log.Error("Migration process failed", err)
		os.Exit(1)
	}

	log.Info("Database migration completed successfully")
}

// setupLogger は構造化ロガーを初期化する
//
//nolint:ireturn // クリーンアーキテクチャのため命名を維持
func setupLogger() logger.Logger {
	return logger.NewLogger(
		logger.WithLevel(logger.LevelInfo),
		logger.WithWriter(os.Stdout),
	)
}

// buildPostgresDSN は環境変数から PostgreSQL のDSNを構築する
func buildPostgresDSN(cfg *config.Config) string {
	return "postgres://" + cfg.DBUser + ":" + cfg.DBPassword +
		"@" + cfg.DBHost + ":" + cfg.DBPort +
		"/" + cfg.DBName + "?sslmode=" + cfg.DBSSLMode
}

// runMigration は指定されたDBに対してマイグレーションを実行する
func runMigration(dsn string, dbName string, log logger.Logger) error {
	// DBへの接続を確立
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open DB: %w", err)
	}
	defer dbConn.Close()

	log.Info("Database connection established for migration")

	// postgresドライバの作成
	driver, err := postgres.WithInstance(dbConn, &postgres.Config{
		MigrationsTable:       "schema_migrations",
		MigrationsTableQuoted: false,
		MultiStatementEnabled: true,
		DatabaseName:          dbName,
		SchemaName:            "",
		StatementTimeout:      0,
		MultiStatementMaxSize: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	log.Info("PostgreSQL migration driver initialized")

	// マイグレーターの作成
	migrator, err := migrate.NewWithDatabaseInstance(
		"file://internal/infra/db/migrations",
		"postgres", driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	log.Info("Migrator successfully created")

	// マイグレーションを実行
	if err := migrator.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("No migrations needed; schema up-to-date")

			return nil
		}

		return fmt.Errorf("migration failed: %w", err)
	}

	log.Info("Migration executed successfully")

	return nil
}
