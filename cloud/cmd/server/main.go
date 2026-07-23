// Command server is the dipole cloud control-plane entry point. It assembles
// the database pool, repositories, services, and both the REST (management)
// and gRPC (edge data-plane) transports.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/config"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/dipolehq/dipole/cloud/internal/service/conditions"
	"github.com/dipolehq/dipole/cloud/internal/service/credentials"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
	"github.com/dipolehq/dipole/cloud/internal/service/pki"
	"github.com/dipolehq/dipole/cloud/internal/service/policy"
	"github.com/dipolehq/dipole/cloud/internal/service/registration"
	"github.com/dipolehq/dipole/cloud/internal/service/trust"
	grpctransport "github.com/dipolehq/dipole/cloud/internal/transport/grpc"
	"github.com/dipolehq/dipole/cloud/internal/transport/rest"
)

func main() {
	logger := setupLogger()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	encKey, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil || len(encKey) != 32 {
		logger.Error("DIPOLE_ENCRYPTION_KEY must be valid 64-char hex (32 bytes)")
		os.Exit(1)
	}

	logger.Info("starting dipole cloud",
		"region", cfg.Region,
		"grpc_port", cfg.GRPCPort,
		"http_port", cfg.HTTPPort,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	if err := run(ctx, cfg, encKey, logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config, encKey []byte, logger *slog.Logger) error {
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connected")

	repos := postgres.NewRepositories(pool)

	issuer, err := identity.NewIdentityIssuer("dipole", cfg.Region)
	if err != nil {
		return fmt.Errorf("init identity issuer: %w", err)
	}

	broker := credentials.NewCredentialBroker(encKey, issuer)
	trustValidator := trust.NewConfiguredService(logger)
	conditionEval := conditions.NewConditionEvaluator("")
	policyService := policy.NewPolicyEvaluationService(repos.AccessPolicies, trustValidator, conditionEval, broker)

	clientWorkloadSvc := service.NewClientWorkloadService(repos.ClientWorkloads)
	serverWorkloadSvc := service.NewServerWorkloadService(repos.ServerWorkloads)
	trustProviderSvc := service.NewTrustProviderService(repos.TrustProviders)
	accessConditionSvc := service.NewAccessConditionService(repos.AccessConditions)
	credentialProviderSvc := service.NewCredentialProviderService(repos.CredentialProviders, encKey)
	accessPolicySvc := service.NewAccessPolicyService(repos.AccessPolicies)
	authenticator := service.NewAPITokenAuthenticator(repos.Tenants)
	registrationSvc := registration.NewRegistrationService(repos.AgentControllers, repos.AgentProxies)
	caService := pki.NewTenantCAService(repos.TenantCAs, encKey)

	httpServer := buildHTTPServer(cfg, repos, clientWorkloadSvc, serverWorkloadSvc, trustProviderSvc, accessConditionSvc, credentialProviderSvc, accessPolicySvc, authenticator, registrationSvc)
	grpcHandler := grpctransport.NewPolicyEvaluationServer(policyService, registrationSvc, repos.AgentProxies, caService, logger)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("http server started", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpctransport.StartServer(runCtx, cfg.GRPCPort, grpcHandler, logger); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	var runErr error
	select {
	case <-runCtx.Done():
		logger.Info("shutdown signal received")
	case runErr = <-errCh:
		logger.Error("server error", "error", runErr)
	}

	cancel()
	shutdownHTTP(httpServer, logger)
	wg.Wait()

	logger.Info("shutdown complete")
	return runErr
}

func buildHTTPServer(
	cfg *config.Config,
	repos *postgres.Repositories,
	clientWorkloadSvc *service.ClientWorkloadService,
	serverWorkloadSvc *service.ServerWorkloadService,
	trustProviderSvc *service.TrustProviderService,
	accessConditionSvc *service.AccessConditionService,
	credentialProviderSvc *service.CredentialProviderService,
	accessPolicySvc *service.AccessPolicyService,
	authenticator rest.APIAuthenticator,
	registrationSvc *registration.RegistrationService,
) *http.Server {
	mux := http.NewServeMux()

	apiHandlers := rest.NewAPIHandlers(
		repos.AgentControllers,
		repos.Users,
		repos.Roles,
		repos.ResourceSets,
		repos.IdentityProviders,
		repos.SignOnPolicies,
		repos.LogStreams,
		repos.GlobalPolicyCompliance,
		repos.TenantProfiles,
		registrationSvc,
	)
	apiHandlers.RegisterAPIRoutes(mux)

	rest.NewClientWorkloadHandler(clientWorkloadSvc).RegisterRoutes(mux)
	rest.NewServerWorkloadHandler(serverWorkloadSvc).RegisterRoutes(mux)
	rest.NewTrustProviderHandler(trustProviderSvc).RegisterRoutes(mux)
	rest.NewAccessConditionHandler(accessConditionSvc).RegisterRoutes(mux)
	rest.NewCredentialProviderHandler(credentialProviderSvc).RegisterRoutes(mux)
	rest.NewAccessPolicyHandler(accessPolicySvc).RegisterRoutes(mux)
	rest.NewEventHandlers(repos.Events).RegisterEventRoutes(mux)
	rest.NewInternalHandlers(registrationSvc).RegisterRoutes(mux)

	handler := applyMiddleware(mux, authenticator)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func applyMiddleware(mux *http.ServeMux, authenticator rest.APIAuthenticator) http.Handler {
	rateLimiter := rest.NewRateLimiter(100, 30)

	protected := rateLimiter.Middleware(mux)
	protected = rest.AuthMiddleware(authenticator)(protected)

	routed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			mux.ServeHTTP(w, r)
			return
		}
		protected.ServeHTTP(w, r)
	})

	return rest.RequestIDMiddleware(rest.CORSMiddleware(rest.ResourceSetMiddleware(routed)))
}

func isPublicPath(path string) bool {
	return path == "/api/v1/health" || strings.HasPrefix(path, "/internal/")
}

func shutdownHTTP(srv *http.Server, logger *slog.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http graceful shutdown failed", "error", err)
	}
}

func setupLogger() *slog.Logger {
	level := slog.LevelInfo
	switch os.Getenv("DIPOLE_LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
