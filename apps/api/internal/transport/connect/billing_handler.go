package connect

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/domain"

	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
	transporterrors "github.com/emailapi/api/internal/transport/errors"

)

type billingHandler struct {
	billingService *service.BillingService
}

func NewBillingHandler(billingService *service.BillingService) v1connect.BillingServiceHandler {
	return &billingHandler{
		billingService: billingService,
	}
}

func (h *billingHandler) GetSubscription(ctx context.Context, req *connect.Request[pb.GetSubscriptionRequest]) (*connect.Response[pb.GetSubscriptionResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	info, err := h.billingService.GetSubscriptionInfo(ctx, userID)
	if err != nil {
		return nil, transporterrors.ToConnectError(err)
	}

	return connect.NewResponse(&pb.GetSubscriptionResponse{
		HasSubscription: info.HasSubscription,
		IsPaid:          info.IsPaid,
		PlanId:          info.PlanID,
		SubscriptionId:  info.SubscriptionID,
		PolarCustomerId: info.PolarCustomerID,
	}), nil
}

func (h *billingHandler) GetPlans(ctx context.Context, req *connect.Request[pb.GetPlansRequest]) (*connect.Response[pb.GetPlansResponse], error) {
	plans := h.billingService.GetPlans()

	pbPlans := make([]*pb.Plan, len(plans))
	for i, p := range plans {
		pbFeatures := make([]*pb.Feature, len(p.Features))
		for j, f := range p.Features {
			pbFeatures[j] = &pb.Feature{
				Name:    f.Name,
				Tooltip: f.Tooltip,
			}
		}

		pbPlans[i] = &pb.Plan{
			Id:                p.ID,
			Name:              p.Name,
			Description:       p.Description,
			MonthlyLimit:      p.MonthlyLimit,
			DailyLimit:        p.DailyLimit,
			PriceCents:        p.PriceCents,
			OveragePriceCents: p.OveragePriceCents,
			Features:          pbFeatures,
		}
	}

	return connect.NewResponse(&pb.GetPlansResponse{
		Plans: pbPlans,
	}), nil
}

func (h *billingHandler) CreateCheckoutSession(ctx context.Context, req *connect.Request[pb.CreateCheckoutSessionRequest]) (*connect.Response[pb.CreateCheckoutSessionResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	checkoutURL, err := h.billingService.CreateCheckoutSession(ctx, userID, req.Msg.PlanId, req.Msg.SuccessUrl)
	if err != nil {
		return nil, transporterrors.ToConnectError(err)
	}

	return connect.NewResponse(&pb.CreateCheckoutSessionResponse{
		CheckoutUrl: checkoutURL,
	}), nil
}

func (h *billingHandler) GetCustomerPortalUrl(ctx context.Context, req *connect.Request[pb.GetCustomerPortalUrlRequest]) (*connect.Response[pb.GetCustomerPortalUrlResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	portalURL, err := h.billingService.GetCustomerPortalUrl(ctx, userID)
	if err != nil {
		return nil, transporterrors.ToConnectError(err)
	}

	return connect.NewResponse(&pb.GetCustomerPortalUrlResponse{
		PortalUrl: portalURL,
	}), nil
}
