// handlers.go implements HTTP handlers for health, token issuance, and Kerberos attestation, plus HMAC-signed outbound requests to Cloud's /internal/controllers/* endpoints.
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const controllerSignatureHeader = "X-Dipole-Controller-Signature"

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
	body := []byte(fmt.Sprintf(`{"tenant_id":"%s","controller_id":"%s"}`, c.config.TenantID, c.controllerID))

	resp, err := c.postSigned(url, body)
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

func (c *AgentController) postSigned(url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if len(c.sharedSecret) > 0 {
		mac := hmac.New(sha256.New, c.sharedSecret)
		mac.Write(body)
		req.Header.Set(controllerSignatureHeader, hex.EncodeToString(mac.Sum(nil)))
	}
	return c.cloudClient.Do(req)
}

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
