package mock

import (
	"net/http"
)

// ESClient は Elasticsearch クライアントのモック
type ESClient struct {
	PerformFunc func(*http.Request) (*http.Response, error)
}

func (m *ESClient) Perform(req *http.Request) (*http.Response, error) {
	return m.PerformFunc(req)
}
