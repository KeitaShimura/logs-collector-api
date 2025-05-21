package middleware

import (
	"errors"
	"fmt"
	"time"

	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// 共通エラー定義
var (
	ErrLogRequired       = errors.New("log is required")
	ErrServiceEmpty      = errors.New("log.service must not be empty")
	ErrMessageEmpty      = errors.New("log.message must not be empty")
	ErrLevelEmpty        = errors.New("log.level must not be empty")
	ErrInvalidLogLevel   = errors.New("invalid log.level")
	ErrTraceIDEmpty      = errors.New("log.trace_id must not be empty")
	ErrTimestampRequired = errors.New("log.timestamp is required")
	ErrTimestampInFuture = errors.New("log.timestamp cannot be in the future")

	ErrServiceParamEmpty = errors.New("service must not be empty")
	ErrLimitOutOfRange   = errors.New("limit must be between 1 and 1000")
	ErrOffsetNegative    = errors.New("offset must be >= 0")
)

// ValidateSendLogRequest はログ送信リクエストのバリデーションを行う
func ValidateSendLogRequest(req *pb.SendLogRequest) error {
	log := req.GetLog()
	if log == nil {
		return ErrLogRequired
	}

	if log.GetService() == "" {
		return ErrServiceEmpty
	}

	if log.GetMessage() == "" {
		return ErrMessageEmpty
	}

	if log.GetLevel() == "" {
		return ErrLevelEmpty
	}

	if !IsValidLogLevel(log.GetLevel()) {
		return fmt.Errorf("%w: %s", ErrInvalidLogLevel, log.GetLevel())
	}

	if log.GetTraceId() == "" {
		return ErrTraceIDEmpty
	}

	if log.GetTimestamp() == nil {
		return ErrTimestampRequired
	}

	ts := log.GetTimestamp().AsTime()
	if ts.After(time.Now().Add(1 * time.Minute)) {
		return ErrTimestampInFuture
	}

	return nil
}

// ValidateGetLogsRequest はログ取得リクエストのバリデーションを行う
func ValidateGetLogsRequest(service, level string, limit, offset int32) error {
	if service == "" {
		return ErrServiceParamEmpty
	}

	if level != "" {
		if !IsValidLogLevel(level) {
			return fmt.Errorf("%w: %s", ErrInvalidLogLevel, level)
		}
	}

	if limit < 1 || limit > 1000 {
		return ErrLimitOutOfRange
	}

	if offset < 0 {
		return ErrOffsetNegative
	}

	return nil
}

// IsValidLogLevel はログレベルが有効かどうかを判定する
func IsValidLogLevel(level string) bool {
	validLevels := map[string]struct{}{
		"DEBUG": {}, "INFO": {}, "WARN": {}, "ERROR": {},
	}
	_, ok := validLevels[level]

	return ok
}
