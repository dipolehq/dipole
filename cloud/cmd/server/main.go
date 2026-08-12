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
	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/dipolehq/dipole/cloud/internal/service/conditions"
	"github.com/dipolehq/dipole/cloud/internal/service/credentials"
	"github.com/dipolehq/dipole/cloud/internal/service/events"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
	mcpsvc "github.com/dipolehq/dipole/cloud/internal/service/mcp"
	mcpgw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
	mcpserver "github.com/dipolehq/dipole/cloud/internal/service/mcp_server"
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

	if err := devBootstrap(ctx, repos, cfg, logger); err != nil {
		return fmt.Errorf("dev bootstrap: %w", err)
	}

	tracingBus := mcpgw.NewTracingBus()

	platformHosts := platform.NewHosts(cfg.PlatformDomain, cfg.Env)

	issuer, err := identity.NewIdentityIssuer("https://id." + platformHosts.Domain())
	if err != nil {
		return fmt.Errorf("init identity issuer: %w", err)
	}

	broker := credentials.NewCredentialBroker(encKey, issuer, repos.Tenants, platformHosts)
	trustValidator := trust.NewConfiguredService(logger, repos.AgentControllers)
	conditionEval := conditions.NewConditionEvaluator(cfg.GeoIPDBPath)
	authEmitter := events.NewAuthEventEmitter(repos.Events)
	policyService := policy.NewPolicyEvaluationService(repos.AccessPolicies, trustValidator, conditionEval, broker, authEmitter, logger)

	clientWorkloadSvc := service.NewClientWorkloadService(repos.ClientWorkloads)
	serverWorkloadSvc := service.NewServerWorkloadService(repos.ServerWorkloads)
	trustProviderSvc := service.NewTrustProviderService(repos.TrustProviders, repos.Tenants, cfg.Region, platformHosts)
	accessConditionSvc := service.NewAccessConditionService(repos.AccessConditions)
	credentialProviderSvc := service.NewCredentialProviderService(repos.CredentialProviders, encKey)
	accessPolicySvc := service.NewAccessPolicyService(repos.AccessPolicies)
	authenticator := service.NewAPITokenAuthenticator(repos.Tenants)
	authenticator.SetIssuer(issuer)
	authenticator.SetAPITokenRepo(repos.APITokens)
	registrationSvc := registration.NewRegistrationService(repos.AgentControllers, repos.AgentProxies, repos.RegistrationTokens)
	caService := pki.NewTenantCAService(repos.TenantCAs, encKey)
	workloadIngester := events.NewWorkloadEventIngester(repos.Events)

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
	mcpIdPResolver := mcpsvc.NewDBIdentityProviderResolver(repos.IdentityProviders, &oidcDiscoveryAdapter{client: service.NewHTTPOIDCClient()})
	mcpIdPService := mcpsvc.NewIdPService(mcpIdPResolver)
	mcpIdPCallback := mcpsvc.NewIdPCallbackService(mcpIdPService, mcpCodeStore, trustValidator, conditionEval)
	mcpTokenConfigResolver := mcpsvc.NewPolicyTokenConfigResolver(repos.AccessPolicies, repos.CredentialProviders, encKey)
	mcpTokenService := mcpsvc.NewTokenService(mcpCodeStore, issuer, mcpTokenConfigResolver)
	mcpAuthorizeService := mcpsvc.NewAuthorizeService(mcpDCR, mcpCodeStore, mcpPolicyFinder, conditionEval, mcpIdPService)

	mcpAuthServer := rest.NewMCPAuthServer(mcpDCR, mcpAuthorizeService, mcpTokenService, mcpIdPCallback, tenantHostResolver, logger)

	wellKnownHandler := rest.NewWellKnownHandler(issuer, tenantHostResolver, logger)

	mcpAuditLogger := mcpserver.NewAuditLogger(repos.Events)
	mcpToolService := mcpserver.NewToolService(repos.Events, mcpAuditLogger)
	mcpResourceService := mcpserver.NewResourceService()
	mcpDipoleServer := rest.NewDipoleMCPServer(mcpToolService, mcpResourceService, authenticator, logger)

	gwLeftAuth := mcpgw.NewLeftSideAuth(issuer)
	gwPolicyFinder := mcpgw.NewDBRightSidePolicyFinder(repos.AccessPolicies)
	gwTokenStore := repos.MCPUserCredentials
	gwDecrypter := mcpgw.NewAESCredentialDecrypter(encKey)
	gwRightAuth := mcpgw.NewRightSideAuth(gwPolicyFinder, gwTokenStore, nil, gwDecrypter)
	gwEventSink := mcpgw.NewDBEventSink(repos.Events, logger)
	gwEventEmitter := mcpgw.NewGatewayEventEmitter(gwEventSink)
	gwClientFactory := mcpgw.NewHTTPMCPClientFactory()
	gwMethodHandler := mcpgw.NewMethodHandler(gwRightAuth, gwClientFactory, gwEventEmitter)

	consentConfigResolver := mcpgw.NewCPConsentConfigResolver(repos.AccessPolicies, repos.CredentialProviders, encKey)
	consentCallback := mcpgw.NewConsentCallbackService(gwTokenStore, consentConfigResolver)

	mcpGatewayServer := rest.NewMCPGatewayServer(gwLeftAuth, gwMethodHandler, consentCallback, logger, tracingBus, tenantHostResolver)

	logger.Info("MCP services initialized",
		"platform_domain", platformHosts.Domain(),
		"auth_server_pattern", "https://<tenant>-mcp."+platformHosts.Domain(),
		"identity_issuer_pattern", "https://<tenant>-id."+platformHosts.Domain(),
		"gateway_pattern", "https://<tenant>-mcpgateway."+platformHosts.Domain(),
	)

	httpServer := buildHTTPServer(cfg, repos, encKey, planLimiter, clientWorkloadSvc, serverWorkloadSvc, trustProviderSvc, accessConditionSvc, credentialProviderSvc, accessPolicySvc, authenticator, registrationSvc, edgeAPI, mcpAuthServer, mcpGatewayServer, mcpDipoleServer, wellKnownHandler, issuer, tracingBus)
	grpcHandler := grpctransport.NewPolicyEvaluationServer(policyService, registrationSvc, repos.AgentProxies, caService, workloadIngester, logger)

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
		if err := grpctransport.StartServer(runCtx, cfg.GRPCPort, grpcHandler, cfg.GRPCTLSCert, cfg.GRPCTLSKey, logger); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	logStreamSvc := service.NewLogStreamService(repos.Events, repos.LogStreams, logger)
	logStreamSvc.RegisterExporter("splunk_siem", service.NewSplunkExporter())
	logStreamSvc.RegisterExporter("crowdstrike_siem", service.NewCrowdStrikeExporter())
	logStreamSvc.RegisterExporter("aws_s3", &service.S3Exporter{})
	logStreamSvc.RegisterExporter("gcs_bucket", &service.GCSExporter{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		runLogStreamScheduler(runCtx, logStreamSvc, logger)
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
	clientWorkloadSvc *service.ClientWorkloadService,
	serverWorkloadSvc *service.ServerWorkloadService,
	trustProviderSvc *service.TrustProviderService,
	accessConditionSvc *service.AccessConditionService,
	credentialProviderSvc *service.CredentialProviderService,
	accessPolicySvc *service.AccessPolicyService,
	authenticator rest.APIAuthenticator,
	registrationSvc *registration.RegistrationService,
	edgeAPI *rest.EdgeAPIHandler,
	mcpAuth *rest.MCPAuthServer,
	mcpGateway *rest.MCPGatewayServer,
	mcpDipole *rest.DipoleMCPServer,
	wellKnown *rest.WellKnownHandler,
	issuer *identity.IdentityIssuer,
	tracingBus *mcpgw.TracingBus,
) *http.Server {
	mux := http.NewServeMux()

	apiHandlers := rest.NewAPIHandlers(
		repos.AgentControllers,
		repos.AgentProxies,
		repos.APITokens,
		repos.Users,
		repos.Roles,
		repos.ResourceSets,
		repos.IdentityProviders,
		repos.SignOnPolicies,
		repos.LogStreams,
		repos.GlobalPolicyCompliance,
		service.NewComplianceService(repos.AccessPolicies),
		repos.TenantProfiles,
		registrationSvc,
		repos.Tenants,
	)
	apiHandlers.RegisterAPIRoutes(mux)
	apiHandlers.SetPlanLimiter(planLimiter)
	apiHandlers.SetHealthPinger(repos)

	auditEmitter := events.NewAuditLogEmitter(repos.Events)
	apiHandlers.SetAudit(auditEmitter)

	cwHandler := rest.NewClientWorkloadHandler(clientWorkloadSvc)
	cwHandler.SetAudit(auditEmitter)
	cwHandler.RegisterRoutes(mux)

	swHandler := rest.NewServerWorkloadHandler(serverWorkloadSvc)
	swHandler.SetAudit(auditEmitter)
	swHandler.RegisterRoutes(mux)

	tpHandler := rest.NewTrustProviderHandler(trustProviderSvc)
	tpHandler.SetAudit(auditEmitter)
	tpHandler.RegisterRoutes(mux)

	acHandler := rest.NewAccessConditionHandler(accessConditionSvc)
	acHandler.SetAudit(auditEmitter)
	acHandler.RegisterRoutes(mux)

	cpHandler := rest.NewCredentialProviderHandler(credentialProviderSvc)
	cpHandler.SetAudit(auditEmitter)
	cpHandler.RegisterRoutes(mux)

	apHandler := rest.NewAccessPolicyHandler(accessPolicySvc)
	apHandler.SetAudit(auditEmitter)
	apHandler.RegisterRoutes(mux)
	rest.NewEventHandlers(repos.Events).RegisterEventRoutes(mux)
	rest.NewMCPTracingHandler(tracingBus).RegisterRoutes(mux)
	rest.NewDashboardHandler(service.NewDashboardService(
		repos.ClientWorkloads, repos.ServerWorkloads, repos.TrustProviders,
		repos.CredentialProviders, repos.AccessPolicies, repos.AccessConditions, repos.Events,
	)).RegisterRoutes(mux)
	rest.NewInternalHandlers(registrationSvc).RegisterRoutes(mux)

	mfaSvc := service.NewMFAService(repos.UserMFA, encKey)
	mfaHandler := rest.NewMFAHandler(mfaSvc, repos.Users)
	mfaHandler.SetAudit(auditEmitter)
	mfaHandler.RegisterRoutes(mux)

	loginSvc := service.NewLoginService(repos.Users, repos.Tenants, issuer)
	loginSvc.SetMFA(repos.UserMFA, encKey)
	passwordResetSvc := service.NewPasswordResetService(repos.Users, repos.PasswordResetTokens)
	emailSvc := service.NewEmailService(cfg.ResendAPIKey, cfg.EmailFrom)
	passwordResetSvc.SetEmail(emailSvc, cfg.ConsoleURL)
	signupSvc := service.NewSignupService(repos.Tenants, repos.Users, repos.ResourceSets, issuer)
	authHandler := rest.NewAuthHandler(loginSvc, passwordResetSvc, signupSvc)
	authHandler.SetAudit(auditEmitter)
	authHandler.RegisterRoutes(mux)

	if ssoTenantID, err := uuid.Parse(cfg.AdminSSOTenantID); err == nil && ssoTenantID != uuid.Nil {
		oidcClient := service.NewHTTPOIDCClient()
		ssoSvc := service.NewAdminSSOService(repos.Users, oidcClient)
		rest.NewSSOHandler(repos.IdentityProviders, ssoSvc, issuer, ssoTenantID).RegisterRoutes(mux)
	}

	edgeAPI.RegisterEdgeRoutes(mux)

	wellKnown.RegisterRoutes(mux)
	mcpAuth.RegisterRoutes(mux)

	mcpGateway.RegisterGatewayRoutes(mux)
	if mcpDipole != nil {
		mcpDipole.RegisterMCPServerRoutes(mux)
	}

	handler := rest.ApplyMiddleware(mux, authenticator, rest.ParseCORSOrigins(cfg.CORSAllowedOrigins), cfg.ProxySecret)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func runLogStreamScheduler(ctx context.Context, svc *service.LogStreamService, logger *slog.Logger) {
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

func shutdownHTTP(srv *http.Server, logger *slog.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http graceful shutdown failed", "error", err)
	}
}

type oidcDiscoveryAdapter struct {
	client *service.HTTPOIDCClient
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
