package helper

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/KeitaShimura/logs-collector-api/internal/logger"
)

// ParseQueryParams はクエリパラメータを抽出・バリデーションするヘルパー関数
func ParseQueryParams(echoCtx echo.Context, log logger.Logger) (string, string, int, int, error) {
	// service パラメータを取得
	service := echoCtx.QueryParam("service")

	// level パラメータを取得
	level := echoCtx.QueryParam("level")

	// limit パラメータ（デフォルト: 100）
	limit := 100

	if limitStr := echoCtx.QueryParam("limit"); limitStr != "" {
		// limit を数値に変換
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			// 数値変換エラーの場合は警告ログを出し、400エラーを返す
			log.Warn("Invalid limit parameter", "value", limitStr, "error", err)

			return "", "", 0, 0, echo.NewHTTPError(http.StatusBadRequest, "invalid limit parameter")
		}

		limit = parsedLimit
	}

	// limit の範囲チェック
	if limit < 1 || limit > 1000 {
		// 範囲外場合は警告ログを出し、400エラーを返す
		log.Warn("Limit parameter out of range", "value", limit)

		return "", "", 0, 0, echo.NewHTTPError(http.StatusBadRequest, "limit must be between 1 and 1000")
	}

	// offset パラメータ（デフォルト: 0）
	offset := 0

	if offsetStr := echoCtx.QueryParam("offset"); offsetStr != "" {
		// offset を数値に変換
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil {
			// 数値変換エラーの場合は警告ログを出し、400エラーを返す
			log.Warn("Invalid offset parameter", "value", offsetStr, "error", err)

			return "", "", 0, 0, echo.NewHTTPError(http.StatusBadRequest, "invalid offset parameter")
		}

		offset = parsedOffset
	}

	// offset の負値チェック
	if offset < 0 {
		// 負の値の場合は警告ログを出し、400エラーを返す
		log.Warn("Offset parameter is negative", "value", offset)

		return "", "", 0, 0, echo.NewHTTPError(http.StatusBadRequest, "offset must be >= 0")
	}

	return service, level, limit, offset, nil
}
