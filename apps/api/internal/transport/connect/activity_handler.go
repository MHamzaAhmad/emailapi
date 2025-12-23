package connect

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/emailapiv1connect"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
)

// ActivityHandler implements the Connect ActivityServiceHandler.
type ActivityHandler struct {
	emailapiv1connect.UnimplementedActivityServiceHandler
	svc *service.ActivityService
}

// NewActivityHandler creates a new ActivityHandler.
func NewActivityHandler(svc *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

// ListActivityLogs retrieves activity logs with pagination and optional filters.
func (h *ActivityHandler) ListActivityLogs(
	ctx context.Context,
	req *connect.Request[emailapiv1.ListActivityLogsRequest],
) (*connect.Response[emailapiv1.ListActivityLogsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	// Build filters from request
	filters := service.ListFilters{}
	if req.Msg.EntityType != nil {
		filters.EntityType = req.Msg.EntityType
	}
	if req.Msg.Action != nil {
		filters.Action = req.Msg.Action
	}
	if req.Msg.StartTime != nil {
		filters.StartTime = req.Msg.StartTime
	}
	if req.Msg.EndTime != nil {
		filters.EndTime = req.Msg.EndTime
	}

	// Call service
	logs, totalCount, err := h.svc.List(ctx, userID, filters, int(req.Msg.PageSize), int(req.Msg.Offset))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
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

	return connect.NewResponse(&emailapiv1.ListActivityLogsResponse{
		Logs:       protoLogs,
		TotalCount: int32(totalCount),
	}), nil
}
