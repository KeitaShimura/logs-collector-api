package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/KeitaShimura/logs-collector-api/internal/testutil"
	testhelper "github.com/KeitaShimura/logs-collector-api/internal/testutil/helper"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// TestGRPC_GetLogs_Success は、正常系でログ取得が成功することを検証します。
func TestGRPC_GetLogs_Success(t *testing.T) {
	t.Parallel()

	// gRPC クライアントと DB をセットアップ
	client, db, _, _, _ := testhelper.SetupGRPCTestHandler(t, false) //nolint:dogsled // テスト対象は client のみのため未使用戻り値は破棄

	// 事前に 1 件のテストログを挿入
	require.NoError(t, testhelper.InsertTestLog(t.Context(), db))

	// GetLogs RPC を呼び出し
	resp, err := client.GetLogs(t.Context(), &pb.GetLogsRequest{
		Service:   testutil.StringPtr("test-service"),
		Level:     testutil.StringPtr("INFO"),
		Limit:     10,
		Offset:    0,
		StartTime: nil,
		EndTime:   nil,
	})
	// エラーなし、レスポンスが取得できていること
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.GreaterOrEqual(t, len(resp.GetLogs()), 1)
}

// TestGRPC_GetLogs_NotFound は、存在しない条件で NotFound エラーが返ることを検証します。
func TestGRPC_GetLogs_NotFound(t *testing.T) {
	t.Parallel()

	// ログを挿入せずにクライアントのみセットアップ
	client, _, _, _, _ := testhelper.SetupGRPCTestHandler(t, false) //nolint:dogsled // テスト対象は client のみのため未使用戻り値は破棄

	// 存在しないサービス・レベルで呼び出し
	resp, err := client.GetLogs(t.Context(), &pb.GetLogsRequest{
		Service:   testutil.StringPtr("non-existent-service"),
		Level:     testutil.StringPtr("DEBUG"),
		Limit:     10,
		Offset:    0,
		StartTime: nil,
		EndTime:   nil,
	})
	// エラーおよびステータスコード確認
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
	// レスポンスは nil であること
	require.Nil(t, resp)
}

// TestGRPC_GetLogs_DBConnectionFailure は、DB 接続障害時に Internal エラーとなることを検証します。
func TestGRPC_GetLogs_DBConnectionFailure(t *testing.T) {
	t.Parallel()

	// closeDB=true に設定し、接続を即時クローズ
	client, _, _, _, _ := testhelper.SetupGRPCTestHandler(t, true) //nolint:dogsled // テスト対象は client のみのため未使用戻り値は破棄

	// RPC 呼び出し
	_, err := client.GetLogs(t.Context(), &pb.GetLogsRequest{
		Service:   testutil.StringPtr("test-service"),
		Level:     testutil.StringPtr("INFO"),
		Limit:     10,
		Offset:    0,
		StartTime: nil,
		EndTime:   nil,
	})
	// Internal エラーを期待
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())
}

// TestGRPC_GetLogs_EmptyQuery は、Level が nil (条件なし) の場合でも正常に取得できることを検証します。
func TestGRPC_GetLogs_EmptyQuery(t *testing.T) {
	t.Parallel()

	client, db, _, _, _ := testhelper.SetupGRPCTestHandler(t, false) //nolint:dogsled // テスト対象は client のみのため未使用戻り値は破棄
	// テストログを挿入
	require.NoError(t, testhelper.InsertTestLog(t.Context(), db))

	// Level を nil に設定
	resp, err := client.GetLogs(t.Context(), &pb.GetLogsRequest{
		Service:   testutil.StringPtr("test-service"),
		Level:     nil, // レベル条件なし
		Limit:     10,
		Offset:    0,
		StartTime: nil,
		EndTime:   nil,
	})
	// 成功とログ件数の確認
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.GreaterOrEqual(t, len(resp.GetLogs()), 1)
}

// TestGRPC_GetLogs_OffsetTooLarge は、オフセットが範囲外の場合 NotFound エラーとなることを検証します。
func TestGRPC_GetLogs_OffsetTooLarge(t *testing.T) {
	t.Parallel()

	client, db, _, _, _ := testhelper.SetupGRPCTestHandler(t, false) //nolint:dogsled // テスト対象は client のみのため未使用戻り値は破棄
	// テストログを挿入
	require.NoError(t, testhelper.InsertTestLog(t.Context(), db))

	// 存在しない大きなオフセットを指定
	resp, err := client.GetLogs(t.Context(), &pb.GetLogsRequest{
		Service:   testutil.StringPtr("test-service"),
		Level:     testutil.StringPtr("INFO"),
		Limit:     10,
		Offset:    9999, // 存在しないオフセット
		StartTime: nil,
		EndTime:   nil,
	})
	// NotFound エラーを期待
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
	require.Nil(t, resp)
}
