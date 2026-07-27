/*
cabundle.go patches the MutatingWebhookConfiguration with the injector's
self-signed CA certificate so the Kubernetes API server trusts the webhook.

Uses raw HTTP against the K8s API (in-cluster service account token + CA)
to avoid pulling k8s.io/client-go and its ~100 transitive dependencies.
*/
package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func patchWebhookCABundle(webhookName string, caCertDER []byte, logger *slog.Logger) error {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		logger.Warn("not running in cluster, skipping caBundle patch")
		return nil
	}

	tokenBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return fmt.Errorf("read service account token: %w", err)
	}
	token := string(tokenBytes)

	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return fmt.Errorf("read cluster CA: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse cluster CA certificate")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    certPool,
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// K8s caBundle field is a []byte containing PEM-encoded CA cert.
	// In JSON, []byte fields are base64-encoded.
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	caBundle := base64.StdEncoding.EncodeToString(caPEM)
	patch := fmt.Sprintf(`[{"op":"replace","path":"/webhooks/0/clientConfig/caBundle","value":"%s"}]`, caBundle)

	url := fmt.Sprintf("https://%s:%s/apis/admissionregistration.k8s.io/v1/mutatingwebhookconfigurations/%s", host, port, webhookName)

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBufferString(patch))
	if err != nil {
		return fmt.Errorf("build patch request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("patch webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("patch webhook returned %d: %s", resp.StatusCode, string(body))
	}

	logger.Info("caBundle patched into MutatingWebhookConfiguration", "webhook", webhookName, "status", resp.StatusCode)
	return nil
}
