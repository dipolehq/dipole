/*
Package config provides environment-based configuration for the Agent Proxy.

LoadFromEnv reads DIPOLE_* and K8S_* environment variables, applies defaults,
validates required fields, and returns a ProxyConfig ready for use.
*/
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

type ProxyConfig struct {
	TenantID                      string
	AgentController               string
	CloudEndpoint                 string
	HTTPServerPort                int
	MetricsPort                   int
	HTTPIdleTimeoutSecs           int
	LogLevel                      string
	SIGTERMStrategy               string
	PassThroughBeforeRegistration bool
	PostStartMaxWaitSec           int
	SteeringAllowedHosts          []string
	K8sPodName                    string
	K8sNamespace                  string
	ResourceSetID                 string
	ClientWorkloadID              string
	ProcessIdentificationEnabled  bool
	AgentTrustPath                string
	GRPCInsecure                  bool
	DevMode                       bool
	MetricsEnabled                bool
	Registered                    *atomic.Bool
}

func LoadFromEnv() (*ProxyConfig, error) {
	cfg := &ProxyConfig{
		TenantID:                      os.Getenv("DIPOLE_TENANT_ID"),
		AgentController:               os.Getenv("DIPOLE_AGENT_CONTROLLER"),
		CloudEndpoint:                 envStr("DIPOLE_CLOUD_GRPC_ENDPOINT", ""),
		HTTPServerPort:                envInt("DIPOLE_HTTP_SERVER_PORT", 8000),
		MetricsPort:                   envInt("DIPOLE_METRICS_PORT", 9099),
		HTTPIdleTimeoutSecs:           envInt("DIPOLE_HTTP_IDLE_TIMEOUT_SECS", 3600),
		LogLevel:                      envStr("DIPOLE_LOG_LEVEL", "info"),
		SIGTERMStrategy:               envStr("DIPOLE_SIGTERM_STRATEGY", "immediate"),
		PassThroughBeforeRegistration: envBool("DIPOLE_PASS_THROUGH_TRAFFIC_BEFORE_REGISTRATION", true),
		PostStartMaxWaitSec:           envInt("DIPOLE_POST_START_MAX_WAIT_SEC", 60),
		K8sPodName:                    os.Getenv("K8S_POD_NAME"),
		K8sNamespace:                  os.Getenv("K8S_NAMESPACE"),
		ResourceSetID:                 os.Getenv("DIPOLE_RESOURCE_SET_ID"),
		ClientWorkloadID:              os.Getenv("CLIENT_WORKLOAD_ID"),
		ProcessIdentificationEnabled:  envBool("DIPOLE_CLIENT_WORKLOAD_PROCESS_IDENTIFICATION_ENABLED", false),
		AgentTrustPath:                os.Getenv("AGENT_TRUST_PATH"),
		GRPCInsecure:                  envBool("DIPOLE_GRPC_INSECURE", false),
		DevMode:                       envBool("DIPOLE_DEV_MODE", false),
		MetricsEnabled:                envBool("DIPOLE_METRICS_ENABLED", true),
		Registered:                    &atomic.Bool{},
	}

	if hosts := os.Getenv("DIPOLE_STEERING_ALLOWED_HOSTS"); hosts != "" {
		cfg.SteeringAllowedHosts = strings.Split(hosts, ",")
	}

	if cfg.TenantID == "" {
		return nil, fmt.Errorf("DIPOLE_TENANT_ID is required")
	}

	return cfg, nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
