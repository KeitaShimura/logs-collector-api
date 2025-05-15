package middleware

import (
	"context"
	"time"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

type contextKey string

const (
	ContextKeyTraceID   contextKey = "trace_id"
	ContextKeyRequestID contextKey = "request_id"
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyClientIP  contextKey = "client_ip"
)

// LoggingHandler はリクエストの処理結果を共通形式でログ出力する関数
func LoggingHandler(
	ctx context.Context,
	log logger.Logger,
	method string,
	statusCode string,
	duration time.Duration,
	err error,
) {
	// コンテキストから各種メタデータを取得（trace_id, request_id など）
	traceID := GetStringFromContext(ctx, ContextKeyTraceID)
	requestID := GetStringFromContext(ctx, ContextKeyRequestID)
	userID := GetStringFromContext(ctx, ContextKeyUserID)
	clientIP := GetStringFromContext(ctx, ContextKeyClientIP)

	// ログ出力用のフィールドを構造化形式で整形
	fields := []interface{}{
		"trace_id", traceID,
		"request_id", requestID,
		"method", method,
		"status_code", statusCode,
		"duration_ms", duration.Milliseconds(),
		"user_id", userID,
		"client_ip", clientIP,
	}

	// 処理成功・失敗に応じてログレベルを切り替えて出力
	if err != nil {
		log.Error("request failed", err, fields...)
	} else {
		log.Info("request completed", fields...)
	}
}

// GetStringFromContext は context から安全に string 型の値を取得するユーティリティ関数
func GetStringFromContext(ctx context.Context, key interface{}) string {
	if val := ctx.Value(key); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}

	return ""
}
