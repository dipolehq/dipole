/*
Command server is the Dipole Cloud control-plane entry point. Assembles the
database pool, repositories, services, and both the REST and gRPC transports.
*/
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
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/config"
	eventbus "github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/dipolehq/dipole/cloud/internal/service/auth"
	"github.com/dipolehq/dipole/cloud/internal/service/conditions"
	"github.com/dipolehq/dipole/cloud/internal/service/credentials"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
	mcpsvc "github.com/dipolehq/dipole/cloud/internal/service/mcp"
	mcpgw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
	"github.com/dipolehq/dipole/cloud/internal/service/pki"
	"github.com/dipolehq/dipole/cloud/internal/service/policy"
	"github.com/dipolehq/dipole/cloud/internal/service/registration"
	"github.com/dipolehq/dipole/cloud/internal/service/reporting"
	"github.com/dipolehq/dipole/cloud/internal/service/resources"
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
	var migrationDone bool
	for attempt := 1; attempt <= 5; attempt++ {
		migrator, merr := postgres.NewMigrator(cfg.DatabaseURL)
		if merr != nil {
			logger.Warn("migration connection failed, retrying", "attempt", attempt, "error", merr)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		if merr = migrator.Up(); merr != nil {
			_ = migrator.Close()
			return fmt.Errorf("migration failed: %w", merr)
		}
		_ = migrator.Close()
		logger.Info("schema up to date")
		migrationDone = true
		break
	}
	if !migrationDone {
		return fmt.Errorf("database migrations failed after 5 connection attempts")
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connected")

	repos := postgres.NewRepositories(pool)
	repos.SetMCPEncryptionKey(encKey)
	repos.SetLogStreamEncryptionKey(encKey)
	repos.SetTrustProviderEncryptionKey(encKey)

	if n, err := repos.LogStreams.BackfillConfigEncryption(ctx); err != nil {
		return fmt.Errorf("backfill log stream config encryption: %w", err)
	} else if n > 0 {
		logger.Info("backfilled log stream config encryption", "count", n)
	}

	if n, err := repos.TrustProviders.BackfillSymmetricKeyEncryption(ctx); err != nil {
		return fmt.Errorf("backfill trust provider symmetric key encryption: %w", err)
	} else if n > 0 {
		logger.Info("backfilled trust provider symmetric key encryption", "count", n)
	}

	if err := devBootstrap(ctx, repos, cfg, logger); err != nil {
		return fmt.Errorf("dev bootstrap: %w", err)
	}

	eventBus := eventbus.NewEventBus(postgres.NewEventPipelineRepo(pool), logger)

	platformHosts := platform.NewHosts(cfg.PlatformDomain, cfg.Env)

	issuer, err := identity.NewIdentityIssuer("https://id." + platformHosts.Domain())
	if err != nil {
		return fmt.Errorf("init identity issuer: %w", err)
	}

	broker := credentials.NewCredentialBroker(encKey, issuer, repos.Tenants, platformHosts)
	trustValidator := trust.NewConfiguredService(logger, repos.AgentControllers, encKey)
	conditionEval := conditions.NewConditionEvaluator(cfg.GeoIPDBPath, logger)
	policyService := policy.NewPolicyEvaluationService(repos.AccessPolicies, trustValidator, conditionEval, broker, eventBus, logger)

	clientWorkloadSvc := resources.NewClientWorkloadService(repos.ClientWorkloads)
	serverWorkloadSvc := resources.NewServerWorkloadService(repos.ServerWorkloads)
	trustProviderSvc := resources.NewTrustProviderService(repos.TrustProviders, repos.Tenants, cfg.Region, platformHosts, encKey)
	accessConditionSvc := resources.NewAccessConditionService(repos.AccessConditions)
	credentialProviderSvc := resources.NewCredentialProviderService(repos.CredentialProviders, encKey)
	accessPolicySvc := resources.NewAccessPolicyService(repos.AccessPolicies, repos.ClientWorkloads, repos.ServerWorkloads, repos.TrustProviders, repos.CredentialProviders, repos.AccessConditions)
	authenticator := auth.NewAPITokenAuthenticator(repos.Tenants)
	authenticator.SetIssuer(issuer)
	authenticator.SetAPITokenRepo(repos.APITokens)
	registrationSvc := registration.NewRegistrationService(repos.AgentControllers, repos.AgentProxies, repos.RegistrationTokens, encKey)
	caService := pki.NewTenantCAService(repos.TenantCAs, encKey)

	resourceCounter := service.NewDBResourceCounter(
		repos.ClientWorkloads, repos.ServerWorkloads, repos.AccessPolicies, repos.TrustProviders,
		repos.CredentialProviders, repos.AccessConditions, repos.ResourceSets, repos.Users,
		repos.AgentControllers, repos.LogStreams,
	)
	planLimiter := service.NewPlanLimiter(repos.Tenants, resourceCounter)
	clientWorkloadSvc.SetLimiter(planLimiter)
	serverWorkloadSvc.SetLimiter(planLimiter)
	trustProviderSvc.SetLimiter(planLimiter)
	accessConditionSvc.SetLimiter(planLimiter)
	credentialProviderSvc.SetLimiter(planLimiter)
	accessPolicySvc.SetLimiter(planLimiter)

	edgeTrustValidator := trust.NewEdgeTrustValidator(repos.TrustProviders, trustValidator)
	edgePolicyAdapter := policy.NewEdgePolicyAdapter(policyService)
	edgeTenantResolver := service.NewTenantResolverAdapter(repos.Tenants)
	edgeAPI := rest.NewEdgeAPIHandler(issuer, edgeTrustValidator, edgePolicyAdapter, edgeTenantResolver)

	tenantHostResolver := rest.NewTenantHostResolver(repos.Tenants, platformHosts, cfg.DevBootstrap)
	if cfg.DevBootstrap {
		logger.Warn("dev bootstrap enabled: tenant may be selected via header — never enable this in a deployed environment",
			"header", rest.DevTenantHeader)
	}

	mcpCodeStore := mcpsvc.NewAuthCodeStore()
	mcpDCR := mcpsvc.NewDCRService(repos.MCPClients)
	mcpPolicyFinder := mcpsvc.NewMCPPolicyFinder(repos.AccessPolicies)
	mcpIdPResolver := mcpsvc.NewDBIdentityProviderResolver(repos.IdentityProviders, &oidcDiscoveryAdapter{client: auth.NewHTTPOIDCClient()})
	mcpIdPService := mcpsvc.NewIdPService(mcpIdPResolver)
	mcpIdPCallback := mcpsvc.NewIdPCallbackService(mcpIdPService, mcpCodeStore, trustValidator, conditionEval)
	mcpTokenConfigResolver := mcpsvc.NewPolicyTokenConfigResolver(repos.AccessPolicies, repos.CredentialProviders, repos.ClientWorkloads, encKey)
	mcpTokenService := mcpsvc.NewTokenService(mcpCodeStore, issuer, mcpTokenConfigResolver)
	mcpAuthorizeService := mcpsvc.NewAuthorizeService(mcpDCR, mcpCodeStore, mcpPolicyFinder, conditionEval, mcpIdPService)

	mcpAuthServer := rest.NewMCPAuthServer(mcpDCR, mcpAuthorizeService, mcpTokenService, mcpIdPCallback, tenantHostResolver, logger)

	wellKnownHandler := rest.NewWellKnownHandler(issuer, tenantHostResolver, logger)

	gwLeftAuth := mcpgw.NewLeftSideAuth(issuer)
	gwPolicyFinder := mcpgw.NewDBRightSidePolicyFinder(repos.AccessPolicies)
	gwTokenStore := repos.MCPUserCredentials
	gwDecrypter := mcpgw.NewAESCredentialDecrypter(encKey)
	gwSharedOAuth := mcpgw.NewAESSharedOAuthTokenResolver(encKey)

	consentConfigResolver := mcpgw.NewCPConsentConfigResolver(repos.AccessPolicies, repos.CredentialProviders, encKey)
	consentCallback := mcpgw.NewConsentCallbackService(gwTokenStore, consentConfigResolver)
	gwUserConsent := mcpgw.NewUserConsentService(gwTokenStore, consentConfigResolver)
	gwTokenRefresher := mcpgw.NewOAuthTokenRefresher(consentConfigResolver)

	gwRightAuth := mcpgw.NewRightSideAuth(gwPolicyFinder, gwTokenStore, gwTokenRefresher, gwDecrypter, gwSharedOAuth)
	gwRightAuth.SetLogger(logger)
	gwClientFactory := mcpgw.NewHTTPMCPClientFactory()

	gwMethodHandler := mcpgw.NewMethodHandler(gwRightAuth, gwClientFactory, gwUserConsent, eventBus, logger)

	mcpGatewayServer := rest.NewMCPGatewayServer(gwLeftAuth, gwMethodHandler, consentCallback, logger, tenantHostResolver)

	logger.Info("MCP services initialized",
		"platform_domain", platformHosts.Domain(),
		"auth_server_pattern", "https://<tenant>-mcp."+platformHosts.Domain(),
		"identity_issuer_pattern", "https://<tenant>-id."+platformHosts.Domain(),
		"gateway_pattern", "https://<tenant>-mcpgateway."+platformHosts.Domain(),
	)

	httpServer := buildHTTPServer(cfg, repos, encKey, planLimiter, clientWorkloadSvc, serverWorkloadSvc, trustProviderSvc, accessConditionSvc, credentialProviderSvc, accessPolicySvc, authenticator, registrationSvc, edgeAPI, mcpAuthServer, mcpGatewayServer, wellKnownHandler, issuer, eventBus, caService, logger)
	grpcHandler := grpctransport.NewPolicyEvaluationServer(policyService, registrationSvc, repos.AgentProxies, caService, eventBus, logger)
	grpcHandler.SetLeafSigner(caService)
	grpcHandler.SetHostnameAuthorizer(repos.AccessPolicies)

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
		if err := grpctransport.StartServer(runCtx, cfg.GRPCPort, grpcHandler, cfg.GRPCTLSCert, cfg.GRPCTLSKey, repos.AgentProxies, logger); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	logStreamSvc := reporting.NewLogStreamService(repos.Events, repos.LogStreams, logger)
	logStreamSvc.RegisterExporter("splunk_siem", reporting.NewSplunkExporter())
	logStreamSvc.RegisterExporter("crowdstrike_siem", reporting.NewCrowdStrikeExporter())
	logStreamSvc.RegisterExporter("aws_s3", reporting.NewS3Exporter())
	logStreamSvc.RegisterExporter("gcs_bucket", reporting.NewGCSExporter())

	wg.Add(1)
	go func() {
		defer wg.Done()
		runLogStreamScheduler(runCtx, logStreamSvc, logger)
	}()

	// MCP authorization codes, SSO sessions, and refresh tokens are held in
	// memory and are only ever marked used/expired on the request path, never
	// deleted. Without this, quiet tenants and abandoned flows accumulate
	// entries forever.
	wg.Add(1)
	go func() {
		defer wg.Done()
		runExpirySweeper(runCtx, "mcp auth code", 5*time.Minute, mcpCodeStore.SweepExpired, logger)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		runExpirySweeper(runCtx, "mcp idp session", 5*time.Minute, mcpIdPService.SweepExpired, logger)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		runExpirySweeper(runCtx, "mcp refresh token", 5*time.Minute, mcpTokenService.SweepExpired, logger)
	}()

	// Cached downstream MCP HTTP clients (and the sessions they hold) are
	// evicted after mcpClientIdleTimeout of disuse, so tenants/servers that
	// have gone quiet don't hold an open session forever.
	wg.Add(1)
	go func() {
		defer wg.Done()
		runExpirySweeper(runCtx, "mcp gateway client cache", 5*time.Minute, gwClientFactory.SweepIdleClients, logger)
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
	encKey []byte,
	planLimiter *service.PlanLimiter,
	clientWorkloadSvc *resources.ClientWorkloadService,
	serverWorkloadSvc *resources.ServerWorkloadService,
	trustProviderSvc *resources.TrustProviderService,
	accessConditionSvc *resources.AccessConditionService,
	credentialProviderSvc *resources.CredentialProviderService,
	accessPolicySvc *resources.AccessPolicyService,
	authenticator rest.APIAuthenticator,
	registrationSvc *registration.RegistrationService,
	edgeAPI *rest.EdgeAPIHandler,
	mcpAuth *rest.MCPAuthServer,
	mcpGateway *rest.MCPGatewayServer,
	wellKnown *rest.WellKnownHandler,
	issuer *identity.IdentityIssuer,
	eventBus *eventbus.EventBus,
	caService *pki.TenantCAService,
	logger *slog.Logger,
) *http.Server {
	mux := http.NewServeMux()

	apiHandlers := rest.NewAPIHandlers(
		repos.AgentControllers,
		repos.AgentProxies,
		repos.APITokens,
		repos.TrustProviders,
		repos.Users,
		repos.Roles,
		repos.ResourceSets,
		repos.IdentityProviders,
		repos.SignOnPolicies,
		repos.LogStreams,
		repos.GlobalPolicyCompliance,
		repos.AccessPolicies,
		repos.TenantProfiles,
		registrationSvc,
		repos.Tenants,
	)
	apiHandlers.RegisterAPIRoutes(mux)
	apiHandlers.SetPlanLimiter(planLimiter)
	apiHandlers.SetHealthPinger(repos)

	apiHandlers.SetAudit(eventBus)

	cwHandler := rest.NewClientWorkloadHandler(clientWorkloadSvc)
	cwHandler.SetAudit(eventBus)
	cwHandler.RegisterRoutes(mux)

	swHandler := rest.NewServerWorkloadHandler(serverWorkloadSvc)
	swHandler.SetAudit(eventBus)
	swHandler.RegisterRoutes(mux)

	tpHandler := rest.NewTrustProviderHandler(trustProviderSvc)
	tpHandler.SetAudit(eventBus)
	tpHandler.RegisterRoutes(mux)

	acHandler := rest.NewAccessConditionHandler(accessConditionSvc)
	acHandler.SetAudit(eventBus)
	acHandler.RegisterRoutes(mux)

	cpHandler := rest.NewCredentialProviderHandler(credentialProviderSvc)
	cpHandler.SetAudit(eventBus)
	cpHandler.RegisterRoutes(mux)

	apHandler := rest.NewAccessPolicyHandler(accessPolicySvc)
	apHandler.SetAudit(eventBus)
	apHandler.RegisterRoutes(mux)
	rest.NewEventHandlers(repos.Events).RegisterEventRoutes(mux)
	rest.NewDashboardHandler(reporting.NewDashboardService(
		repos.ClientWorkloads, repos.ServerWorkloads, repos.TrustProviders,
		repos.CredentialProviders, repos.AccessPolicies, repos.AccessConditions, repos.Events,
	)).RegisterRoutes(mux)
	rest.NewTlsDecryptHandler(caService).RegisterRoutes(mux)
	rest.NewInternalHandlers(registrationSvc).RegisterRoutes(mux)

	mfaSvc := auth.NewMFAService(repos.UserMFA, encKey)
	mfaHandler := rest.NewMFAHandler(mfaSvc, repos.Users)
	mfaHandler.SetAudit(eventBus)
	mfaHandler.RegisterRoutes(mux)

	loginSvc := auth.NewLoginService(repos.Users, repos.Tenants, issuer)
	loginSvc.SetMFA(repos.UserMFA, encKey)
	loginSvc.SetRoles(repos.Roles)
	passwordResetSvc := auth.NewPasswordResetService(repos.Users, repos.PasswordResetTokens)
	emailSvc := service.NewEmailService(cfg.ResendAPIKey, cfg.EmailFrom)
	passwordResetSvc.SetEmail(emailSvc, cfg.ConsoleURL)

	feedbackHandler := rest.NewFeedbackHandler(cfg.ResendAPIKey, cfg.EmailFrom, "support@dipolehq.com")
	feedbackHandler.RegisterRoutes(mux)
	signupSvc := auth.NewSignupService(repos.Tenants, repos.Users, repos.Roles, repos.ResourceSets, repos.TenantProfiles, issuer)
	authHandler := rest.NewAuthHandler(loginSvc, passwordResetSvc, signupSvc)
	authHandler.SetAudit(eventBus)
	authHandler.RegisterRoutes(mux)

	if ssoTenantID, err := uuid.Parse(cfg.AdminSSOTenantID); err == nil && ssoTenantID != uuid.Nil {
		oidcClient := auth.NewHTTPOIDCClient()
		ssoSvc := auth.NewAdminSSOService(repos.Users, oidcClient)
		rest.NewSSOHandler(repos.IdentityProviders, ssoSvc, issuer, ssoTenantID).RegisterRoutes(mux)
	}

	edgeAPI.RegisterEdgeRoutes(mux)

	wellKnown.RegisterRoutes(mux)
	mcpAuth.RegisterRoutes(mux)

	mcpGateway.RegisterGatewayRoutes(mux)

	handler := rest.ApplyMiddleware(mux, authenticator, rest.ParseCORSOrigins(cfg.CORSAllowedOrigins), cfg.ProxySecret, repos.Users, repos.Roles, logger)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func runLogStreamScheduler(ctx context.Context, svc *reporting.LogStreamService, logger *slog.Logger) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	logger.Info("log stream export scheduler started", "interval", "60s")

	for {
		select {
		case <-ctx.Done():
			logger.Info("log stream export scheduler stopped")
			return
		case <-ticker.C:
			svc.ExportCycle(ctx)
		}
	}
}

// runExpirySweeper periodically invokes sweep with the current time until ctx
// is cancelled. Shared by the MCP in-memory stores (auth codes, IdP sessions,
// refresh tokens), which all need the same ticker/shutdown plumbing around a
// store-specific sweep call.
func runExpirySweeper(ctx context.Context, name string, interval time.Duration, sweep func(time.Time), logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info(name+" sweeper started", "interval", interval.String())

	for {
		select {
		case <-ctx.Done():
			logger.Info(name + " sweeper stopped")
			return
		case <-ticker.C:
			sweep(time.Now().UTC())
		}
	}
}

func shutdownHTTP(srv *http.Server, logger *slog.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http graceful shutdown failed", "error", err)
	}
}

type oidcDiscoveryAdapter struct {
	client *auth.HTTPOIDCClient
}

func (a *oidcDiscoveryAdapter) FetchDiscovery(ctx context.Context, discoveryURL string) (*mcpsvc.OIDCDiscovery, error) {
	doc, err := a.client.FetchDiscovery(ctx, discoveryURL)
	if err != nil {
		return nil, err
	}
	return &mcpsvc.OIDCDiscovery{
		AuthorizationEndpoint: doc.AuthorizationEndpoint,
		TokenEndpoint:         doc.TokenEndpoint,
		JWKSURI:               doc.JWKSURI,
		Issuer:                doc.Issuer,
	}, nil
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
