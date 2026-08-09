/* Registration logic: device-code and trust-provider flows, plus health reporting. */
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
		return c.registerWithTrustProvider()
	}
	return fmt.Errorf("no DIPOLE_DEVICE_CODE or DIPOLE_AGENT_CONTROLLER_ID configured")
}

func (c *AgentController) registerWithTrustProvider() error {
	id, err := uuid.Parse(c.config.ControllerID)
	if err != nil {
		return fmt.Errorf("invalid controller ID: %w", err)
	}

	evidence := map[string]interface{}{
		"tenant_id":     c.config.TenantID,
		"controller_id": id.String(),
	}

	hostname, _ := os.Hostname()
	evidence["hostname"] = hostname

	if k8sToken, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		evidence["k8s_service_account_token"] = string(k8sToken)
		evidence["k8s_namespace"] = os.Getenv("K8S_NAMESPACE")
	}

	body, _ := json.Marshal(evidence)
	url := c.config.CloudEndpoint + "/internal/controllers/register-attestation"

	resp, err := c.cloudClient.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("cloud unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("attestation registration rejected: status %d", resp.StatusCode)
	}

	var result struct {
		ControllerID string `json:"controller_id"`
		SharedSecret string `json:"shared_secret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parse attestation response: %w", err)
	}

	c.controllerID = id

	if result.SharedSecret != "" {
		secretBytes, err := hex.DecodeString(result.SharedSecret)
		if err != nil {
			return fmt.Errorf("invalid shared_secret in response: %w", err)
		}
		c.sharedSecret = secretBytes
	}

	return nil
}

func (c *AgentController) registerWithDeviceCode() error {
	url := c.config.CloudEndpoint + "/internal/controllers/register"
	body := fmt.Sprintf(`{"tenant_id":"%s","device_code":"%s"}`, c.config.TenantID, c.config.DeviceCode)

	resp, err := c.cloudClient.Post(url, "application/json", strings.NewReader(body))
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
	url := c.config.CloudEndpoint + "/internal/controllers/health"
	body := fmt.Sprintf(`{"tenant_id":"%s","controller_id":"%s","status":"Healthy"}`, c.config.TenantID, c.controllerID)

	resp, err := c.cloudClient.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health report rejected: %d", resp.StatusCode)
	}
	return nil
}
