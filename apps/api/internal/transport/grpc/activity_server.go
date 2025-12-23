package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/middleware"
	"github.com/emailapi/api/internal/service"
)

// ActivityServer implements the ActivityService gRPC server.
type ActivityServer struct {
	emailapiv1.UnimplementedActivityServiceServer
	svc *service.ActivityService
}

// NewActivityServer creates a new ActivityServer.
func NewActivityServer(svc *service.ActivityService) *ActivityServer {
	return &ActivityServer{svc: svc}
}

// ListActivityLogs retrieves activity logs with pagination and optional filters.
func (s *ActivityServer) ListActivityLogs(ctx context.Context, req *emailapiv1.ListActivityLogsRequest) (*emailapiv1.ListActivityLogsResponse, error) {
	// Get user ID from authenticated context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Build filters from request
	filters := service.ListFilters{}
	if req.EntityType != nil {
		filters.EntityType = req.EntityType
	}
	if req.Action != nil {
		filters.Action = req.Action
	}
	if req.StartTime != nil {
		filters.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		filters.EndTime = req.EndTime
	}

	// Call service
	logs, totalCount, err := s.svc.List(ctx, userID, filters, int(req.PageSize), int(req.Offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list activity logs: %v", err)
	}

	// Convert to proto
	protoLogs := make([]*emailapiv1.ActivityLog, len(logs))
	for i, log := range logs {
		protoLogs[i] = &emailapiv1.ActivityLog{
			Id:         log.ID,
			UserId:     log.UserID,
			EntityType: log.EntityType,
			EntityId:   log.EntityID,
			Action:     log.Action,
			Status:     log.Status,
			Details:    log.Details,
			Metadata:   "",
			Timestamp:  timestamppb.New(time.UnixMilli(log.Timestamp)),
		}
	}

	return &emailapiv1.ListActivityLogsResponse{
		Logs:       protoLogs,
		TotalCount: int32(totalCount),
	}, nil
}
