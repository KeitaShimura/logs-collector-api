package config

import (
	"errors"
	"fmt"

	"github.com/caarlos0/env/v11"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// 共通エラー定義
var (
	errDBHostRequired     = errors.New("DB_HOST is required but not set")
	errDBPortRequired     = errors.New("DB_PORT is required but not set")
	errDBNameRequired     = errors.New("DB_NAME is required but not set")
	errDBUserRequired     = errors.New("DB_USER is required but not set")
	errDBPasswordRequired = errors.New("DB_PASS is required but not set")
)

// Config は環境変数から取得する設定情報を保持する構造体
type Config struct {
	// --- Database 設定 ---
	DBHost     string `env:"DB_HOST"     required:"true"`
	DBPort     string `env:"DB_PORT"     required:"true"`
	DBName     string `env:"DB_NAME"     required:"true"`
	DBUser     string `env:"DB_USER"     required:"true"`
	DBPassword string `env:"DB_PASS"     required:"true"`
	DBSSLMode  string `env:"DB_SSLMODE"  envDefault:"disable"`
	DBTimeZone string `env:"DB_TIMEZONE" envDefault:"Asia/Tokyo"`

	// --- NATS 設定 ---
	NATSURL string `env:"NATS_URL" envDefault:"nats://nats:4222"`

	// --- Elasticsearch 設定 ---
	ElasticsearchURL string `env:"ELASTICSEARCH_URL" envDefault:"http://elasticsearch:9200"`

	// --- サーバーポート/バインド設定 ---
	GRPCPort     string `env:"GRPC_PORT"      envDefault:"50051"`
	GRPCBindAddr string `env:"GRPC_BIND_ADDR" envDefault:"0.0.0.0"`
	RESTPort     string `env:"REST_PORT"      envDefault:"8080"`
	RESTBindAddr string `env:"REST_BIND_ADDR" envDefault:"0.0.0.0"`

	// --- タイムアウト/ログ設定 ---
	GRPCTimeoutSec     int    `env:"GRPC_TIMEOUT_SEC"     envDefault:"2"`
	RESTTimeoutSec     int    `env:"REST_TIMEOUT_SEC"     envDefault:"3"`
	ShutdownTimeoutSec int    `env:"SHUTDOWN_TIMEOUT_SEC" envDefault:"10"`
	LogLevel           string `env:"LOG_LEVEL"            envDefault:"INFO"`
}

// NewConfig は環境変数から設定値を読み込み、Config 構造体を返す
func NewConfig(log logger.Logger) (*Config, error) {
	var cfg Config

	// 環境変数から構造体にマッピング
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env config: %w", err)
	}

	// --- 必須項目チェック ---
	if cfg.DBHost == "" {
		return nil, fmt.Errorf("validation error: %w", errDBHostRequired)
	}

	if cfg.DBPort == "" {
		return nil, fmt.Errorf("validation error: %w", errDBPortRequired)
	}

	if cfg.DBName == "" {
		return nil, fmt.Errorf("validation error: %w", errDBNameRequired)
	}

	if cfg.DBUser == "" {
		return nil, fmt.Errorf("validation error: %w", errDBUserRequired)
	}

	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("validation error: %w", errDBPasswordRequired)
	}

	// ログ出力（パスワードは出さない）
	log.Info("Configuration loaded successfully",
		"DBHost", cfg.DBHost,
		"DBPort", cfg.DBPort,
		"DBName", cfg.DBName,
		"DBUser", cfg.DBUser,
		"DBSSLMode", cfg.DBSSLMode,
		"DBTimeZone", cfg.DBTimeZone,
		"NATSURL", cfg.NATSURL,
		"ElasticsearchURL", cfg.ElasticsearchURL,
		"GRPCPort", cfg.GRPCPort,
		"GRPCBindAddr", cfg.GRPCBindAddr,
		"RESTPort", cfg.RESTPort,
		"RESTBindAddr", cfg.RESTBindAddr,
		"GRPCTimeout", cfg.GRPCTimeoutSec,
		"RESTTimeoutSec", cfg.RESTTimeoutSec,
		"ShutdownTimeout", cfg.ShutdownTimeoutSec,
		"LogLevel", cfg.LogLevel,
	)

	return &cfg, nil
}
