// internal/interface/restmw/validation.go
package restmw

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/KeitaShimura/logs-collector-api/internal/app/helper"
	"github.com/KeitaShimura/logs-collector-api/internal/app/middleware"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// ValidationMiddlewareSendLog は Echo の POST /logs 用バリデーションミドルウェア
func ValidationMiddlewareSendLog(logger logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(echoCtx echo.Context) error {
			var req pb.SendLogRequest

			// リクエストボディを読み込む
			body, err := io.ReadAll(echoCtx.Request().Body)
			if err != nil {
				logger.Error("failed to read body", err)

				return echoCtx.JSON(http.StatusBadRequest, echo.Map{"error": "failed to read request"})
			}

			// 読み取った body を再利用できるように復元（後続で再度読み取れるように）
			echoCtx.Request().Body = io.NopCloser(bytes.NewBuffer(body))

			// JSON → Protobuf へ変換（Unmarshal）
			if err := protojson.Unmarshal(body, &req); err != nil {
				logger.Error("invalid protobuf json", err)

				return echoCtx.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
			}

			// バリデーション
			if err := middleware.ValidateSendLogRequest(&req); err != nil {
				logger.Warn("SendLog validation failed", "error", err)

				return echoCtx.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
			}

			// バリデーション済みのリクエストをコンテキストに格納（ハンドラで再利用するため）
			echoCtx.Set("send_log_request", &req)

			// 次のハンドラへ
			return next(echoCtx)
		}
	}
}

// ValidationMiddlewareGetLogs は Echo の GET /logs 用のバリデーションミドルウェア
func ValidationMiddlewareGetLogs(log logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(echoCtx echo.Context) error {
			service, level, limit, offset, err := helper.ParseQueryParams(echoCtx, log)
			if err != nil {
				// ParseQueryParams 側で echo.NewHTTPError が返るのでそのまま返してOK
				return fmt.Errorf("failed to parse query params: %w", err)
			}

			// int32キャスト前に安全確認（ParseQueryParams で範囲チェック済みなのでOK）
			//
			//nolint:gosec
			limit32 := int32(limit)
			//nolint:gosec
			offset32 := int32(offset)

			// バリデーション
			if err := middleware.ValidateGetLogsRequest(service, level, limit32, offset32); err != nil {
				log.Warn("GetLogs validation failed", "error", err)

				return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// 次のハンドラへ
			return next(echoCtx)
		}
	}
}
