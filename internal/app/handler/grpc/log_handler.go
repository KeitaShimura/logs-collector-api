package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/KeitaShimura/logs-collector-api/internal/app/usecase"
	"github.com/KeitaShimura/logs-collector-api/internal/domain/model"
	"github.com/KeitaShimura/logs-collector-api/internal/logger"
	pb "github.com/KeitaShimura/logs-collector-protos/go/logs/v1"
)

// LogHandler はログ関連のgRPCリクエストを処理するハンドラー構造体
type LogHandler struct {
	pb.UnimplementedLogServiceServer
	logUseCase usecase.LogUseCase
	logger     logger.Logger
}

// NewLogHandler は新しいLogHandlerインスタンスを作成する
func NewLogHandler(uc usecase.LogUseCase, logger logger.Logger) *LogHandler {
	return &LogHandler{
		UnimplementedLogServiceServer: pb.UnimplementedLogServiceServer{},
		logUseCase:                    uc,
		logger:                        logger,
	}
}

// SendLog はログ保存リクエストを受け付け、処理するgRPCメソッド
func (h *LogHandler) SendLog(ctx context.Context, req *pb.SendLogRequest) (*pb.SendLogResponse, error) {
	// ID 補完（未指定場合はサーバー側で生成）
	logID := req.GetLog().GetId()
	if logID == "" {
		logID = uuid.NewString()
	}

	// Metadata 補完（未指定場合は空mapで初期化）
	metadata := req.GetLog().GetMetadata()
	if metadata == nil {
		metadata = make(map[string]string)
	}

	// gRPCリクエストをドメインモデルに変換
	log := model.Log{
		ID:        logID,
		TraceID:   req.GetLog().GetTraceId(),
		Timestamp: req.GetLog().GetTimestamp().AsTime(),
		Level:     req.GetLog().GetLevel(),
		Service:   req.GetLog().GetService(),
		Message:   req.GetLog().GetMessage(),
		Metadata:  metadata,
	}

	// ユースケース層にログ保存を依頼
	if err := h.logUseCase.SendLog(ctx, &log); err != nil {
		// 保存失敗ログ
		h.logger.Error("Failed to save log", err,
			"ID", log.ID,
			"TraceID", log.TraceID,
			"Timestamp", log.Timestamp,
			"Level", log.Level,
			"Service", log.Service,
			"Message", log.Message,
			"Metadata", log.Metadata,
		)

		// 失敗レスポンスを返却（クライアントにもエラーを通知）
		grpcCode := AppErrorToGRPCCode(err)

		return nil, status.Errorf(grpcCode, "failed to save log: %v", err)
	}

	// 保存成功ログ
	h.logger.Info("Log saved successfully",
		"ID", log.ID,
		"TraceID", log.TraceID,
		"Timestamp", log.Timestamp,
		"Level", log.Level,
		"Service", log.Service,
		"Message", log.Message,
		"Metadata", log.Metadata,
	)

	// 成功レスポンスを返却
	return &pb.SendLogResponse{
		Success:      true,
		ErrorMessage: nil,
	}, nil
}

// GetLogs は指定された条件でログを取得するgRPCメソッド
func (h *LogHandler) GetLogs(ctx context.Context, req *pb.GetLogsRequest) (*pb.GetLogsResponse, error) {
	// ユースケース層から条件に基づきログを取得
	logs, err := h.logUseCase.GetLogs(
		ctx,
		req.GetService(),
		req.GetLevel(),
		int(req.GetLimit()),
		int(req.GetOffset()),
	)
	if err != nil {
		// 取得失敗ログ
		h.logger.Error("Failed to get logs", err,
			"Service", req.GetService(),
			"Level", req.GetLevel(),
			"Limit", req.GetLimit(),
			"Offset", req.GetOffset(),
		)

		// gRPCエラーとして返却
		grpcCode := AppErrorToGRPCCode(err)

		return nil, status.Errorf(grpcCode, "failed to get logs: %v", err)
	}

	// ドメインモデルをgRPCレスポンス形式に変換
	pbLogs := make([]*pb.Log, 0, len(logs))
	for _, log := range logs {
		pbLogs = append(pbLogs, &pb.Log{
			Id:        log.ID,
			TraceId:   log.TraceID,
			Timestamp: timestamppb.New(log.Timestamp),
			Level:     log.Level,
			Service:   log.Service,
			Message:   log.Message,
			Metadata:  log.Metadata,
		})
	}

	// 成功ログ（総件数含む）
	h.logger.Info("Logs retrieved successfully",
		"Count", len(pbLogs),
		"Service", req.GetService(),
		"Level", req.GetLevel(),
		"Limit", req.GetLimit(),
		"Offset", req.GetOffset(),
	)

	// レスポンス返却
	return &pb.GetLogsResponse{Logs: pbLogs}, nil
}

// AppErrorToGRPCCode はアプリケーションエラーを gRPC ステータスコードに変換する
func AppErrorToGRPCCode(err error) codes.Code {
	switch {
	case errors.Is(err, usecase.ErrValidationFailure):
		return codes.InvalidArgument
	case errors.Is(err, usecase.ErrRepositoryFailure):
		return codes.Internal
	case errors.Is(err, usecase.ErrNoLogsFound):
		return codes.NotFound
	// 必要に応じて追加
	default:
		return codes.Unknown
	}
}
