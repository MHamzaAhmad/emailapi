package http

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
)

//go:embed templates/*.html
var templateFS embed.FS

// UnsubscribeHandler handles HTTP requests for unsubscribe pages.
type UnsubscribeHandler struct {
	svc       *service.UnsubscribeService
	templates *template.Template
}

// NewUnsubscribeHandler creates a new UnsubscribeHandler.
func NewUnsubscribeHandler(svc *service.UnsubscribeService) (*UnsubscribeHandler, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &UnsubscribeHandler{
		svc:       svc,
		templates: tmpl,
	}, nil
}

// PageData holds data for template rendering.
type PageData struct {
	Email      string
	SenderName string
	Error      string
	Success    bool
	Token      string
}

// Handle routes requests to the appropriate handler based on method.
func (h *UnsubscribeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGet shows the unsubscribe confirmation page.
func (h *UnsubscribeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderError(w, "Missing unsubscribe token", http.StatusBadRequest)
		return
	}

	// Decode token to get email for display
	tokenData, err := h.svc.TokenService().Decode(token)
	if err != nil {
		if err == service.ErrExpiredToken {
			h.renderError(w, "This unsubscribe link has expired. Please contact the sender directly.", http.StatusGone)
			return
		}
		h.renderError(w, "Invalid unsubscribe link", http.StatusBadRequest)
		return
	}

	// Render confirmation page
	data := PageData{
		Email: maskEmail(tokenData.Email),
		Token: token,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "unsubscribe.html", data); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// handlePost processes the unsubscribe request.
func (h *UnsubscribeHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if token == "" {
		h.renderError(w, "Missing unsubscribe token", http.StatusBadRequest)
		return
	}

	// Process unsubscribe
	tokenData, err := h.svc.ProcessUnsubscribe(r.Context(), token, domain.UnsubscribeSourceLink)
	if err != nil {
		if err == service.ErrExpiredToken {
			h.renderError(w, "This unsubscribe link has expired.", http.StatusGone)
			return
		}
		if err == service.ErrInvalidToken {
			h.renderError(w, "Invalid unsubscribe link", http.StatusBadRequest)
			return
		}
		h.renderError(w, "Failed to process unsubscribe", http.StatusInternalServerError)
		return
	}

	// Render success page
	data := PageData{
		Email:   maskEmail(tokenData.Email),
		Success: true,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "success.html", data); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// HandleOneClick handles RFC 8058 one-click unsubscribe (from email client).
// Expects POST with List-Unsubscribe=One-Click header.
func (h *UnsubscribeHandler) HandleOneClick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.FormValue("token")
	}
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	// Process unsubscribe (no confirmation needed for one-click)
	_, err := h.svc.ProcessUnsubscribe(r.Context(), token, domain.UnsubscribeSourceOneClick)
	if err != nil {
		if err == service.ErrExpiredToken || err == service.ErrInvalidToken {
			http.Error(w, "Invalid or expired token", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to unsubscribe", http.StatusInternalServerError)
		return
	}

	// Return 200 OK (RFC 8058 requirement)
	w.WriteHeader(http.StatusOK)
}

// renderError renders the error template.
func (h *UnsubscribeHandler) renderError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	data := PageData{Error: message}
	if err := h.templates.ExecuteTemplate(w, "error.html", data); err != nil {
		http.Error(w, message, statusCode)
	}
}

// maskEmail masks part of the email for privacy (e.g., "jo***@example.com").
func maskEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "***"
	}

	// Show first 2 chars, mask rest before @
	if at <= 2 {
		return email[:at] + "@" + email[at+1:]
	}

	return email[:2] + "***" + email[at:]
}
