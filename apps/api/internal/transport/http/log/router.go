package log

import (
	"github.com/gin-gonic/gin"
)

// Handler handles log HTTP requests.
type Handler struct {
	// In production, this would have a LogService
}

// NewHandler creates a new log handler.
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes registers log routes.
func RegisterRoutes(rg *gin.RouterGroup) {
	h := NewHandler()

	logs := rg.Group("/logs")
	{
		logs.GET("", h.Query)
		logs.GET("/emails/:email_id", h.GetByEmailID)
	}
}
