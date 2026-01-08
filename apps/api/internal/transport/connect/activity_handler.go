package connect

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
	transporterrors "github.com/emailapi/api/internal/transport/errors"
)

// ActivityHandler implements the Connect ActivityServiceHandler.
type ActivityHandler struct {
	v1connect.UnimplementedActivityServiceHandler
	svc *service.ActivityService
}

// NewActivityHandler creates a new ActivityHandler.
func NewActivityHandler(svc *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

// ListActivityLogs retrieves activity logs with pagination and optional filters.
func (h *ActivityHandler) ListActivityLogs(
	ctx context.Context,
	req *connect.Request[v1.ListActivityLogsRequest],
) (*connect.Response[v1.ListActivityLogsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
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
		return nil, transporterrors.ToConnectError(err)
	}

	// Convert to proto
	protoLogs := make([]*v1.ActivityLog, len(logs))
	for i, log := range logs {
		protoLogs[i] = &v1.ActivityLog{
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

	return connect.NewResponse(&v1.ListActivityLogsResponse{
		Logs:       protoLogs,
		TotalCount: int32(totalCount),
	}), nil
}
