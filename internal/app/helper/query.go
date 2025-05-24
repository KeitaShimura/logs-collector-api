package helper

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

const (
	defaultLimit  = 100
	defaultOffset = 0
)

// QueryParams はログ取得用のクエリパラメータを表す構造体
type QueryParams struct {
	Service string
	Level   string
	Limit   int
	Offset  int
}

// ParseQueryParams はクエリパラメータを抽出・バリデーションするヘルパー関数
func ParseQueryParams(echoCtx echo.Context, log logger.Logger) (*QueryParams, error) {
	service := echoCtx.QueryParam("service")
	level := echoCtx.QueryParam("level")
	limit := defaultLimit

	if limitStr := echoCtx.QueryParam("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			// 数値変換エラーの場合は警告ログを出し、400エラーを返す
			log.Warn("Invalid limit parameter", "value", limitStr, "error", err)

			return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid limit parameter")
		}

		limit = parsedLimit
	}

	offset := defaultOffset

	if offsetStr := echoCtx.QueryParam("offset"); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil {
			// 数値変換エラーの場合は警告ログを出し、400エラーを返す
			log.Warn("Invalid offset parameter", "value", offsetStr, "error", err)

			return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid offset parameter")
		}

		offset = parsedOffset
	}

	return &QueryParams{
		Service: service,
		Level:   level,
		Limit:   limit,
		Offset:  offset,
	}, nil
}
