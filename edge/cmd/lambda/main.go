/*
Lambda Extension wrapper. Registers with the AWS Lambda Extensions API, starts
the Agent Proxy binary as a subprocess, and manages its lifecycle across Lambda
invocations.

The Layer zip contains:

	extensions/dipole-proxy       (this binary — the extension entry point)
	extensions/dipole-proxy-bin   (the actual proxy binary from cmd/proxy)

The function must set HTTP_PROXY=http://localhost:8000 and
HTTPS_PROXY=http://localhost:8000 in its environment variables.

Requirements: Lambda function must be in a VPC with NAT Gateway for Cloud access.
*/
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	extensionsAPI = "http://127.0.0.1:9001/2020-01-31/extension"
	extensionName = "dipole-proxy"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	extensionID, err := registerExtension()
	if err != nil {
		logger.Error("extension registration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("lambda extension registered", "id", extensionID)

	proxyCmd := startProxyProcess(logger)
	if proxyCmd == nil {
		logger.Error("cannot start without proxy binary, exiting")
		os.Exit(1)
	}

	for {
		event, err := nextEvent(extensionID)
		if err != nil {
			logger.Error("next event failed", "error", err)
			break
		}

		if event.EventType == "SHUTDOWN" {
			logger.Info("shutdown event received")
			break
		}
	}

	if proxyCmd != nil && proxyCmd.Process != nil {
		proxyCmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- proxyCmd.Wait() }()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			proxyCmd.Process.Kill()
		}
	}
}

func startProxyProcess(logger *slog.Logger) *exec.Cmd {
	selfDir := filepath.Dir(os.Args[0])
	proxyBin := filepath.Join(selfDir, "dipole-proxy-bin")

	if _, err := os.Stat(proxyBin); err != nil {
		logger.Error("proxy binary not found", "path", proxyBin)
		return nil
	}

	cmd := exec.Command(proxyBin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		logger.Error("failed to start proxy process", "error", err)
		return nil
	}

	logger.Info("proxy process started", "pid", cmd.Process.Pid, "bin", proxyBin)
	return cmd
}

type extensionEvent struct {
	EventType string `json:"eventType"`
}

func registerExtension() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	body := `{"events":["INVOKE","SHUTDOWN"]}`
	req, err := http.NewRequest("POST", extensionsAPI+"/register", io.NopCloser(strings.NewReader(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Lambda-Extension-Name", extensionName)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("register returned %d: %s", resp.StatusCode, string(b))
	}

	return resp.Header.Get("Lambda-Extension-Identifier"), nil
}

func nextEvent(extensionID string) (*extensionEvent, error) {
	client := &http.Client{Timeout: 0}

	req, err := http.NewRequest("GET", extensionsAPI+"/event/next", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Lambda-Extension-Identifier", extensionID)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("next event: %w", err)
	}
	defer resp.Body.Close()

	var event extensionEvent
	if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}

	return &event, nil
}
