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
	clientTLSKey := fs.String("client-tls-private-key", "", "Path to PEM private key for X.509-SVID retrieval")
	resourceSetID := fs.String("resource-set-id", "", "Scope to Resource Set")
	logLevel := fs.String("log-level", "warn", "Log level: off/trace/debug/info/warn/error")
	clientWorkloadID := fs.String("client-workload-id", "", "Explicit Client Workload UUID for disambiguation")

	fs.Parse(os.Args[3:])

	_ = logLevel
	_ = resourceSetID

	if *deploymentModel != "" && *idToken == "" {
		fmt.Fprintf(os.Stderr, "Error: --deployment-model=%s requires platform-specific attestation (not yet supported — use --id-token for OIDC attestation)\n", *deploymentModel)
		os.Exit(1)
	}
	if *clientTLSKey != "" {
		fmt.Fprintf(os.Stderr, "Error: --client-tls-private-key is not yet supported\n")
		os.Exit(1)
	}
	_ = clientWorkloadID

	if *clientID == "" || *serverHost == "" {
		fmt.Fprintf(os.Stderr, "Error: --client-id and --server-workload-host are required\n")
		os.Exit(1)
	}

	endpoint, identityType, err := deriveEndpoint(*clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	accessToken, err := callAuth(client, endpoint, *clientID, *idToken, identityType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: authentication failed — %s\n", err)
		os.Exit(1)
	}

	cred, err := callCredentials(client, endpoint, accessToken, *idToken, identityType, *serverHost, *serverPort, *credType)
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
	endpoint = fmt.Sprintf("https://%s.ec.%s.dipole.io", tenant, region)
	return endpoint, identityType, nil
}

func callAuth(client *http.Client, endpoint, clientID, idToken, identityType string) (string, error) {
	body := map[string]any{
		"clientId": clientID,
		"client":   buildClientBlock(idToken, identityType),
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

func callCredentials(client *http.Client, endpoint, accessToken, idToken, identityType, host string, port int, credentialType string) (map[string]any, error) {
	body := map[string]any{
		"client":         buildClientBlock(idToken, identityType),
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

func buildClientBlock(idToken, identityType string) map[string]any {
	switch identityType {
	case "github_idtoken":
		return map[string]any{"github": map[string]any{"identityToken": idToken}}
	case "gitlab_idtoken":
		return map[string]any{"gitlab": map[string]any{"identityToken": idToken}}
	case "terraform_cloud_idtoken":
		return map[string]any{"terraform": map[string]any{"identityToken": idToken}}
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
