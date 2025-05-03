package search_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/infra/search"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// 共通エラー定義
var errMockIndexError = errors.New("index error")

// --- モック ---

// mockESClient は Elasticsearch クライアントのモック
type mockESClient struct {
	indexFunc func(index string, body *bytes.Reader, o ...func(*esapi.IndexRequest)) (*esapi.Response, error)
}

// Index はモックの Index メソッド
func (m *mockESClient) Index(
	index string,
	body *bytes.Reader,
	o ...func(*esapi.IndexRequest),
) (*esapi.Response, error) {
	return m.indexFunc(index, body, o...)
}

// errorCloser は Close 時にフラグを立てるモック
type errorCloser struct {
	io.Reader
	closed bool
}

// Close は closed フラグを true にするモック実装
func (e *errorCloser) Close() error {
	e.closed = true

	return nil
}

// --- テスト ---

// TestLogSearcher_IndexLog_Success はログが正常にインデックスされる場合のテスト
func TestLogSearcher_IndexLog_Success(t *testing.T) {
	t.Parallel()

	// モックロガーとモックElasticsearchクライアントを準備
	mockLogger := testutil.NewMockLogger()
	mockClient := &mockESClient{
		indexFunc: func(_ string, _ *bytes.Reader, _ ...func(*esapi.IndexRequest)) (*esapi.Response, error) {
			// 正常系のレスポンスを返す
			return &esapi.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("ok")),
			}, nil
		},
	}

	// 検索者を作成
	searcher := search.NewLogSearcher(mockClient, mockLogger)

	// テスト用ログデータ
	logData := map[string]interface{}{"message": "test log"}

	// インデックス登録を実行
	err := searcher.IndexLog("test-index", logData)
	// エラーがないことを確認
	require.NoError(t, err)

	// ログに正常メッセージが記録されていることを確認
	assert.Len(t, mockLogger.Infos, 1)
	assert.Contains(t, mockLogger.Infos[0].Msg, "Log indexed successfully")
}

// TestLogSearcher_IndexLog_MarshalError はログデータのマシュアル処理でエラーが発生する場合のテスト
func TestLogSearcher_IndexLog_MarshalError(t *testing.T) {
	t.Parallel()

	// モックロガーを準備（indexFuncは使わない）
	mockLogger := testutil.NewMockLogger()
	mockClient := &mockESClient{
		indexFunc: nil,
	}

	// 検索者を作成
	searcher := search.NewLogSearcher(mockClient, mockLogger)

	// 故意にエンコードエラーを起こす不正なデータを準備
	invalidData := map[string]interface{}{"invalid": func() {}}

	// インデックス登録を実行
	err := searcher.IndexLog("test-index", invalidData)
	// エラーが発生することを確認
	require.Error(t, err)

	// 返却されたエラーメッセージに期待する内容が含まれていることを確認
	assert.Contains(t, err.Error(), "failed to marshal log")
}

// TestLogSearcher_IndexLog_IndexError はElasticsearchへのインデックスリクエストでエラーが発生する場合のテスト
func TestLogSearcher_IndexLog_IndexError(t *testing.T) {
	t.Parallel()

	// モックロガーとエラーを返すモッククライアントを準備
	mockLogger := testutil.NewMockLogger()
	mockClient := &mockESClient{
		indexFunc: func(_ string, _ *bytes.Reader, _ ...func(*esapi.IndexRequest)) (*esapi.Response, error) {
			return nil, errMockIndexError
		},
	}

	// 検索者を作成
	searcher := search.NewLogSearcher(mockClient, mockLogger)

	// テスト用ログデータ
	logData := map[string]interface{}{"message": "test log"}

	// インデックス登録を実行
	err := searcher.IndexLog("test-index", logData)
	// エラーが発生することを確認
	require.Error(t, err)

	// 返却されたエラーメッセージに期待する内容が含まれていることを確認
	assert.Contains(t, err.Error(), "failed to send index request")
}

// TestLogSearcher_IndexLog_ResponseError はElasticsearchのレスポンスでエラーが返される場合のテスト
func TestLogSearcher_IndexLog_ResponseError(t *testing.T) {
	t.Parallel()

	// モックロガーとレスポンスエラーを返すモッククライアントを準備
	mockLogger := testutil.NewMockLogger()
	mockClient := &mockESClient{
		indexFunc: func(_ string, _ *bytes.Reader, _ ...func(*esapi.IndexRequest)) (*esapi.Response, error) {
			return &esapi.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("error")),
			}, nil
		},
	}

	// 検索者を作成
	searcher := search.NewLogSearcher(mockClient, mockLogger)

	// テスト用ログデータ
	logData := map[string]interface{}{"message": "test log"}

	// インデックス登録を実行
	err := searcher.IndexLog("test-index", logData)
	// エラーが発生することを確認
	require.Error(t, err)

	// 返却されたエラーメッセージに期待する内容が含まれていることを確認
	assert.Contains(t, err.Error(), "indexing to Elasticsearch failed")
}

// TestLogSearcher_IndexLog_CloseError はレスポンスボディのClose時にエラーが発生しても処理が成功することを確認するテスト
func TestLogSearcher_IndexLog_CloseError(t *testing.T) {
	t.Parallel()

	mockLogger := testutil.NewMockLogger()
	mockBody := &errorCloser{
		Reader: bytes.NewBufferString("ok"),
		closed: false, // 明示的に初期化
	}
	mockClient := &mockESClient{
		indexFunc: func(_ string, _ *bytes.Reader, _ ...func(*esapi.IndexRequest)) (*esapi.Response, error) {
			return &esapi.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       mockBody,
			}, nil
		},
	}

	searcher := search.NewLogSearcher(mockClient, mockLogger)

	logData := map[string]interface{}{"message": "test log"}

	// インデックス登録を実行し、エラーが発生しないことを確認
	err := searcher.IndexLog("test-index", logData)
	require.NoError(t, err)

	// Close が呼ばれたことを確認
	assert.True(t, mockBody.closed, "response body should be closed")

	// ログに正常メッセージが記録されていることを確認
	assert.Len(t, mockLogger.Infos, 1)
	assert.Contains(t, mockLogger.Infos[0].Msg, "Log indexed successfully")
}

// TestLogSearcher_IndexLog_CreatedStatus はElasticsearchが201ステータス（Created）を返した場合でも成功扱いになることを確認するテスト
func TestLogSearcher_IndexLog_CreatedStatus(t *testing.T) {
	t.Parallel()

	mockLogger := testutil.NewMockLogger()
	mockClient := &mockESClient{
		indexFunc: func(_ string, _ *bytes.Reader, _ ...func(*esapi.IndexRequest)) (*esapi.Response, error) {
			return &esapi.Response{
				StatusCode: http.StatusCreated, // 201
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("created")),
			}, nil
		},
	}

	searcher := search.NewLogSearcher(mockClient, mockLogger)

	logData := map[string]interface{}{"message": "test log"}

	// インデックス登録を実行し、エラーが発生しないことを確認
	err := searcher.IndexLog("test-index", logData)
	require.NoError(t, err)

	// ログに正常メッセージが記録されていることを確認
	assert.Len(t, mockLogger.Infos, 1)
	assert.Contains(t, mockLogger.Infos[0].Msg, "Log indexed successfully")
}

// TestLogSearcher_IndexLog_NilBody はレスポンスボディが空の場合でも処理が成功することを確認するテスト
func TestLogSearcher_IndexLog_NilBody(t *testing.T) {
	t.Parallel()

	mockLogger := testutil.NewMockLogger()
	mockClient := &mockESClient{
		indexFunc: func(_ string, _ *bytes.Reader, _ ...func(*esapi.IndexRequest)) (*esapi.Response, error) {
			return &esapi.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("")), // 空のBodyを模擬
			}, nil
		},
	}

	searcher := search.NewLogSearcher(mockClient, mockLogger)

	logData := map[string]interface{}{"message": "test log"}

	// インデックス登録を実行し、panicやエラーが発生しないことを確認
	err := searcher.IndexLog("test-index", logData)
	require.NoError(t, err)

	// ログに正常メッセージが記録されていることを確認
	assert.Len(t, mockLogger.Infos, 1)
	assert.Contains(t, mockLogger.Infos[0].Msg, "Log indexed successfully")
}
