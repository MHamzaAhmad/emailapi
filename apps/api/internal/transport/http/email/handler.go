package email

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/transport/http/middleware"
)

// Send handles POST /v1/send
func (h *Handler) Send(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req domain.SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	resp, err := h.svc.Send(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

// GetByID handles GET /v1/emails/:id
func (h *Handler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	emailID := c.Param("id")

	email, err := h.svc.GetByID(c.Request.Context(), userID, emailID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "email not found",
		})
		return
	}

	c.JSON(http.StatusOK, email)
}

// List handles GET /v1/emails
func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	emails, err := h.svc.List(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   emails,
		"limit":  limit,
		"offset": offset,
	})
}
