package user

import (
	"github.com/gin-gonic/gin"

	"github.com/emailapi/api/internal/service"
)

// Handler handles user HTTP requests.
type Handler struct {
	svc *service.UserService
}

// NewHandler creates a new user handler.
func NewHandler(svc *service.UserService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers user routes.
func RegisterRoutes(public *gin.RouterGroup, protected *gin.RouterGroup, svc *service.UserService) {
	h := NewHandler(svc)

	// Public routes (no auth required)
	public.POST("/users", h.Create)

	// Protected routes
	protected.GET("/users/me", h.GetMe)
	protected.POST("/users/me/api-key", h.RegenerateAPIKey)
}
