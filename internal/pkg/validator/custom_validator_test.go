package validator_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KeitaShimura/logs-collector-api/internal/pkg/validator"
)

// TestCustomValidator_Validate_Success は必須項目が正しく入力された場合にバリデーションが成功することを確認するテスト
func TestCustomValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	// カスタムバリデーターを初期化
	validatorInstance := validator.NewValidator()

	// テスト用構造体（Name は必須）
	type TestStruct struct {
		Name string `validate:"required"`
	}

	// 必須項目が埋まっているケース
	s := TestStruct{Name: "ok"}

	// バリデーション実行
	err := validatorInstance.Validate(s)

	// エラーがないことを確認（成功）
	require.NoError(t, err)
}

// TestCustomValidator_Validate_Failure は必須項目が未入力の場合にバリデーションが失敗することを確認するテスト
func TestCustomValidator_Validate_Failure(t *testing.T) {
	t.Parallel()

	// カスタムバリデーターを初期化
	validatorInstance := validator.NewValidator()

	// テスト用構造体（Name は必須）
	type TestStruct struct {
		Name string `validate:"required"`
	}

	// 必須項目が空のケース
	s := TestStruct{
		Name: "",
	}

	// バリデーション実行
	err := validatorInstance.Validate(s)

	// エラーが返ることを確認（失敗）
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation failed")
}
