package queue_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/infra/queue"
	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
)

// 共通エラー定義
var errNATSPublish = errors.New("nats publish error")

// --- モックNATS ---

// MockNATSConn は NATS の Publish メソッドをモック化する構造体。
// テスト中に呼び出し状況や送信内容を検証するために使用する。
type MockNATSConn struct {
	publishCalled bool   // Publish が呼ばれたかどうかのフラグ
	subject       string // Publish に渡された subject
	data          []byte // Publish に渡されたデータ
	err           error  // Publish が返すエラー（任意に指定可能）
}

// Publish は Publish 呼び出し状況と引数を記録し、指定されたエラーを返すモック実装。
func (m *MockNATSConn) Publish(subject string, data []byte) error {
	m.publishCalled = true
	m.subject = subject
	m.data = data

	return m.err
}

// --- テスト ---

// TestProducer_Publish_Success は、ログメッセージが正常に NATS に送信される場合のテスト
func TestProducer_Publish_Success(t *testing.T) {
	t.Parallel()

	// モックの NATS 接続とロガーを用意
	mockConn := &MockNATSConn{
		publishCalled: false,
		subject:       "",
		data:          []byte{},
		err:           nil,
	}
	mockLogger := testutil.NewMockLogger()

	// テスト対象の Producer を生成
	producer := &queue.Producer{
		Conn:   mockConn,
		Logger: mockLogger,
	}

	// ログメッセージを作成
	msg := queue.LogMessage{
		ID:        "abc123",
		TraceID:   "",
		Timestamp: "",
		Level:     "",
		Service:   "",
		Message:   "test message",
		Metadata:  map[string]string{},
	}

	// JSON 変換して期待値を作成
	expectedJSON, err := json.Marshal(msg)
	require.NoError(t, err)

	// メッセージ送信処理を実行
	err = producer.Publish("logs.test", msg)
	require.NoError(t, err)

	// モックが正しく呼び出されたか検証
	assert.True(t, mockConn.publishCalled)
	assert.Equal(t, "logs.test", mockConn.subject)
	assert.JSONEq(t, string(expectedJSON), string(mockConn.data))

	// ログ出力の検証
	assert.Len(t, mockLogger.Infos, 1)
	assert.Contains(t, mockLogger.Infos[0].Msg, "Message published successfully")
}

// TestProducer_Publish_PublishError は、NATS への送信時にエラーが発生する場合のテスト
func TestProducer_Publish_PublishError(t *testing.T) {
	t.Parallel()

	// Publish エラーを返すモック接続を用意
	mockConn := &MockNATSConn{
		publishCalled: false,
		subject:       "",
		data:          nil,
		err:           errNATSPublish,
	}
	mockLogger := testutil.NewMockLogger()

	// テスト対象の Producer を生成
	producer := &queue.Producer{
		Conn:   mockConn,
		Logger: mockLogger,
	}

	// ログメッセージを作成
	msg := queue.LogMessage{
		ID:        "error123",
		TraceID:   "",
		Timestamp: "",
		Level:     "",
		Service:   "",
		Message:   "fail",
		Metadata:  map[string]string{},
	}

	// メッセージ送信処理を実行（失敗を期待）
	err := producer.Publish("logs.test", msg)

	// エラーが返されたことを確認
	require.Error(t, err)

	// モックが呼び出されたことを確認
	assert.True(t, mockConn.publishCalled)
}
