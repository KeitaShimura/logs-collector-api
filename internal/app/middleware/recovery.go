package middleware

import (
	"runtime/debug"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// RecoveryHandler は panic を回収し、詳細ログを残す共通関数
func RecoveryHandler(log logger.Logger, contextInfo map[string]interface{}) string {
	// スタックトレースを取得
	stack := string(debug.Stack())

	// ログ出力用のフィールドを整形
	logFields := []interface{}{
		"stack", stack,
	}
	for k, v := range contextInfo {
		logFields = append(logFields, k, v)
	}

	// 構造化ログとして panic 内容と付随情報を記録
	log.Error("panic recovered", nil, logFields...)

	return stack
}
