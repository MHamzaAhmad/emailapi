package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/transport/http/middleware"
)

// Create handles POST /v1/users
func (h *Handler) Create(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	user, apiKey, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":    user,
		"api_key": apiKey,
		"message": "Store this API key securely. It will not be shown again.",
	})
}

// GetMe handles GET /v1/users/me
func (h *Handler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// RegenerateAPIKey handles POST /v1/users/me/api-key
func (h *Handler) RegenerateAPIKey(c *gin.Context) {
	userID := middleware.GetUserID(c)

	resp, err := h.svc.RegenerateAPIKey(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key": resp.APIKey,
		"message": "Store this API key securely. It will not be shown again.",
	})
}
