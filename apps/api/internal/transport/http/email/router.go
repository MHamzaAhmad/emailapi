package email

import (
	"github.com/gin-gonic/gin"

	"github.com/emailapi/api/internal/service"
)

// Handler handles email HTTP requests.
type Handler struct {
	svc *service.EmailService
}

// NewHandler creates a new email handler.
func NewHandler(svc *service.EmailService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers email routes.
func RegisterRoutes(rg *gin.RouterGroup, svc *service.EmailService) {
	h := NewHandler(svc)

	rg.POST("/send", h.Send)
	rg.GET("/emails", h.List)
	rg.GET("/emails/:id", h.GetByID)
}
