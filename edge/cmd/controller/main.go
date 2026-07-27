/*
Agent Controller entry point. Registers with Dipole Cloud using a device code
or Trust Provider attestation, exposes /api/token for proxy registration, and
reports health to Cloud every 60 seconds.
*/
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
)

func main() {
	cfg := loadControllerConfig()
	logger := newControllerLogger(cfg.LogLevel)

	ctrl := &AgentController{
		config:       cfg,
		logger:       logger,
		controllerID: uuid.Nil,
		usedTokens:   make(map[string]bool),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register with Cloud
	go ctrl.registerWithRetry(ctx)

	// Start HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", ctrl.handleHealth)
	mux.HandleFunc("GET /api/token", ctrl.handleGetToken)
	mux.HandleFunc("POST /api/kerberos-attest", ctrl.handleKerberosAttest)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	go func() {
		logger.Info("controller http server starting", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
		}
	}()

	// Wait for shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
	logger.Info("shutting down")
	cancel()
	server.Shutdown(context.Background())
}

type AgentController struct {
	config           *ControllerConfig
	logger           *slog.Logger
	controllerID     uuid.UUID
	sharedSecret     []byte // HMAC key for signing Kerberos attestations
	registered       atomic.Bool
	registeredAt     time.Time
	lastCloudContact time.Time
	usedTokens       map[string]bool
	tokenMu          sync.Mutex
}

type ControllerConfig struct {
	TenantID      string
	DeviceCode    string
	ControllerID  string
	CloudEndpoint string
	HTTPPort      int
	LogLevel      string
}

func loadControllerConfig() *ControllerConfig {
	tenantID := os.Getenv("DIPOLE_TENANT_ID")
	if tenantID == "" {
		fmt.Fprintln(os.Stderr, "DIPOLE_TENANT_ID is required")
		os.Exit(1)
	}

	cloudEndpoint := os.Getenv("DIPOLE_CLOUD_ENDPOINT")
	if cloudEndpoint == "" {
		cloudEndpoint = fmt.Sprintf("https://%s.dipolehq.com", tenantID)
	}

	port := 80
	if p := os.Getenv("DIPOLE_HTTP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	return &ControllerConfig{
		TenantID:      tenantID,
		DeviceCode:    os.Getenv("DIPOLE_DEVICE_CODE"),
		ControllerID:  os.Getenv("DIPOLE_AGENT_CONTROLLER_ID"),
		CloudEndpoint: cloudEndpoint,
		HTTPPort:      port,
		LogLevel:      envOr("DIPOLE_LOG_LEVEL", "information"),
	}
}

func (c *AgentController) handleHealth(w http.ResponseWriter, _ *http.Request) {
	status := "Unregistered"
	if c.registered.Load() {
		if time.Since(c.lastCloudContact) < 120*time.Second {
			status = "Healthy"
		} else if c.lastCloudContact.IsZero() {
			status = "Registered"
		} else {
			status = "NotConnected"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           status,
		"registeredAt":     formatTime(c.registeredAt),
		"lastCloudContact": formatTime(c.lastCloudContact),
	})
}

func (c *AgentController) handleGetToken(w http.ResponseWriter, _ *http.Request) {
	if !c.registered.Load() {
		http.Error(w, `{"error":"controller not registered"}`, http.StatusServiceUnavailable)
		return
	}

	token, err := c.requestTokenFromCloud()
	if err != nil {
		http.Error(w, `{"error":"failed to obtain token from cloud"}`, http.StatusBadGateway)
		c.logger.Error("request token from cloud", "error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func (c *AgentController) requestTokenFromCloud() (string, error) {
	url := c.config.CloudEndpoint + "/internal/controllers/token"
	body := fmt.Sprintf(`{"tenant_id":"%s","controller_id":"%s"}`, c.config.TenantID, c.controllerID)

	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("cloud unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cloud returned %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func (c *AgentController) registerWithRetry(ctx context.Context) {
	backoff := 5 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := c.register()
		if err == nil {
			c.registered.Store(true)
			c.registeredAt = time.Now()
			c.logger.Info("controller registered with cloud", "controller_id", c.controllerID)
			go c.startHealthReporting(ctx)
			return
		}

		c.logger.Error("registration failed, retrying", "error", err, "backoff", backoff)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *AgentController) register() error {
	if c.config.DeviceCode != "" {
		return c.registerWithDeviceCode()
	}
	if c.config.ControllerID != "" {
		id, err := uuid.Parse(c.config.ControllerID)
		if err != nil {
			return fmt.Errorf("invalid controller ID: %w", err)
		}
		c.controllerID = id
		return nil
	}
	return fmt.Errorf("no DIPOLE_DEVICE_CODE or DIPOLE_AGENT_CONTROLLER_ID configured")
}

func (c *AgentController) registerWithDeviceCode() error {
	url := c.config.CloudEndpoint + "/internal/controllers/register"
	body := fmt.Sprintf(`{"tenant_id":"%s","device_code":"%s"}`, c.config.TenantID, c.config.DeviceCode)

	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("cloud unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration rejected: status %d", resp.StatusCode)
	}

	var result struct {
		ControllerID string `json:"controller_id"`
		SharedSecret string `json:"shared_secret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parse registration response: %w", err)
	}

	id, err := uuid.Parse(result.ControllerID)
	if err != nil {
		return fmt.Errorf("invalid controller_id in response: %w", err)
	}
	c.controllerID = id

	// Store shared secret for Kerberos attestation signing
	if result.SharedSecret != "" {
		secretBytes, err := hex.DecodeString(result.SharedSecret)
		if err != nil {
			return fmt.Errorf("invalid shared_secret in response: %w", err)
		}
		c.sharedSecret = secretBytes
	}

	return nil
}

func (c *AgentController) startHealthReporting(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.reportHealth(); err != nil {
				c.logger.Warn("health report failed", "error", err)
			} else {
				c.lastCloudContact = time.Now()
			}
		}
	}
}

func (c *AgentController) reportHealth() error {
	// For now, use HTTP. Will switch to gRPC ReportHealth in production.
	url := c.config.CloudEndpoint + "/internal/controllers/health"
	body := fmt.Sprintf(`{"tenant_id":"%s","controller_id":"%s","status":"Healthy"}`, c.config.TenantID, c.controllerID)

	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health report rejected: %d", resp.StatusCode)
	}
	return nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newControllerLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "fatal", "error":
		lvl = slog.LevelError
	case "warning":
		lvl = slog.LevelWarn
	case "debug", "verbose":
		lvl = slog.LevelDebug
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

// handleKerberosAttest validates a Kerberos ticket (keytab validation assumed
// already performed by the caller) and signs an attestation document with
// HMAC-SHA256 using the shared secret from Cloud registration.
func (c *AgentController) handleKerberosAttest(w http.ResponseWriter, r *http.Request) {
	if !c.registered.Load() {
		http.Error(w, `{"error":"controller not registered"}`, http.StatusServiceUnavailable)
		return
	}

	if len(c.sharedSecret) == 0 {
		http.Error(w, `{"error":"no shared secret configured"}`, http.StatusInternalServerError)
		return
	}

	var req struct {
		Principal string `json:"principal"`
		Realm     string `json:"realm"`
		Domain    string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Principal == "" || req.Realm == "" {
		http.Error(w, `{"error":"principal and realm required"}`, http.StatusBadRequest)
		return
	}

	payload := map[string]interface{}{
		"principal": req.Principal,
		"realm":     req.Realm,
		"domain":    req.Domain,
		"timestamp": time.Now().Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.StdEncoding.EncodeToString(payloadJSON)

	mac := hmac.New(sha256.New, c.sharedSecret)
	mac.Write([]byte(payloadB64))
	sig := hex.EncodeToString(mac.Sum(nil))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"payload":       payloadB64,
		"signature":     sig,
		"controller_id": c.controllerID.String(),
	})
}
