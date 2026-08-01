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

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/config"
	"github.com/dipolehq/dipole/cloud/internal/domain"
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
	if migrator, merr := postgres.NewMigrator(cfg.DatabaseURL); merr != nil {
		logger.Warn("skip auto-migration", "error", merr)
	} else {
		if merr = migrator.Up(); merr != nil {
			logger.Warn("auto-migration issue", "error", merr)
		} else {
			logger.Info("schema up to date")
		}
		_ = migrator.Close()
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connected")

	repos := postgres.NewRepositories(pool)
	repos.SetMCPEncryptionKey(encKey)

	if err := autoBootstrap(ctx, repos, cfg, logger); err != nil {
		return fmt.Errorf("auto-bootstrap: %w", err)
	}

	tracingBus := mcpgw.NewTracingBus()

	issuerSubdomain := cfg.TenantSubdomain
	if issuerSubdomain == "" {
		issuerSubdomain = "dipole"
	}
	issuer, err := identity.NewIdentityIssuer(issuerSubdomain, cfg.Region)
	if err != nil {
		return fmt.Errorf("init identity issuer: %w", err)
	}

	broker := credentials.NewCredentialBroker(encKey, issuer)
	trustValidator := trust.NewConfiguredService(logger, repos.AgentControllers)
	conditionEval := conditions.NewConditionEvaluator(cfg.GeoIPDBPath)
	authEmitter := events.NewAuthEventEmitter(repos.Events)
	policyService := policy.NewPolicyEvaluationService(repos.AccessPolicies, trustValidator, conditionEval, broker, authEmitter, logger)

	clientWorkloadSvc := service.NewClientWorkloadService(repos.ClientWorkloads)
	serverWorkloadSvc := service.NewServerWorkloadService(repos.ServerWorkloads)
	trustProviderSvc := service.NewTrustProviderService(repos.TrustProviders, cfg.Region, cfg.TenantSubdomain)
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

	var mcpAuthServer *rest.MCPAuthServer
	var mcpGatewayServer *rest.MCPGatewayServer
	var mcpDipoleServer *rest.DipoleMCPServer

	if cfg.TenantID != "" && cfg.TenantSubdomain != "" {
		tenantUUID, _ := uuid.Parse(cfg.TenantID)

		mcpAuthServerURL := "https://" + cfg.TenantSubdomain + ".mcp." + cfg.Region + ".dipolehq.com"
		gatewayURL := "https://" + cfg.TenantSubdomain + ".mcpgateway." + cfg.Region + ".dipolehq.com/"

		mcpCodeStore := mcpsvc.NewAuthCodeStore()
		mcpDCR := mcpsvc.NewDCRService(repos.MCPClients)
		mcpPolicyFinder := mcpsvc.NewMCPPolicyFinder(repos.AccessPolicies)
		mcpIdPService := mcpsvc.NewIdPService(mcpAuthServerURL)
		mcpIdPCallback := mcpsvc.NewIdPCallbackService(mcpIdPService, mcpCodeStore, trustValidator, conditionEval)
		mcpTokenConfigResolver := mcpsvc.NewPolicyTokenConfigResolver(repos.AccessPolicies, repos.CredentialProviders, encKey)
		mcpTokenService := mcpsvc.NewTokenService(mcpCodeStore, issuer, mcpTokenConfigResolver)
		mcpAuthorizeService := mcpsvc.NewAuthorizeService(mcpDCR, mcpCodeStore, mcpPolicyFinder, conditionEval, mcpIdPService)

		gwLeftAuth := mcpgw.NewLeftSideAuth(issuer, gatewayURL)
		gwPolicyFinder := mcpgw.NewDBRightSidePolicyFinder(repos.AccessPolicies)
		gwTokenStore := repos.MCPUserCredentials
		gwRightAuth := mcpgw.NewRightSideAuth(gwPolicyFinder, gwTokenStore, nil, gatewayURL)
		gwEventSink := mcpgw.NewDBEventSink(repos.Events, tenantUUID, logger)
		gwEventEmitter := mcpgw.NewGatewayEventEmitter(gwEventSink)
		gwClients := buildMCPServerClients(ctx, repos, tenantUUID, logger)
		gwMethodHandler := mcpgw.NewMethodHandler(gwRightAuth, gwClients, tenantUUID, gwEventEmitter)

		consentConfigResolver := mcpgw.NewCPConsentConfigResolver(repos.AccessPolicies, repos.CredentialProviders, encKey, gatewayURL)
		consentCallback := mcpgw.NewConsentCallbackService(gwTokenStore, consentConfigResolver)

		mcpAuditLogger := mcpserver.NewAuditLogger(repos.Events)
		mcpToolService := mcpserver.NewToolService(repos.Events, mcpAuditLogger)
		mcpResourceService := mcpserver.NewResourceService()

		mcpAuthConfig := rest.MCPAuthConfig{
			TenantID:        tenantUUID,
			TenantSubdomain: cfg.TenantSubdomain,
			Region:          cfg.Region,
			Domain:          "dipolehq.com",
		}
		mcpAuthServer = rest.NewMCPAuthServer(issuer, mcpDCR, mcpAuthorizeService, mcpTokenService, mcpIdPCallback, mcpAuthConfig, logger)
		mcpGatewayServer = rest.NewMCPGatewayServer(gwLeftAuth, gwMethodHandler, consentCallback, logger, tracingBus)
		mcpDipoleServer = rest.NewDipoleMCPServer(mcpToolService, mcpResourceService, authenticator, logger)

		logger.Info("MCP services initialized",
			"auth_server", mcpAuthServerURL,
			"gateway", gatewayURL,
		)
	}

	httpServer := buildHTTPServer(cfg, repos, encKey, planLimiter, clientWorkloadSvc, serverWorkloadSvc, trustProviderSvc, accessConditionSvc, credentialProviderSvc, accessPolicySvc, authenticator, registrationSvc, edgeAPI, mcpAuthServer, mcpGatewayServer, mcpDipoleServer, issuer, tracingBus)
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
	issuer rest.OIDCIssuer,
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
		repos.TenantProfiles,
		registrationSvc,
		repos.Tenants,
	)
	apiHandlers.RegisterAPIRoutes(mux)
	apiHandlers.SetPlanLimiter(planLimiter)

	rest.NewClientWorkloadHandler(clientWorkloadSvc).RegisterRoutes(mux)
	rest.NewServerWorkloadHandler(serverWorkloadSvc).RegisterRoutes(mux)
	rest.NewTrustProviderHandler(trustProviderSvc).RegisterRoutes(mux)
	rest.NewAccessConditionHandler(accessConditionSvc).RegisterRoutes(mux)
	rest.NewCredentialProviderHandler(credentialProviderSvc).RegisterRoutes(mux)
	rest.NewAccessPolicyHandler(accessPolicySvc).RegisterRoutes(mux)
	rest.NewEventHandlers(repos.Events).RegisterEventRoutes(mux)
	rest.NewMCPTracingHandler(tracingBus).RegisterRoutes(mux)
	rest.NewDashboardHandler(service.NewDashboardService(
		repos.ClientWorkloads, repos.ServerWorkloads, repos.TrustProviders,
		repos.CredentialProviders, repos.AccessPolicies, repos.AccessConditions, repos.Events,
	)).RegisterRoutes(mux)
	rest.NewInternalHandlers(registrationSvc).RegisterRoutes(mux)

	if cfg.TenantID != "" {
		mfaTenantID, _ := uuid.Parse(cfg.TenantID)
		if mfaTenantID != uuid.Nil {
			mfaSvc := service.NewMFAService(repos.UserMFA, encKey, mfaTenantID)
			rest.NewMFAHandler(mfaSvc, repos.Users).RegisterRoutes(mux)
		}
	}

	if cfg.TenantID != "" {
		loginTenantID, _ := uuid.Parse(cfg.TenantID)
		if loginTenantID != uuid.Nil {
			loginSvc := service.NewLoginService(repos.Users, repos.Tenants, issuer.(*identity.IdentityIssuer), loginTenantID)
			loginSvc.SetMFA(repos.UserMFA, encKey)
			passwordResetSvc := service.NewPasswordResetService(repos.Users, repos.PasswordResetTokens, loginTenantID)
			emailSvc := service.NewEmailService(cfg.ResendAPIKey, cfg.EmailFrom)
			passwordResetSvc.SetEmail(emailSvc, "https://app.dipolehq.com")
			signupSvc := service.NewSignupService(repos.Tenants, repos.Users, repos.ResourceSets, issuer.(*identity.IdentityIssuer))
			rest.NewAuthHandler(loginSvc, passwordResetSvc, signupSvc).RegisterRoutes(mux)

			oidcClient := service.NewHTTPOIDCClient()
			ssoSvc := service.NewAdminSSOService(repos.Users, oidcClient)
			rest.NewSSOHandler(repos.IdentityProviders, ssoSvc, issuer.(*identity.IdentityIssuer), loginTenantID).RegisterRoutes(mux)
		}
	}

	edgeAPI.RegisterEdgeRoutes(mux)

	if mcpAuth != nil {
		mcpAuth.RegisterRoutes(mux)
	} else {
		rest.NewOIDCHandler(issuer).RegisterRoutes(mux)
	}
	if mcpGateway != nil {
		mcpGateway.RegisterGatewayRoutes(mux)
	}
	if mcpDipole != nil {
		mcpDipole.RegisterMCPServerRoutes(mux)
	}

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
	if path == "/api/v1/health" || path == "/api/v1/auth/login" || path == "/api/v1/auth/signup" || path == "/api/v1/auth/forgot-password" || path == "/api/v1/auth/reset-password" || path == "/api/v1/auth/mfa-verify" || strings.HasPrefix(path, "/api/v1/auth/sso/") || strings.HasPrefix(path, "/internal/") {
		return true
	}
	if strings.HasPrefix(path, "/edge/") {
		return true
	}
	if strings.HasPrefix(path, "/.well-known/") ||
		strings.HasPrefix(path, "/register") ||
		strings.HasPrefix(path, "/connect/") ||
		strings.HasPrefix(path, "/callback/") ||
		strings.HasPrefix(path, "/mcp") {
		return true
	}
	return false
}

func buildMCPServerClients(ctx context.Context, repos *postgres.Repositories, tenantID uuid.UUID, logger *slog.Logger) map[string]mcpgw.MCPServerClient {
	clients := make(map[string]mcpgw.MCPServerClient)

	workloads, _, err := repos.ServerWorkloads.List(ctx, tenantID, domain.ListOptions{Page: 1, PerPage: 100})
	if err != nil {
		logger.Warn("failed to load MCP server workloads", "error", err)
		return clients
	}

	for _, sw := range workloads {
		if sw.AppProtocol != domain.AppProtocolMCP {
			continue
		}
		clients[sw.Name] = mcpgw.NewHTTPMCPClient(sw.Host, sw.Port, sw.URLPath, sw.TLS)
	}

	if len(clients) > 0 {
		logger.Info("loaded MCP server clients", "count", len(clients))
	}

	return clients
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

func autoBootstrap(ctx context.Context, repos *postgres.Repositories, cfg *config.Config, logger *slog.Logger) error {
	if cfg.TenantID != "" {
		if _, err := uuid.Parse(cfg.TenantID); err != nil {
			return fmt.Errorf("invalid DIPOLE_TENANT_ID %q: %w", cfg.TenantID, err)
		}
		return nil
	}

	subdomain := cfg.TenantSubdomain
	if subdomain == "" {
		subdomain = "dev"
	}

	existing, err := repos.Tenants.GetBySubdomain(ctx, subdomain)
	if err != nil {
		return fmt.Errorf("lookup tenant: %w", err)
	}

	if existing != nil {
		cfg.TenantID = existing.ID.String()
		cfg.TenantSubdomain = existing.Subdomain
		logger.Info("tenant resolved from database", "id", existing.ID, "subdomain", existing.Subdomain)
		return nil
	}

	token, err := service.GenerateAPIToken()
	if err != nil {
		return fmt.Errorf("generate api token: %w", err)
	}
	hash := service.HashAPIToken(token)

	tenant := &domain.Tenant{Name: "Dev Tenant", Subdomain: subdomain, APITokenHash: hash}
	if err := repos.Tenants.Create(ctx, tenant); err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}

	cfg.TenantID = tenant.ID.String()
	cfg.TenantSubdomain = subdomain
	logger.Info("dev tenant auto-created", "id", tenant.ID, "subdomain", subdomain)

	passwordHash, err := service.HashPassword("admin123")
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	adminUser := &domain.User{
		Email:        "admin@dipole.dev",
		Name:         "Admin",
		FirstName:    "Admin",
		IsActive:     true,
		PasswordHash: passwordHash,
	}
	if err := repos.Users.Create(ctx, tenant.ID, adminUser); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	logger.Info("admin user auto-created", "email", "admin@dipole.dev", "password", "admin123")
	logger.Info("API token (save this)", "token", token)

	return nil
}
