// internal/testutil/mock/log_searcher.go

package mock

import "github.com/KeitaShimura/logs-collector-api/internal/infra/search"

type LogSearcher struct {
	IndexLogFunc func(index string, logData map[string]interface{}) error
	Calls        []IndexCall
}

type IndexCall struct {
	Index   string
	LogData map[string]interface{}
}

// コンストラクタ
func NewLogSearcher() *LogSearcher {
	return &LogSearcher{
		IndexLogFunc: nil,
		Calls:        nil,
	}
}

func (m *LogSearcher) IndexLog(index string, logData map[string]interface{}) error {
	m.Calls = append(m.Calls, IndexCall{
		Index:   index,
		LogData: logData,
	})

	return nil
}

var _ search.LogIndexer = (*LogSearcher)(nil) // interface compliance check
