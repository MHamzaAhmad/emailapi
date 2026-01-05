package connect

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
)

type billingHandler struct {
	billingService *service.BillingService
}

func NewBillingHandler(billingService *service.BillingService) v1connect.BillingServiceHandler {
	return &billingHandler{
		billingService: billingService,
	}
}

func (h *billingHandler) SyncSubscription(ctx context.Context, req *connect.Request[pb.SyncSubscriptionRequest]) (*connect.Response[pb.SyncSubscriptionResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	user, err := h.billingService.SyncSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	polarCustomerID := ""
	if user.PolarCustomerID != nil {
		polarCustomerID = *user.PolarCustomerID
	}

	return connect.NewResponse(&pb.SyncSubscriptionResponse{
		UserId:          user.ID,
		Plan:            string(user.Plan),
		PolarCustomerId: polarCustomerID,
	}), nil
}
