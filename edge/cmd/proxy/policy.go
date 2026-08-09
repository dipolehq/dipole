/* Policy evaluation logic for the agent proxy. */
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/cloudclient"
	"github.com/dipolehq/dipole/edge/internal/config"
	"github.com/dipolehq/dipole/edge/internal/credcache"
	"github.com/dipolehq/dipole/edge/internal/identity"
	"github.com/dipolehq/dipole/edge/internal/injection"
	"github.com/dipolehq/dipole/edge/internal/metrics"
)

type policyResult struct {
	Allowed          bool
	Credential       *injection.Credential
	InjectionConfig  *injection.InjectionConfig
	PolicyID         string
	ClientWorkloadID string
	ServerWorkloadID string
}

func evaluatePolicy(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, sourceIP string, sourcePort int, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) *policyResult {
	if cloud == nil {
		if !cfg.DevMode {
			logger.Error("no cloud endpoint configured and DIPOLE_DEV_MODE is not enabled, denying", "connection_id", connID)
			return &policyResult{Allowed: false}
		}
		return &policyResult{Allowed: true}
	}

	clientID := sourceIP
	if cfg.ClientWorkloadID != "" {
		clientID = cfg.ClientWorkloadID
	}

	if entry, ok := credCache.Get(clientID, targetHost, targetPort); ok {
		metrics.CredentialCacheHits.Inc()
		return &policyResult{
			Allowed:          entry.Allowed,
			Credential:       entry.Credential,
			InjectionConfig:  entry.InjectionConfig,
			PolicyID:         entry.PolicyID,
			ClientWorkloadID: entry.ClientWorkloadID,
			ServerWorkloadID: entry.ServerWorkloadID,
		}
	}

	evidence := gatherer.Gather(sourceIP, sourcePort)

	ctx := context.Background()
	start := time.Now()
	result, err := cloud.Evaluate(ctx, evidence, targetHost, targetPort)
	metrics.PolicyEvaluationDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		if entry, ok := credCache.Get(clientID, targetHost, targetPort); ok {
			logger.Debug("serving cached credential during cloud outage", "host", targetHost, "connection_id", connID)
			return &policyResult{
				Allowed:          entry.Allowed,
				Credential:       entry.Credential,
				InjectionConfig:  entry.InjectionConfig,
				PolicyID:         entry.PolicyID,
				ClientWorkloadID: entry.ClientWorkloadID,
				ServerWorkloadID: entry.ServerWorkloadID,
			}
		}
		if cfg.PassThroughBeforeRegistration && !cfg.Registered.Load() {
			logger.Debug("cloud evaluation failed, passthrough mode (pre-registration)", "error", err, "connection_id", connID)
			return &policyResult{Allowed: true}
		}
		logger.Warn("cloud evaluation failed", "error", err, "connection_id", connID)
		metrics.PolicyEvaluationsTotal.WithLabelValues("error").Inc()
		return &policyResult{Allowed: false}
	}

	if !result.Allowed {
		logger.Warn("access denied", "reason", result.DenyReason, "host", targetHost, "connection_id", connID)
		metrics.PolicyEvaluationsTotal.WithLabelValues("denied").Inc()
		return &policyResult{
			Allowed:          false,
			PolicyID:         result.PolicyID,
			ClientWorkloadID: result.ClientWorkloadID,
			ServerWorkloadID: result.ServerWorkloadID,
		}
	}

	metrics.PolicyEvaluationsTotal.WithLabelValues("allowed").Inc()
	metrics.CredentialCacheMisses.Inc()

	pr := &policyResult{
		Allowed:          true,
		PolicyID:         result.PolicyID,
		ClientWorkloadID: result.ClientWorkloadID,
		ServerWorkloadID: result.ServerWorkloadID,
	}

	if result.CredentialValue != "" || result.Username != "" || result.AwsAccessKeyId != "" {
		pr.Credential = &injection.Credential{
			Type:               result.CredentialType,
			Value:              result.CredentialValue,
			Username:           result.Username,
			Password:           result.Password,
			AwsAccessKeyId:     result.AwsAccessKeyId,
			AwsSecretAccessKey: result.AwsSecretAccessKey,
			AwsSessionToken:    result.AwsSessionToken,
		}
	}

	if result.InjectionMethod != "" {
		pr.InjectionConfig = &injection.InjectionConfig{
			Method:     result.InjectionMethod,
			HeaderName: result.InjectionHeader,
			Prefix:     result.InjectionPrefix,
			QueryParam: result.InjectionQuery,
		}
	}

	var expiresAt time.Time
	if result.ExpiresAt > 0 {
		expiresAt = time.Unix(result.ExpiresAt, 0)
	} else {
		expiresAt = time.Now().Add(credCache.DefaultTTL())
	}

	credCache.Set(clientID, targetHost, targetPort, &credcache.CacheEntry{
		Allowed:          true,
		Credential:       pr.Credential,
		InjectionConfig:  pr.InjectionConfig,
		PolicyID:         pr.PolicyID,
		ClientWorkloadID: pr.ClientWorkloadID,
		ServerWorkloadID: pr.ServerWorkloadID,
		ExpiresAt:        expiresAt,
	})
	metrics.CredentialCacheSize.Set(float64(credCache.Size()))

	return pr
}
