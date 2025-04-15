package config_test

import (
	"os"
	"testing"

	"github.com/KeitaShimura/logs-collector-api/internal/config"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// TestNewConfig_SuccessWithDefaults はすべての必須環境変数が揃っており、デフォルト値が適用される場合のテスト
func TestNewConfig_SuccessWithDefaults(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "logs")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASS", "pass")
	// DB_SSLMODE / DB_TIMEZONE は未定義 → envDefaultが使われる

	log := testutil.NewMockLogger()

	cfg, err := config.NewConfig(log)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.DBSSLMode != "disable" {
		t.Errorf("expected DBSSLMode to be 'disable', got: %s", cfg.DBSSLMode)
	}

	if cfg.DBTimeZone != "Asia/Tokyo" {
		t.Errorf("expected DBTimeZone to be 'Asia/Tokyo', got: %s", cfg.DBTimeZone)
	}
}

// TestNewConfig_MissingRequiredEnv は必須環境変数（DB_HOST）が設定されていない場合のエラーテスト
func TestNewConfig_MissingRequiredEnv(t *testing.T) {
	os.Clearenv() // ✅ これで全環境変数をクリア

	// 必須の一部だけ設定（DB_HOST は設定しない）
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "logs")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASS", "pass")

	log := testutil.NewMockLogger()
	cfg, err := config.NewConfig(log)

	if err == nil {
		t.Fatal("expected error due to missing DB_HOST, got nil")
	}

	if cfg != nil {
		t.Errorf("expected nil config on error, got: %+v", cfg)
	}
}

// TestNewConfig_WithAllEnvDefined はすべての環境変数が明示的に設定されている場合のテスト
func TestNewConfig_WithAllEnvDefined(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "logs")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASS", "pass")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("DB_TIMEZONE", "UTC")

	log := testutil.NewMockLogger()

	cfg, err := config.NewConfig(log)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.DBSSLMode != "require" {
		t.Errorf("expected DBSSLMode to be 'require', got: %s", cfg.DBSSLMode)
	}

	if cfg.DBTimeZone != "UTC" {
		t.Errorf("expected DBTimeZone to be 'UTC', got: %s", cfg.DBTimeZone)
	}
}
