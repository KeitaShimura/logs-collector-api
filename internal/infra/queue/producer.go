package queue

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// NATSConn は NATS 接続インターフェース（テストのために抽象化）
type NATSConn interface {
	Publish(subject string, data []byte) error
}

// ProducerInterface はNATS送信用のインターフェース
type LogProducer interface {
	Publish(subject string, msg LogMessage) error
}

// Producer はログを NATS に送信する構造体
type Producer struct {
	Conn   NATSConn
	Logger logger.Logger
}

// NewProducer は NATS に接続して Producer を作成
func NewProducer(url string, logger logger.Logger) (*Producer, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS (url: %s): %w", url, err)
	}

	logger.Info("Connected to NATS successfully", "url", url)

	return &Producer{Conn: conn, Logger: logger}, nil
}

// Publish は subject にログメッセージを送信
func (p *Producer) Publish(subject string, msg LogMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message (subject: %s, msgID: %s): %w", subject, msg.ID, err)
	}

	if err := p.Conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish message to NATS (subject: %s, msgID: %s): %w", subject, msg.ID, err)
	}

	p.Logger.Info("Message published successfully", "subject", subject, "msgID", msg.ID)

	return nil
}
