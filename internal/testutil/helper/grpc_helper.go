package testhelper

import (
	"context"
	"database/sql"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/test/bufconn"

	grpcHandler "github.com/KeitaShimura/logs-collector-api/internal/app/handler/grpc"
	grpcmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/grpc"
	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
	appmock "github.com/KeitaShimura/logs-collector-api/internal/testutil/mock"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// テスト用バッファサイズと gRPC タイムアウトを定義
const (
	bufSize     = 1024 * 1024
	grpcTimeout = 2 * time.Second
)

// SetupGRPCTestHandler は、テスト用の gRPC クライアントを初期化し、
// LogServiceClient、SQL DB、gRPC サーバ、モックプロデューサ・サーチャを返します。
// closeDB が true の場合は、即時に DB 接続を閉じます。
//
//nolint:ireturn // gRPC クライアントの具体型は unexported なので、ireturn 警告は許容する
func SetupGRPCTestHandler(
	t *testing.T,
	closeDB bool,
) (
	pb.LogServiceClient,
	*sql.DB,
	*grpc.Server,
	*appmock.Producer,
	*appmock.LogSearcher,
) {
	t.Helper()

	// ユースケース層とモックを初期化
	uc, sqlDB, mockProducer, mockSearcher := initTestUseCase(t)

	// ロガーを生成
	log := logger.NewLogger()
	// gRPC サーバとバッファリスナを初期化
	server, lis := initGRPCServer(uc, log)

	// オプションで DB を即時クローズ
	if closeDB {
		sqlDB.Close()
	}

	// bufconn 用 Dialer を定義
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	// gRPC クライアント接続を生成（passthrough:/// を利用）
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	// テスト終了時のクリーンアップ処理を登録
	t.Cleanup(func() {
		conn.Close()  // クライアント接続をクローズ
		server.Stop() // gRPC サーバを停止

		// closeDB が false の場合は DB をクローズ
		if !closeDB {
			sqlDB.Close()
		}
	})

	// LogServiceClient インスタンスを生成
	client := pb.NewLogServiceClient(conn)

	return client, sqlDB, server, mockProducer, mockSearcher
}

// initGRPCServer は、bufconn リスナと gRPC サーバを初期化します。
// 各種ミドルウェア（ロギング、タイムアウト、リカバリ、バリデーション）を設定して起動します。
func initGRPCServer(logUseCase usecase.LogUseCase, log logger.Logger) (*grpc.Server, *bufconn.Listener) {
	logHandler := grpcHandler.NewLogHandler(logUseCase, log)

	// gRPC サーバを生成し、UnaryInterceptor チェーンを設定
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcmw.LoggingInterceptor(log),
			grpcmw.TimeoutInterceptor(grpcTimeout, log),
			grpcmw.RecoveryInterceptor(log),
			grpcmw.ValidationInterceptor(log),
		),
	)

	// LogServiceServer を登録
	pb.RegisterLogServiceServer(grpcServer, logHandler)
	reflection.Register(grpcServer)

	// bufconn リスナを生成
	listener := bufconn.Listen(bufSize)

	// サーバを別ゴルーチンで起動
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Error("gRPC server encountered an error", err)
		}
	}()

	return grpcServer, listener
}
