package validator

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// CustomValidator は echo のバリデーションをラップした構造体
type CustomValidator struct {
	validator *validator.Validate
}

// Validate は echo の Validator インターフェース実装
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// NewValidator はバリデーターのインスタンスを返す
func NewValidator() *CustomValidator {
	return &CustomValidator{
		validator: validator.New(),
	}
}
