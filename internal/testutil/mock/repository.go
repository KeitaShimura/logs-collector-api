package mock

import (
	"context"
	"errors"
	"fmt"

	"github.com/stretchr/testify/mock"

	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
)

// 共通エラー定義
var (
	errMockLogsNil         = errors.New("mock GetLogs returned nil logs")
	errMockLogsTypeInvalid = errors.New("mock GetLogs type assertion failed")
)

// LogRepository は LogRepository を模倣するモック構造体
type LogRepository struct{ mock.Mock }

// SendLog はログの永続化処理を模倣する
func (m *LogRepository) SendLog(ctx context.Context, log *model.Log) error {
	args := m.Called(ctx, log)

	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock SendLog error: %w", err)
	}

	return nil
}

// GetLogs はログ検索処理を模倣する
func (m *LogRepository) GetLogs(
	ctx context.Context,
	service, level string,
	limit, offset int,
) ([]model.Log, error) {
	args := m.Called(ctx, service, level, limit, offset)

	if args.Get(0) == nil {
		return nil, fmt.Errorf("%w", errMockLogsNil)
	}

	logs, ok := args.Get(0).([]model.Log)
	if !ok {
		return nil, fmt.Errorf("%w: got=%T", errMockLogsTypeInvalid, args.Get(0))
	}

	if args.Error(1) != nil {
		return nil, fmt.Errorf("mock GetLogs error: %w", args.Error(1))
	}

	return logs, nil
}
