package webhook

import (
	"github.com/gin-gonic/gin"

	"github.com/emailapi/api/internal/service"
)

// Handler handles webhook HTTP requests.
type Handler struct {
	svc *service.WebhookService
}

// NewHandler creates a new webhook handler.
func NewHandler(svc *service.WebhookService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers webhook routes.
func RegisterRoutes(rg *gin.RouterGroup, svc *service.WebhookService) {
	h := NewHandler(svc)

	webhooks := rg.Group("/webhooks")
	{
		webhooks.POST("", h.Create)
		webhooks.GET("", h.List)
		webhooks.GET("/:id", h.GetByID)
		webhooks.PATCH("/:id", h.Update)
		webhooks.DELETE("/:id", h.Delete)
	}
}
