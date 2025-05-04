package rest

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const statusClientClosedRequest = 499 // 非標準: クライアントが接続をキャンセル

// GRPCErrorToHTTPStatus は gRPC ステータスコードを対応する HTTP ステータスコードに変換する関数
func GRPCErrorToHTTPStatus(err error) int {
	// gRPC エラーから status.Status を取得
	grpcStatus, ok := status.FromError(err)
	if !ok {
		// gRPC ステータスでない場合は内部サーバーエラーを返す
		return http.StatusInternalServerError
	}

	// gRPC コードと HTTP ステータスコードの対応表
	grpcToHTTPStatusMap := map[codes.Code]int{
		codes.OK:                 http.StatusOK,
		codes.Canceled:           statusClientClosedRequest, // 499: クライアントが閉じた場合（独自定義）
		codes.Unknown:            http.StatusInternalServerError,
		codes.InvalidArgument:    http.StatusBadRequest,
		codes.DeadlineExceeded:   http.StatusGatewayTimeout,
		codes.NotFound:           http.StatusNotFound,
		codes.AlreadyExists:      http.StatusConflict,
		codes.PermissionDenied:   http.StatusForbidden,
		codes.ResourceExhausted:  http.StatusTooManyRequests,
		codes.FailedPrecondition: http.StatusPreconditionFailed,
		codes.Aborted:            http.StatusConflict,
		codes.OutOfRange:         http.StatusRequestedRangeNotSatisfiable,
		codes.Unimplemented:      http.StatusNotImplemented,
		codes.Internal:           http.StatusInternalServerError,
		codes.Unavailable:        http.StatusServiceUnavailable,
		codes.DataLoss:           http.StatusInternalServerError,
		codes.Unauthenticated:    http.StatusUnauthorized,
	}

	// 対応する HTTP ステータスコードがあれば返却
	if httpCode, exists := grpcToHTTPStatusMap[grpcStatus.Code()]; exists {
		return httpCode
	}

	// マッピングがない場合は内部サーバーエラーを返す
	return http.StatusInternalServerError
}
