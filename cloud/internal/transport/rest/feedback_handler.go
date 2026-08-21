/*
Routes:

	POST /api/v1/feedback
*/
package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type FeedbackHandler struct {
	apiKey       string
	fromEmail    string
	supportEmail string
}

func NewFeedbackHandler(apiKey string, fromEmail string, supportEmail string) *FeedbackHandler {
	return &FeedbackHandler{apiKey: apiKey, fromEmail: fromEmail, supportEmail: supportEmail}
}

func (h *FeedbackHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/feedback", h.HandleSubmitFeedback)
}

func (h *FeedbackHandler) HandleSubmitFeedback(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	userEmail := UserEmailFromContext(r.Context())

	var req struct {
		Message string `json:"message"`
		Image   string `json:"image"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, "invalid request body")
		return
	}

	if req.Message == "" {
		respondBadRequest(w, "message is required")
		return
	}

	subject := fmt.Sprintf("Feedback from %s (tenant %s)", userEmail, tenantID.String()[:8])
	html := buildFeedbackEmail(userEmail, tenantID.String(), req.Message, req.Image != "")

	payload := map[string]interface{}{
		"from":    h.fromEmail,
		"to":      []string{h.supportEmail},
		"subject": subject,
		"html":    html,
	}

	if req.Image != "" {

		content := req.Image
		if idx := strings.Index(content, ","); idx != -1 {
			content = content[idx+1:]
		}
		payload["attachments"] = []map[string]string{
			{
				"content":    content,
				"filename":   "screenshot.png",
				"content_id": "screenshot",
			},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("feedback marshal failed", "error", err)
		respondError(w, http.StatusInternalServerError, "email_failed", "failed to send feedback")
		return
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		slog.Error("feedback request creation failed", "error", err)
		respondError(w, http.StatusInternalServerError, "email_failed", "failed to send feedback")
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+h.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		slog.Error("feedback email send failed", "error", err)
		respondError(w, http.StatusInternalServerError, "email_failed", "failed to send feedback")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("feedback resend API error", "status", resp.StatusCode, "body", string(respBody))
		respondError(w, http.StatusInternalServerError, "email_failed", "failed to send feedback")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func buildFeedbackEmail(userEmail, tenantID, message string, hasImage bool) string {
	html := `<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; max-width: 600px;">` +
		`<h2 style="color: #1a1a1a; font-size: 18px; margin-bottom: 16px;">User Feedback</h2>` +
		`<table style="font-size: 14px; color: #4a4a4a; margin-bottom: 16px;">` +
		`<tr><td style="padding: 4px 12px 4px 0; font-weight: 600;">From:</td><td>` + userEmail + `</td></tr>` +
		`<tr><td style="padding: 4px 12px 4px 0; font-weight: 600;">Tenant:</td><td style="font-family: monospace; font-size: 12px;">` + tenantID + `</td></tr>` +
		`</table>` +
		`<div style="background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; font-size: 14px; color: #374151; white-space: pre-wrap;">` + message + `</div>`

	if hasImage {
		html += `<div style="margin-top: 16px;"><p style="font-size: 12px; color: #6b7280; margin-bottom: 8px;">Attached screenshot:</p>` +
			`<img src="cid:screenshot" style="max-width: 100%; border-radius: 8px; border: 1px solid #e5e7eb;" /></div>`
	}

	html += `</div>`
	return html
}
