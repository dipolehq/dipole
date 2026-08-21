/*
Agent Proxy entry point. Initializes the TLS MITM pipeline, Cloud gRPC client,
protocol detection, metrics server, and both explicit + transparent listeners.
*/
package main

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/dipolehq/dipole/edge/internal/cloudclient"
	"github.com/dipolehq/dipole/edge/internal/config"
	"github.com/dipolehq/dipole/edge/internal/credcache"
	"github.com/dipolehq/dipole/edge/internal/identity"
	"github.com/dipolehq/dipole/edge/internal/metrics"
	"github.com/dipolehq/dipole/edge/internal/protocols"
	"github.com/dipolehq/dipole/edge/internal/proxy"
	dipoletls "github.com/dipolehq/dipole/edge/internal/tls"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	logger.Info("starting agent proxy",
		"tenant_id", cfg.TenantID,
		"port", cfg.HTTPServerPort,
		"sigterm_strategy", cfg.SIGTERMStrategy,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cloud *cloudclient.CloudClient
	if cfg.CloudEndpoint != "" {
		cloud, err = cloudclient.NewCloudClient(cfg.CloudEndpoint, cfg.TenantID, cfg.AgentTrustPath, cfg.GRPCInsecure, logger)
		if err != nil {
			logger.Error("failed to connect to cloud", "endpoint", cfg.CloudEndpoint, "error", err)
			os.Exit(1)
		}
		defer cloud.Close()
		logger.Info("cloud client connected", "endpoint", cfg.CloudEndpoint)
	} else {
		logger.Warn("no DIPOLE_CLOUD_GRPC_ENDPOINT configured, running without policy evaluation")
	}

	if cloud != nil && cfg.ResourceSetID != "" {
		cloud.SetResourceSetID(cfg.ResourceSetID)
	}

	var regResult *cloudclient.RegistrationResult
	var activeConnCount atomic.Int64
	if cloud != nil {
		regResult = cloudclient.RunRegistration(ctx, cfg.AgentController, cloud, cfg.PassThroughBeforeRegistration, cfg.PostStartMaxWaitSec, func() int32 { return int32(activeConnCount.Load()) }, logger)
		if regResult != nil && regResult.Success {
			cfg.Registered.Store(true)
		}
	}

	// TLS Decrypt requires a live Cloud connection to sign leaf certificates
	// with the tenant's Root CA — the private key never leaves Cloud, so
	// there is no local/dev fallback. If Cloud is unavailable or did not
	// return a CA certificate, the proxy still starts (other protocols and
	// policy evaluation degrade the same way), but the MITM engine is left
	// nil and HTTPS/TLS-Decrypt connections fail closed per-connection.
	var engine *dipoletls.MITMEngine
	if cloud != nil {
		anchor, err := buildTrustAnchor(regResult)
		if err != nil {
			logger.Error("failed to obtain tenant CA certificate, TLS Decrypt disabled", "error", err)
		} else {
			writeCAToSharedVolume(anchor, logger)
			cache := dipoletls.NewCertCache(anchor, cloud)
			engine = dipoletls.NewMITMEngine(cache)
		}
	} else {
		logger.Warn("no cloud connection, TLS Decrypt disabled")
	}

	credentialCache := credcache.NewCredentialCache(60 * time.Second)
	go credentialCache.StartCleanupLoop(ctx, 5*time.Minute)

	httpHandler := &protocols.HTTPHandler{Logger: logger}
	gatherer := identity.NewEvidenceGatherer(logger, cfg.ProcessIdentificationEnabled)
	handler := newConnectionHandler(engine, httpHandler, cloud, gatherer, credentialCache, cfg, &activeConnCount, logger)

	if cfg.MetricsEnabled {
		go func() {
			if err := metrics.StartMetricsServer(cfg.MetricsPort); err != nil {
				logger.Warn("metrics server failed", "error", err)
			}
		}()
	}

	explicitListener := proxy.NewExplicitListener(cfg, logger, handler)
	go func() {
		if err := explicitListener.Start(ctx); err != nil {
			logger.Error("explicit listener failed", "error", err)
			cancel()
		}
	}()

	transparentListener := proxy.NewTransparentListener(cfg, logger, handler)
	go func() {
		if err := transparentListener.Start(ctx); err != nil {
			logger.Warn("transparent listener failed (expected on non-Linux)", "error", err)
		}
	}()

	waitForShutdown(cancel, cfg, logger, explicitListener)
}
