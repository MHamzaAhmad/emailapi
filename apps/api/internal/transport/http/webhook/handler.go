package webhook

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/transport/http/middleware"
)

// Create handles POST /v1/webhooks
func (h *Handler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req domain.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	webhook, secret, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"webhook": webhook,
		"secret":  secret,
		"message": "Store this secret securely. It will not be shown again.",
	})
}

// GetByID handles GET /v1/webhooks/:id
func (h *Handler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	webhookID := c.Param("id")

	webhook, err := h.svc.GetByID(c.Request.Context(), userID, webhookID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "webhook not found",
		})
		return
	}

	c.JSON(http.StatusOK, webhook)
}

// List handles GET /v1/webhooks
func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	webhooks, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": webhooks,
	})
}

// Update handles PATCH /v1/webhooks/:id
func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	webhookID := c.Param("id")

	var req domain.UpdateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	webhook, err := h.svc.Update(c.Request.Context(), userID, webhookID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, webhook)
}

// Delete handles DELETE /v1/webhooks/:id
func (h *Handler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	webhookID := c.Param("id")

	if err := h.svc.Delete(c.Request.Context(), userID, webhookID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
