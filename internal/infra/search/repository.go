package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// 共通エラー定義
var ErrIndexingFailed = errors.New("indexing to Elasticsearch failed")

// ESClient は Elasticsearch クライアントのインターフェース（テスト用抽象化）
type ESClient interface {
	Index(index string, body *bytes.Reader, o ...func(*esapi.IndexRequest)) (*esapi.Response, error)
}

// LogSearcher は Elasticsearch へのログ登録用構造体
type LogSearcher struct {
	client ESClient
	log    logger.Logger
}

// NewLogSearcher は LogSearcher を作成する
func NewLogSearcher(client ESClient, log logger.Logger) *LogSearcher {
	return &LogSearcher{client: client, log: log}
}

// IndexLog は指定された index に logData を登録する
func (ls *LogSearcher) IndexLog(index string, logData map[string]interface{}) error {
	// ログデータを JSON に変換
	data, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("failed to marshal log (index: %s): %w", index, err)
	}

	// Elasticsearch にデータを送信
	res, err := ls.client.Index(index, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send index request (index: %s): %w", index, err)
	}
	defer res.Body.Close()

	// エラーステータスの場合はエラー扱い
	if res.IsError() {
		return fmt.Errorf("%w (index: %s, status: %s)", ErrIndexingFailed, index, res.Status())
	}

	// 成功ログ
	ls.log.Info("Log indexed successfully", "index", index)

	return nil
}
