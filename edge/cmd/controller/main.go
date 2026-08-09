/*
Agent Controller entry point. Registers with Dipole Cloud using a device code
or Trust Provider attestation, exposes /api/token for proxy registration, and
reports health to Cloud every 60 seconds.
*/
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := loadControllerConfig()
	logger := newControllerLogger(cfg.LogLevel)

	ctrl := &AgentController{
		config:       cfg,
		logger:       logger,
		controllerID: uuid.Nil,
		cloudClient:  buildCloudHTTPClient(cfg.AgentTrustPath),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.registerWithRetry(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", ctrl.handleHealth)
	mux.HandleFunc("GET /api/token", ctrl.handleGetToken)
	mux.HandleFunc("POST /api/kerberos-attest", ctrl.handleKerberosAttest)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	go func() {
		if cfg.TLSCertPath != "" && cfg.TLSKeyPath != "" {
			logger.Info("controller https server starting", "port", cfg.HTTPPort)
			if err := server.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath); err != nil && err != http.ErrServerClosed {
				logger.Error("https server error", "error", err)
			}
		} else if cfg.HTTPPortDisabled {
			logger.Error("DIPOLE_HTTP_PORT_DISABLED=true but no TLS_PEM_PATH/TLS_KEY_PATH configured, cannot start")
			os.Exit(1)
		} else {
			logger.Info("controller http server starting", "port", cfg.HTTPPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("http server error", "error", err)
			}
		}
	}()

	go func() {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", promhttp.Handler())
		logger.Info("controller metrics server starting", "port", 9090)
		if err := http.ListenAndServe(":9090", metricsMux); err != nil {
			logger.Warn("metrics server error", "error", err)
		}
	}()

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
	sharedSecret     []byte
	registered       atomic.Bool
	registeredAt     time.Time
	lastCloudContact time.Time
	cloudClient      *http.Client
}
