// email.go provides a generic EmailService interface with a Resend-backed implementation, falling back to a no-op sender when no API key is configured.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type EmailService interface {
	Send(ctx context.Context, to string, subject string, htmlBody string) error
}

type ResendEmailService struct {
	apiKey     string
	fromEmail  string
	httpClient *http.Client
}

func NewResendEmailService(apiKey string, fromEmail string) *ResendEmailService {
	return &ResendEmailService{
		apiKey:     apiKey,
		fromEmail:  fromEmail,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *ResendEmailService) Send(ctx context.Context, to string, subject string, htmlBody string) error {
	payload := map[string]interface{}{
		"from":    s.fromEmail,
		"to":      []string{to},
		"subject": subject,
		"html":    htmlBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create email request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Dipole/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

type NoopEmailService struct{}

func (s *NoopEmailService) Send(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func NewEmailService(apiKey string, fromEmail string) EmailService {
	if apiKey == "" {
		return &NoopEmailService{}
	}
	return NewResendEmailService(apiKey, fromEmail)
}
