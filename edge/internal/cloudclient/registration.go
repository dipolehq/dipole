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

func RunRegistration(ctx context.Context, controllerURL string, client *CloudClient, passThrough bool, maxWaitSec int, activeConns func() int32, logger *slog.Logger) *RegistrationResult {
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

	go startHealthReporting(ctx, client, activeConns, logger)
	go startSessionTokenRefresh(ctx, client, logger)

	return &RegistrationResult{
		ProxyID:   result.ProxyID,
		Success:   true,
		CACertPEM: result.CACertPEM,
	}
}

func startHealthReporting(ctx context.Context, client *CloudClient, activeConns func() int32, logger *slog.Logger) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := client.ReportHealth(ctx, activeConns()); err != nil {
				logger.Warn("health report failed", "error", err)
			} else {
				logger.Debug("health reported")
			}
		}
	}
}

// sessionTokenRefreshFraction is how far into the current session token's
// lifetime the proxy proactively renews it (0.5 = halfway). Refreshing
// well before expiry, rather than waiting until close to it, leaves ample
// margin to retry with backoff (see startSessionTokenRefresh) without ever
// running out the clock on the token still in use.
const sessionTokenRefreshFraction = 0.5

// startSessionTokenRefresh proactively renews the proxy's session token
// before it expires, for as long as ctx is live. It schedules the next
// refresh attempt at sessionTokenRefreshFraction of the current token's
// remaining lifetime; on failure it retries with exponential backoff
// (capped) rather than waiting for the next full-lifetime cycle, since a
// token nearing expiry that fails to refresh must be retried promptly, not
// left until the next scheduled check.
func startSessionTokenRefresh(ctx context.Context, client *CloudClient, logger *slog.Logger) {
	const (
		minRetryBackoff = 5 * time.Second
		maxRetryBackoff = 60 * time.Second
	)

	timer := time.NewTimer(nextRefreshDelay(client.SessionTokenExpiresAt()))
	defer timer.Stop()

	backoff := minRetryBackoff
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		expiresAt, err := client.RefreshSessionToken(ctx)
		if err != nil {
			logger.Warn("session token refresh failed, retrying with backoff", "error", err, "backoff", backoff)
			timer.Reset(backoff)
			backoff *= 2
			if backoff > maxRetryBackoff {
				backoff = maxRetryBackoff
			}
			continue
		}

		logger.Info("session token refreshed", "expires_at", expiresAt)
		backoff = minRetryBackoff
		timer.Reset(nextRefreshDelay(expiresAt))
	}
}

// nextRefreshDelay returns how long to wait before the next refresh
// attempt, given when the current token expires. Refreshing at
// sessionTokenRefreshFraction of the remaining lifetime rather than at a
// fixed interval means the delay self-corrects to whatever lifetime Cloud
// actually issued, without either side hardcoding the other's assumption
// about token lifetime.
func nextRefreshDelay(expiresAt time.Time) time.Duration {
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return 0
	}
	return time.Duration(float64(remaining) * sessionTokenRefreshFraction)
}
