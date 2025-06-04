package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	_ "github.com/lib/pq"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	appgrpc "github.com/KeitaShimura/logs-collector-api/internal/app/handler/grpc"
	"github.com/KeitaShimura/logs-collector-api/internal/app/handler/rest"
	grpcmw "github.com/KeitaShimura/logs-collector-api/internal/app/middleware/grpc"
	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/config"
	dbpkg "github.com/KeitaShimura/logs-collector-api/internal/infra/db"
	"github.com/KeitaShimura/logs-collector-api/internal/infra/queue"
	"github.com/KeitaShimura/logs-collector-api/internal/infra/search"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
	"github.com/KeitaShimura/logs-collector-api/internal/pkg/server"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

func main() {
	// アプリケーションの起動処理を実行し、エラーがあれば終了コード1で終了
	if err := run(); err != nil {
		os.Exit(1)
	}
}

// run はアプリケーションの初期化と各種サービスの起動を行います。
func run() error {
	// 暫定ロガーで最低限のログを出力
	tmpLogger := setupLogger("INFO")

	// 設定読み込み（ログレベルを含む）
	cfg, err := config.NewConfig(tmpLogger)
	if err != nil {
		tmpLogger.Error("Failed to load configuration", err)

		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 正式なログレベルで再設定
	log := setupLogger(cfg.LogLevel)
	log.Info("Application startup initiated")

	log.Info("Configuration loaded successfully", "DBHost", cfg.DBHost, "DBPort", cfg.DBPort)

	// PostgreSQLとの接続を確立
	dbConn, err := connectDatabase(cfg, log)
	if err != nil {
		log.Error("Failed to establish database connection", err)

		return fmt.Errorf("failed to establish database connection: %w", err)
	}

	defer dbConn.Close()
	log.Info("Database connection established")

	// SQLBoilerの executor を作成してリポジトリに注入
	exec := boil.ContextExecutor(dbConn)
	logRepo := dbpkg.NewLogRepository(exec, log)

	// NATS Producer を初期化（設定から読み込む）
	natsProducer, err := queue.NewProducer(cfg.NATSURL, log)
	if err != nil {
		log.Error("Failed to initialize NATS Producer", err)

		return fmt.Errorf("failed to initialize NATS Producer: %w", err)
	}

	// Elasticsearch クライアントを指定URLで初期化
	//
	//nolint:exhaustruct // 使用するフィールドのみ明示的に指定
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.ElasticsearchURL},
	}

	esClient, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		log.Error("Failed to initialize Elasticsearch client", err)

		return fmt.Errorf("failed to initialize Elasticsearch client: %w", err)
	}

	// Elasticsearch ラッパーの生成
	esSearcher := search.NewLogSearcher(esClient, log)
	// ユースケース層の初期化
	logUseCase := usecase.NewLogUseCase(logRepo, natsProducer, esSearcher, log)
	// gRPC呼び出しのタイムアウト（設定値[秒]から time.Duration に変換）
	grpcTimeout := time.Duration(cfg.GRPCTimeoutSec) * time.Second
	// RESTサーバーのシャットダウン猶予時間（設定値[秒]から time.Duration に変換）
	shutdownTimeout := time.Duration(cfg.ShutdownTimeoutSec) * time.Second

	// gRPC と REST サーバーを非同期で起動（ポートは cfg から取得）
	go startGRPCServer(logUseCase, log, cfg, grpcTimeout)
	go startRESTServer(logUseCase, log, cfg, shutdownTimeout)

	// 終了シグナル（Ctrl+C, SIGTERM）を受け取るまで待機
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Info("Shutting down both servers gracefully")

	return nil
}

// setupLogger は構造化ロガーを初期化する
//
//nolint:ireturn // クリーンアーキテクチャのため命名を維持
func setupLogger(logLevel string) logger.Logger {
	return logger.NewLogger(
		logger.WithLevel(logger.ParseLevel(logLevel)),
		logger.WithWriter(os.Stdout),
	)
}

// connectDatabase は指定された設定をもとにデータベースへ接続します。
func connectDatabase(cfg *config.Config, log logger.Logger) (*sql.DB, error) {
	hostPort := net.JoinHostPort(cfg.DBHost, cfg.DBPort)

	// PostgreSQL接続文字列（DSN）を構築
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s&TimeZone=%s",
		cfg.DBUser, cfg.DBPassword, hostPort, cfg.DBName, cfg.DBSSLMode, cfg.DBTimeZone,
	)

	// DB接続を試みる
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Error("Database connection failed", err, "dsn", dsn)

		return nil, fmt.Errorf("sql.Open failed: %w", err)
	}

	return dbConn, nil
}

// startGRPCServer はgRPCサーバーを初期化し、指定ポートで起動します。
func startGRPCServer(
	logUseCase usecase.LogUseCase,
	log logger.Logger,
	cfg *config.Config,
	grpcTimeout time.Duration,
) *grpc.Server {
	logHandler := appgrpc.NewLogHandler(logUseCase, log)

	// 各種 gRPC ミドルウェアを適用
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcmw.LoggingInterceptor(log),
			grpcmw.TimeoutInterceptor(grpcTimeout, log),
			grpcmw.RecoveryInterceptor(log),
			grpcmw.ValidationInterceptor(log),
		),
	)

	// サービス登録とリフレクション設定
	pb.RegisterLogServiceServer(grpcServer, logHandler)
	reflection.Register(grpcServer)

	// アドレス構築（例: 0.0.0.0:50051）
	addr := net.JoinHostPort(cfg.GRPCBindAddr, cfg.GRPCPort)

	// ポートバインド（設定に応じて localhost または全体に公開）
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("Failed to bind gRPC server to address", err, "address", addr)
		os.Exit(1)
	}

	// サーバー起動（非同期）
	go func() {
		log.Info("gRPC server started", "address", addr)

		if err := grpcServer.Serve(listener); err != nil {
			log.Error("gRPC server encountered an error", err)
		}
	}()

	return grpcServer
}

// startRESTServer はREST APIサーバーを起動し、SIGINT/SIGTERM を受けて優雅に終了します。
func startRESTServer(
	logUseCase usecase.LogUseCase,
	log logger.Logger,
	cfg *config.Config,
	shutdownTimeout time.Duration,
) {
	logHandler := rest.NewLogHandler(logUseCase, log)

	// Echo サーバーインスタンスの作成
	restTimeout := time.Duration(cfg.RESTTimeoutSec) * time.Second
	echoServer := server.NewRouter(logHandler, log, restTimeout)
	addr := net.JoinHostPort(cfg.RESTBindAddr, cfg.RESTPort)
	log.Info("Starting REST server", "address", addr)

	// Echo サーバーの起動（非同期）
	go func() {
		if err := echoServer.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("REST server failed", err)
		}
	}()

	// シャットダウン処理（SIGINT/SIGTERM 受信時）
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := echoServer.Shutdown(ctx); err != nil {
			log.Error("Failed to shutdown REST server gracefully", err)
		}
	}()
}
