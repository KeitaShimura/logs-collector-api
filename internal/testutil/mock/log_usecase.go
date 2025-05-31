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
	ErrMockSendLog        = errors.New("mock SendLog error")
	ErrMockGetLogs        = errors.New("mock GetLogs error")
	ErrUnexpectedMockType = errors.New("unexpected mock type")
)

// LogUseCase は LogUseCase インターフェースのモック実装
type LogUseCase struct {
	mock.Mock
}

// SendLog はモックの SendLog メソッドを呼び出す
func (m *LogUseCase) SendLog(ctx context.Context, log *model.Log) error {
	args := m.Called(ctx, log)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("%w: %w", ErrMockSendLog, err)
	}

	return nil
}

// GetLogs はモックの GetLogs メソッドを呼び出す
func (m *LogUseCase) GetLogs(ctx context.Context, service, level string, limit, offset int) ([]model.Log, error) {
	args := m.Called(ctx, service, level, limit, offset)

	v := args.Get(0)
	logs, ok := v.([]model.Log)

	if !ok {
		return nil, fmt.Errorf("%w: %T", ErrUnexpectedMockType, v)
	}

	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMockGetLogs, err)
	}

	return logs, nil
}
