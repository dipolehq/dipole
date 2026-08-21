/*
Dipole CLI — command-line tool for proxyless credential retrieval in CI/CD.

Usage: dipole credentials get --client-id <id> --server-workload-host <host> --server-workload-port <port> --id-token <jwt>
Output: export TOKEN='<credential_value>' (shell-evaluable)

The CLI wraps the Edge API (/edge/v1/auth + /edge/v1/credentials) for use in
CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins) where no Agent Proxy is deployed.
*/
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "credentials" || os.Args[2] != "get" {
		fmt.Fprintf(os.Stderr, "Usage: dipole credentials get [flags]\n")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("credentials get", flag.ExitOnError)
	clientID := fs.String("client-id", "", "Edge SDK Client ID (required). Format: dipole:<region>:<tenant>:identity:<type>:<uuid>")
	serverHost := fs.String("server-workload-host", "", "Target server hostname (required)")
	serverPort := fs.Int("server-workload-port", 443, "Target server port (required)")
	idToken := fs.String("id-token", "", "OIDC identity token for attestation")
	credNames := fs.String("credential-names", "TOKEN", "Env var name(s) for output (comma-separated)")
	outputFormat := fs.String("output-format", "sh-export", "Output format: sh-export, sh-env, powershell-env, json")
	credType := fs.String("credential-type", "ApiKey", "Expected credential type: ApiKey, UsernamePassword, OAuthToken, AwsStsFederation")
	deploymentModel := fs.String("deployment-model", "", "Environment type: vm, kubernetes, ecs_fargate, lambda_container")
	clientTLSKey := fs.String("client-tls-private-key", "", "Path to PEM private key, paired with --client-tls-certificate, for certificate_signed_attestation")
	clientTLSCert := fs.String("client-tls-certificate", "", "Path to PEM certificate, paired with --client-tls-private-key, for certificate_signed_attestation")
	resourceSetID := fs.String("resource-set-id", "", "Scope to Resource Set")
	logLevel := fs.String("log-level", "warn", "Log level: off/trace/debug/info/warn/error")
	clientWorkloadID := fs.String("client-workload-id", "", "Explicit Client Workload UUID for disambiguation")

	fs.Parse(os.Args[3:])

	_ = logLevel
	_ = resourceSetID
	_ = clientWorkloadID

	// Cheap, universally-required flags checked first, before any file
	// I/O or signing work (deployment-model discovery, certificate
	// loading) that would otherwise run needlessly on a request that's
	// going to fail regardless.
	if *clientID == "" || *serverHost == "" {
		fmt.Fprintf(os.Stderr, "Error: --client-id and --server-workload-host are required\n")
		os.Exit(1)
	}

	if *deploymentModel != "" && *idToken == "" {
		discovered := discoverPlatformToken(*deploymentModel)
		if discovered == "" {
			fmt.Fprintf(os.Stderr, "Error: --deployment-model=%s but could not discover platform identity token automatically — provide --id-token explicitly\n", *deploymentModel)
			os.Exit(1)
		}
		idToken = &discovered
	}
	var certAttest *certificateAttestation
	if *clientTLSKey != "" || *clientTLSCert != "" {
		if *clientTLSKey == "" || *clientTLSCert == "" {
			fmt.Fprintf(os.Stderr, "Error: --client-tls-private-key and --client-tls-certificate must be provided together\n")
			os.Exit(1)
		}
		attest, err := buildCertificateAttestation(*clientTLSCert, *clientTLSKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		certAttest = attest
	}

	endpoint, identityType, err := deriveEndpoint(*clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	if certAttest != nil && identityType != "certificate" {
		fmt.Fprintf(os.Stderr, "Warning: --client-tls-certificate/--client-tls-private-key were provided, but --client-id's identity type is %q, not certificate — the certificate attestation will not be sent\n", identityType)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	accessToken, err := callAuth(client, endpoint, *clientID, *idToken, identityType, certAttest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: authentication failed — %s\n", err)
		os.Exit(1)
	}

	cred, err := callCredentials(client, endpoint, accessToken, *idToken, identityType, *serverHost, *serverPort, *credType, certAttest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: credential retrieval failed — %s\n", err)
		os.Exit(1)
	}

	output := formatOutput(cred, *credNames, *outputFormat)
	fmt.Print(output)
}

func deriveEndpoint(clientID string) (endpoint, identityType string, err error) {
	parts := strings.Split(clientID, ":")
	if len(parts) != 6 || parts[0] != "dipole" || parts[3] != "identity" {
		return "", "", fmt.Errorf("invalid client-id format — expected dipole:<region>:<tenant>:identity:<type>:<uuid>")
	}
	region := parts[1]
	tenant := parts[2]
	identityType = parts[4]
	endpoint = fmt.Sprintf("https://%s.ec.%s.dipolehq.com", tenant, region)
	return endpoint, identityType, nil
}

func callAuth(client *http.Client, endpoint, clientID, idToken, identityType string, certAttest *certificateAttestation) (string, error) {
	body := map[string]any{
		"clientId": clientID,
		"client":   buildClientBlock(idToken, identityType, certAttest),
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", endpoint+"/edge/v1/auth", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

func callCredentials(client *http.Client, endpoint, accessToken, idToken, identityType, host string, port int, credentialType string, certAttest *certificateAttestation) (map[string]any, error) {
	body := map[string]any{
		"client":         buildClientBlock(idToken, identityType, certAttest),
		"server":         map[string]any{"host": host, "port": port, "transportProtocol": "TCP"},
		"credentialType": credentialType,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", endpoint+"/edge/v1/credentials", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if ct, _ := result["credentialType"].(string); ct == "Unknown" {
		return nil, fmt.Errorf("no matching access policy found for %s:%d", host, port)
	}

	return result, nil
}

func buildClientBlock(idToken, identityType string, certAttest *certificateAttestation) map[string]any {
	switch identityType {
	case "github_idtoken":
		return map[string]any{"github": map[string]any{"identityToken": idToken}}
	case "gitlab_idtoken":
		return map[string]any{"gitlab": map[string]any{"identityToken": idToken}}
	case "terraform_cloud_idtoken":
		return map[string]any{"terraform": map[string]any{"identityToken": idToken}}
	case "certificate":
		if certAttest == nil {
			return map[string]any{}
		}
		return map[string]any{"certificate": map[string]any{
			"certificatePem": certAttest.CertificatePEM,
			"timestamp":      certAttest.Timestamp,
			"signature":      certAttest.SignatureBase64,
		}}
	default:
		return map[string]any{"oidc": map[string]any{"identityToken": idToken}}
	}
}

func formatOutput(cred map[string]any, credNames, format string) string {
	data, _ := cred["data"].(map[string]any)
	if data == nil {
		return ""
	}

	names := strings.Split(credNames, ",")
	values := extractValues(data, names)

	var lines []string
	for i, name := range names {
		if i >= len(values) {
			break
		}
		val := shellEscape(values[i])
		switch format {
		case "sh-export":
			lines = append(lines, fmt.Sprintf("export %s='%s'", name, val))
		case "sh-env":
			lines = append(lines, fmt.Sprintf("%s=%s", name, val))
		case "powershell-env":
			lines = append(lines, fmt.Sprintf("$env:%s = \"%s\"", name, val))
		case "json":
			jsonOut, _ := json.Marshal(map[string]string{name: values[i]})
			return string(jsonOut) + "\n"
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func extractValues(data map[string]any, names []string) []string {
	// Map credential data fields to output names
	if v, ok := data["apiKey"].(string); ok {
		return []string{v}
	}
	if v, ok := data["token"].(string); ok {
		return []string{v}
	}
	if u, ok := data["username"].(string); ok {
		p, _ := data["password"].(string)
		return []string{u, p}
	}
	if ak, ok := data["awsAccessKeyId"].(string); ok {
		sk, _ := data["awsSecretAccessKey"].(string)
		st, _ := data["awsSessionToken"].(string)
		return []string{ak, sk, st}
	}
	// Fallback: first string value
	for _, v := range data {
		if s, ok := v.(string); ok {
			return []string{s}
		}
	}
	return nil
}

func shellEscape(s string) string {
	// Escape single quotes for POSIX shell (replace ' with '\'' )
	return strings.ReplaceAll(s, "'", "'\\''")
}

func discoverPlatformToken(model string) string {
	switch model {
	case "kubernetes":
		token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(token))
	case "vm", "ecs_fargate", "lambda_container":
		client := &http.Client{Timeout: 2 * time.Second}
		req, _ := http.NewRequest("PUT", "http://169.254.169.254/latest/api/token", nil)
		req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")
		resp, err := client.Do(req)
		if err != nil {
			return ""
		}
		defer resp.Body.Close()
		imdsToken, _ := io.ReadAll(resp.Body)

		idReq, _ := http.NewRequest("GET", "http://169.254.169.254/latest/dynamic/instance-identity/document", nil)
		idReq.Header.Set("X-aws-ec2-metadata-token", string(imdsToken))
		idResp, err := client.Do(idReq)
		if err != nil {
			return ""
		}
		defer idResp.Body.Close()
		doc, _ := io.ReadAll(idResp.Body)
		return strings.TrimSpace(string(doc))
	default:
		return ""
	}
}
