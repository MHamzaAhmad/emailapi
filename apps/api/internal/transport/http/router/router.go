package router

import (
	"github.com/gin-gonic/gin"

	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/http/email"
	"github.com/emailapi/api/internal/transport/http/log"
	"github.com/emailapi/api/internal/transport/http/middleware"
	"github.com/emailapi/api/internal/transport/http/user"
	"github.com/emailapi/api/internal/transport/http/webhook"
)

// New creates a new Gin router with all routes configured.
func New(svc *service.Service) *gin.Engine {
	r := gin.Default()

	// Global middleware
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := r.Group("/v1")
	{
		// Auth middleware for protected routes
		protected := v1.Group("")
		protected.Use(middleware.Auth(svc.User))

		// Register entity routers
		email.RegisterRoutes(protected, svc.Email)
		webhook.RegisterRoutes(protected, svc.Webhook)
		log.RegisterRoutes(protected)

		// User routes (some public, some protected)
		user.RegisterRoutes(v1, protected, svc.User)
	}

	return r
}
