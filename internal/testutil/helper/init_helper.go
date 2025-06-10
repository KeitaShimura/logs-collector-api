package testhelper

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/infra/db"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
)

// initTestUseCase はユースケース層のテスト環境を初期化します。
// 成功時には LogUseCaseImpl インスタンスとリソースを返却します。
func initTestUseCase(t *testing.T) (*usecase.LogUseCaseImpl, *sql.DB, *appmock.Producer, *appmock.LogSearcher) {
	t.Helper()

	// テスト用のBoilエグゼキュータとSQL DBを初期化
	exec, sqlDB, err := newTestBoilExecutor()
	// 初期化に失敗したらテストを中断
	require.NoError(t, err)

	// ロガーを生成
	log := logger.NewLogger()
	// リポジトリにBoilエグゼキュータとロガーを注入
	repo := db.NewLogRepository(exec, log)

	// メッセージ送信用のモックProducerを生成
	mockProducer := appmock.NewProducer()
	// 検索用のモックSearcherを生成
	mockSearcher := appmock.NewLogSearcher()

	// ユースケース実装に依存性を注入して生成
	uc := usecase.NewLogUseCase(repo, mockProducer, mockSearcher, log)

	// ユースケース、DB、モックProducer/Searcherを返却
	return uc, sqlDB, mockProducer, mockSearcher
}
