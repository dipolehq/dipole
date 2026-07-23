/*
registration.go handles the proxy startup registration flow: fetch token from
Agent Controller, register with Cloud via gRPC, start periodic health reporting.
Matches the flow: Proxy → Controller /api/token → Cloud Register RPC → Health loop.
*/
package cloudclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type RegistrationResult struct {
	ProxyID   string
	Success   bool
	CACertPEM string
	CAKeyPEM  string
}

func FetchTokenFromController(ctx context.Context, controllerURL string, logger *slog.Logger) (string, error) {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		token, err := doFetchToken(controllerURL)
		if err == nil {
			return token, nil
		}

		logger.Warn("controller unreachable, retrying", "error", err, "backoff", backoff)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func doFetchToken(controllerURL string) (string, error) {
	resp, err := http.Get(controllerURL + "/api/token")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("controller returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Token == "" {
		return "", fmt.Errorf("empty token from controller")
	}
	return result.Token, nil
}

// RunRegistration performs the full proxy registration flow: fetch token from
// controller, register with Cloud gRPC, and start health reporting on success.
// Returns a RegistrationResult indicating whether registration succeeded and the proxy ID.
func RunRegistration(ctx context.Context, controllerURL string, client *CloudClient, passThrough bool, maxWaitSec int, logger *slog.Logger) *RegistrationResult {
	if controllerURL == "" {
		logger.Warn("no DIPOLE_AGENT_CONTROLLER configured, skipping registration")
		return &RegistrationResult{Success: false}
	}

	regCtx := ctx
	if maxWaitSec > 0 {
		var cancel context.CancelFunc
		regCtx, cancel = context.WithTimeout(ctx, time.Duration(maxWaitSec)*time.Second)
		defer cancel()
	}

	token, err := FetchTokenFromController(regCtx, controllerURL, logger)
	if err != nil {
		if !passThrough {
			logger.Error("registration failed: could not obtain token, exiting", "error", err)
			os.Exit(1)
		}
		logger.Warn("registration failed: could not obtain token, continuing in passthrough", "error", err)
		return &RegistrationResult{Success: false}
	}

	logger.Info("obtained registration token from controller")

	result, err := client.Register(regCtx, token, "1.0.0")
	if err != nil {
		if !passThrough {
			logger.Error("registration failed: cloud rejected registration, exiting", "error", err)
			os.Exit(1)
		}
		logger.Warn("registration failed: cloud rejected registration, continuing in passthrough", "error", err)
		return &RegistrationResult{Success: false}
	}

	logger.Info("proxy registered", "proxy_id", result.ProxyID)

	go startHealthReporting(ctx, client, logger)

	return &RegistrationResult{
		ProxyID:   result.ProxyID,
		Success:   true,
		CACertPEM: result.CACertPEM,
		CAKeyPEM:  result.CAKeyPEM,
	}
}

// startHealthReporting sends a health report to Cloud every 60 seconds
// until the context is cancelled.
func startHealthReporting(ctx context.Context, client *CloudClient, logger *slog.Logger) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := client.ReportHealth(ctx, 0); err != nil {
				logger.Warn("health report failed", "error", err)
			} else {
				logger.Debug("health reported")
			}
		}
	}
}
