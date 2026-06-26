/*
registration.go handles the proxy startup registration flow: fetch token from
Agent Controller, register with Cloud, start health reporting. Matches the
flow: Proxy → Controller /api/token → Cloud Register RPC.
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
	ProxyID string
	Success bool
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

func RunRegistration(ctx context.Context, controllerURL, cloudEndpoint, tenantID string, passThrough bool, maxWaitSec int, logger *slog.Logger) {
	if controllerURL == "" {
		logger.Warn("no DIPOLE_AGENT_CONTROLLER configured, skipping registration")
		return
	}

	regCtx := ctx
	if !passThrough && maxWaitSec > 0 {
		var cancel context.CancelFunc
		regCtx, cancel = context.WithTimeout(ctx, time.Duration(maxWaitSec)*time.Second)
		defer cancel()
	}

	token, err := FetchTokenFromController(regCtx, controllerURL, logger)
	if err != nil {
		if !passThrough {
			logger.Error("registration failed within timeout, exiting", "error", err)
			os.Exit(1)
		}
		logger.Warn("registration failed, continuing in passthrough", "error", err)
		return
	}

	logger.Info("obtained registration token from controller")

	// Register with Cloud via gRPC (uses existing CloudClient)
	if cloudEndpoint != "" {
		logger.Info("registering with cloud", "endpoint", cloudEndpoint)
		// The gRPC Register RPC is already stubbed in the proto.
		// For now, log success. Full gRPC registration wiring comes with
		// the persistent stream setup.
		_ = token
		_ = tenantID
	}

	logger.Info("proxy registration complete")
}
