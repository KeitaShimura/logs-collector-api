package rest

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/KeitaShimura/logs-collector-api/internal/app/helper"
	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// LogHandler はログ関連の REST リクエストを処理するハンドラー構造体
type LogHandler struct {
	logUseCase usecase.LogUseCase
	logger     logger.Logger
}

// NewLogHandler は新しい LogHandler インスタンスを作成するコンストラクタ関数
func NewLogHandler(uc usecase.LogUseCase, logger logger.Logger) *LogHandler {
	return &LogHandler{
		logUseCase: uc,
		logger:     logger,
	}
}

// SendLogRequest は REST API 経由で受け取るログ保存リクエストのリクエストボディ構造体
type SendLogRequest struct {
	ID        string            `json:"id"`
	TraceID   string            `json:"traceId"`
	Message   string            `json:"message"   validate:"required"`
	Level     string            `json:"level"     validate:"required,oneof=debug info warn error"`
	Service   string            `json:"service"   validate:"required"`
	Timestamp string            `json:"timestamp" validate:"omitempty,datetime=2006-01-02T15:04:05Z"`
	Metadata  map[string]string `json:"metadata"`
}

// SuccessResponse は成功レスポンスの構造体
type SuccessResponse struct {
	Status string `json:"status"`
}

// ErrorResponse はエラーレスポンスの構造体
type ErrorResponse struct {
	Error string `json:"error"`
}

// SendLog は新しいログを登録するエンドポイント
// SendLog godoc
// @Summary Register a new log entry
// @Description Registers a new log entry with ID, trace_id, message, level, service, timestamp, and metadata.
// @Tags logs
// @Accept  json
// @Produce  json
// @Param   log body SendLogRequest true "Log payload"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /logs [post]
func (h *LogHandler) SendLog(echoCtx echo.Context) error {
	var req SendLogRequest

	// リクエストボディをバインド（JSON → 構造体へ変換）
	req, timestamp, err := h.ParseAndValidateRequest(echoCtx)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return RespondJSON(echoCtx, httpErr.Code, ErrorResponse{Error: httpErr.Error()})
		}

		return RespondJSON(echoCtx, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	// ID 補完（未指定の場合はサーバー側で生成）
	logID := req.ID
	if logID == "" {
		logID = uuid.NewString()
	}

	// Metadata 補完（未指定の場合は空mapを設定）
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	// ログエントリを作成
	logEntry := &model.Log{
		ID:        logID,
		TraceID:   req.TraceID,
		Timestamp: timestamp,
		Level:     req.Level,
		Service:   req.Service,
		Message:   req.Message,
		Metadata:  metadata,
	}

	// ユースケースを呼び出してログを保存
	if err := h.logUseCase.SendLog(echoCtx.Request().Context(), logEntry); err != nil {
		statusCode := AppErrorToHTTPStatus(err)
		h.logger.Error("Failed to save log entry", err,
			"ID", logEntry.ID,
			"TraceID", logEntry.TraceID,
			"Timestamp", logEntry.Timestamp,
			"Level", logEntry.Level,
			"Service", logEntry.Service,
			"Message", logEntry.Message,
			"Metadata", logEntry.Metadata,
		)

		return RespondJSON(echoCtx, statusCode, ErrorResponse{Error: err.Error()})
	}

	// 保存成功ログを出力
	h.logger.Info("Log entry saved successfully",
		"ID", logEntry.ID,
		"TraceID", logEntry.TraceID,
		"Timestamp", logEntry.Timestamp,
		"Level", logEntry.Level,
		"Service", logEntry.Service,
		"Message", logEntry.Message,
		"Metadata", logEntry.Metadata,
	)

	return RespondJSON(echoCtx, http.StatusOK, SuccessResponse{Status: "success"})
}

// GetLogs は指定されたクエリパラメータに一致するログを取得するエンドポイント
// GetLogs godoc
// @Summary Retrieve logs
// @Description Retrieves logs that match the provided query parameters (service, level, limit, offset).
// @Tags logs
// @Accept  json
// @Produce  json
// @Param   service query string false "Service name (e.g., frontend)" example(frontend)
// @Param   level   query string false "Log level" Enums(info, warn, error) example(info)
// @Param   limit   query int    false "Number of logs to return (default: 100)" minimum(1) maximum(1000) example(100)
// @Param   offset  query int    false "Offset for pagination (default: 0)" minimum(0) example(0)
// @Success 200 {array} model.Log
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /logs [get]
func (h *LogHandler) GetLogs(echoCtx echo.Context) error {
	service, level, limit, offset, err := helper.ParseQueryParams(echoCtx, h.logger)
	if err != nil {
		h.logger.Warn("Invalid query parameters", "error", err)

		var httpErr *echo.HTTPError

		if errors.As(err, &httpErr) {
			return RespondJSON(echoCtx, httpErr.Code, ErrorResponse{Error: err.Error()})
		}

		return RespondJSON(echoCtx, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}

	// ユースケースからログを取得
	logs, err := h.logUseCase.GetLogs(echoCtx.Request().Context(), service, level, limit, offset)
	if err != nil {
		statusCode := AppErrorToHTTPStatus(err)

		// エラーログを出力
		h.logger.Error("Failed to fetch logs", err,
			"Service", service,
			"Level", level,
			"Limit", limit,
			"Offset", offset,
		)

		return RespondJSON(echoCtx, statusCode, ErrorResponse{Error: err.Error()})
	}

	// 取得成功ログを出力
	h.logger.Info("Logs fetched successfully",
		"Service", service,
		"Level", level,
		"Limit", limit,
		"Offset", offset,
		"ResultCount", len(logs),
	)

	return RespondJSON(echoCtx, http.StatusOK, logs)
}

// RespondJSON はJSONでレスポンスを返しつつ、内部エラーをラップして返す
func RespondJSON(echoCtx echo.Context, code int, body interface{}) error {
	if jsonErr := echoCtx.JSON(code, body); jsonErr != nil {
		return fmt.Errorf("failed to return JSON response: %w", jsonErr)
	}

	return nil
}

// ParseAndValidateRequest はリクエストボディをパース・バリデーションし、パース済み構造体とタイムスタンプを返す関数
func (h *LogHandler) ParseAndValidateRequest(echoCtx echo.Context) (SendLogRequest, time.Time, error) {
	var req SendLogRequest

	// --- リクエストボディをバインド（JSON → 構造体へ変換） ---
	if err := echoCtx.Bind(&req); err != nil {
		// バインド失敗時は警告ログを出力し、400エラーを返却
		h.logger.Warn("Failed to bind request body", "error", err)

		return req, time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// --- バリデーションチェック ---
	if err := echoCtx.Validate(&req); err != nil {
		// バリデーション失敗時は警告ログを出力し、400エラーを返却
		h.logger.Warn("Validation failed", "error", err)

		return req, time.Time{}, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// --- タイムスタンプをパース（RFC3339形式を期待） ---
	timestamp, err := ParseTimestamp(req.Timestamp)
	if err != nil {
		// タイムスタンプのパース失敗時は警告ログを出力し、400エラーを返却
		h.logger.Warn("Timestamp parsing failed", "timestamp", req.Timestamp, "error", err)

		return req, time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "invalid timestamp format (use RFC3339)")
	}

	return req, timestamp, nil
}

// ParseTimestamp は文字列で渡されたタイムスタンプを time.Time 型にパースするヘルパー関数
func ParseTimestamp(timestampStr string) (time.Time, error) {
	if timestampStr == "" {
		// タイムスタンプが空の場合は現在時刻を返却
		return time.Now(), nil
	}

	// RFC3339 形式でパースを試みる
	parsed, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		// パース失敗時はエラーをラップして返却
		return time.Time{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	return parsed, nil
}

// AppErrorToHTTPStatus はアプリケーションエラーを HTTP ステータスコードに変換する
func AppErrorToHTTPStatus(err error) int {
	switch {
	case errors.Is(err, usecase.ErrValidationFailure):
		return http.StatusBadRequest
	case errors.Is(err, usecase.ErrRepositoryFailure):
		return http.StatusInternalServerError
	case errors.Is(err, usecase.ErrNoLogsFound):
		return http.StatusNotFound
	// 必要に応じて追加
	default:
		return http.StatusInternalServerError
	}
}
