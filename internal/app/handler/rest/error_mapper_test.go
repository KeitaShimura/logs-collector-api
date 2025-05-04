package rest_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
)

// 非gRPCエラー用のダミーエラー定義
var errSomeNonGRPC = errors.New("some non-gRPC error")

// TestGRPCErrorToHTTPStatus_NormalCases は、各gRPCステータスコードが正しくHTTPステータスにマッピングされることを確認するテスト
func TestGRPCErrorToHTTPStatus_NormalCases(t *testing.T) {
	t.Parallel()

	// テストケース一覧
	tests := []struct {
		name         string     // サブテスト名
		grpcCode     codes.Code // 入力するgRPCコード
		expectedHTTP int        // 期待するHTTPステータス
	}{
		{"OK", codes.OK, http.StatusOK},
		{"Canceled", codes.Canceled, 499},
		{"Unknown", codes.Unknown, http.StatusInternalServerError},
		{"InvalidArgument", codes.InvalidArgument, http.StatusBadRequest},
		{"DeadlineExceeded", codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{"NotFound", codes.NotFound, http.StatusNotFound},
		{"AlreadyExists", codes.AlreadyExists, http.StatusConflict},
		{"PermissionDenied", codes.PermissionDenied, http.StatusForbidden},
		{"ResourceExhausted", codes.ResourceExhausted, http.StatusTooManyRequests},
		{"FailedPrecondition", codes.FailedPrecondition, http.StatusPreconditionFailed},
		{"Aborted", codes.Aborted, http.StatusConflict},
		{"OutOfRange", codes.OutOfRange, http.StatusRequestedRangeNotSatisfiable},
		{"Unimplemented", codes.Unimplemented, http.StatusNotImplemented},
		{"Internal", codes.Internal, http.StatusInternalServerError},
		{"Unavailable", codes.Unavailable, http.StatusServiceUnavailable},
		{"DataLoss", codes.DataLoss, http.StatusInternalServerError},
		{"Unauthenticated", codes.Unauthenticated, http.StatusUnauthorized},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// 該当gRPCコードのエラーを生成
			err := status.Error(testCase.grpcCode, "test error")

			// マッピング関数を呼び出し
			got := rest.GRPCErrorToHTTPStatus(err)

			// 期待されるHTTPステータスと比較
			require.Equal(t, testCase.expectedHTTP, got)
		})
	}
}

// TestGRPCErrorToHTTPStatus_UnknownError は、gRPCエラーでない通常のエラーを渡した場合に500が返ることを確認するテスト
func TestGRPCErrorToHTTPStatus_UnknownError(t *testing.T) {
	t.Parallel()

	// 非gRPCエラーを渡す
	got := rest.GRPCErrorToHTTPStatus(errSomeNonGRPC)

	// HTTP 500 が返ることを確認
	require.Equal(t, http.StatusInternalServerError, got)
}

// TestGRPCErrorToHTTPStatus_UnmappedGRPCCode は、マッピングされていない未知のgRPCコードを渡した場合に500が返ることを確認するテスト
func TestGRPCErrorToHTTPStatus_UnmappedGRPCCode(t *testing.T) {
	t.Parallel()

	// マッピングされていないgRPCコードを用意（例: 999）
	err := status.Error(codes.Code(999), "unknown code")

	// マッピング関数を呼び出し
	got := rest.GRPCErrorToHTTPStatus(err)

	// HTTP 500 が返ることを確認
	require.Equal(t, http.StatusInternalServerError, got)
}
