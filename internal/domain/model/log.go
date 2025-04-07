package model

import (
	"time"
)

// Log はログエントリを表すドメインモデル
type Log struct {
	ID        string            // UUID
	TraceID   string            // 分散トレーシング用の一意なID
	Timestamp time.Time         // ログ発生時刻
	Level     string            // ログレベル (INFO, WARN, ERROR)
	Service   string            // 送信元サービス
	Message   string            // ログメッセージ
	Metadata  map[string]string // 追加情報（IP, リクエスト ID など）
}
