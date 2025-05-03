package search

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// NewESClient は指定されたアドレスで Elasticsearch クライアントを作成する
func NewESClient(addresses []string, log logger.Logger) (*elasticsearch.Client, error) {
	//nolint:exhaustruct // 必要なフィールドだけ初期化する
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}

	// Elasticsearch クライアントを作成
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to initialize elasticsearch client (addresses: %v): %w",
			cfg.Addresses, err,
		)
	}

	log.Info("Elasticsearch client initialized successfully", "addresses", cfg.Addresses)

	return client, nil
}
