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
	DBHost     string `env:"DB_HOST"     required:"true"`
	DBPort     string `env:"DB_PORT"     required:"true"`
	DBName     string `env:"DB_NAME"     required:"true"`
	DBUser     string `env:"DB_USER"     required:"true"`
	DBPassword string `env:"DB_PASS"     required:"true"`
	DBSSLMode  string `env:"DB_SSLMODE"  envDefault:"disable"`
	DBTimeZone string `env:"DB_TIMEZONE" envDefault:"Asia/Tokyo"`
}

// NewConfig は環境変数から設定値を読み込み、Config 構造体を返す
func NewConfig(log logger.Logger) (*Config, error) {
	var cfg Config

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

	log.Info("Configuration loaded successfully",
		"DBHost", cfg.DBHost,
		"DBPort", cfg.DBPort,
		"DBName", cfg.DBName,
		"DBSSLMode", cfg.DBSSLMode,
		"DBTimeZone", cfg.DBTimeZone,
	)

	return &cfg, nil
}
