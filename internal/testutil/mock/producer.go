package mock

import (
	"github.com/KeitaShimura/logs-collector-api/internal/infra/queue"
)

// Producer は queue.ProducerInterface のモック
type Producer struct {
	PublishedMessages []queue.LogMessage
}

func NewProducer() *Producer {
	return &Producer{
		PublishedMessages: []queue.LogMessage{},
	}
}

func (m *Producer) Publish(_ string, msg queue.LogMessage) error {
	m.PublishedMessages = append(m.PublishedMessages, msg)

	return nil // 成功扱い
}
