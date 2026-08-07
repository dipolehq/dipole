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

	// Single source of truth for per-tenant service hostnames.
	platformHosts := platform.NewHosts(cfg.PlatformDomain)

	// One shared signing key for all tenants; the per-tenant `iss` is supplied
	// per issuance request. The platform issuer URL below is only used for
	// tokens we verify ourselves (admin sessions, MFA challenges, Edge API
	// tokens), so nothing fetches OIDC discovery from it.
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
	// Edge SDK Client ID / Audience must embed the subdomain of the tenant that
	// owns the Trust Provider, so the service resolves it per call rather than
	// capturing one subdomain at startup.
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

	// Resolves which tenant a request belongs to from its hostname. Shared by
	// every component that serves per-tenant hostnames.
	tenantHostResolver := rest.NewTenantHostResolver(repos.Tenants, platformHosts, cfg.DevBootstrap)
	if cfg.DevBootstrap {
		logger.Warn("dev bootstrap enabled: tenant may be selected via header — never enable this in a deployed environment",
			"header", rest.DevTenantHeader)
	}

	// MCP Authorization Server — tenant-agnostic, serves every tenant. Each
	// request resolves its own tenant from the hostname it arrived on.
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

	// Discovery endpoints are shared between the -mcp and -id hostnames, so they
	// are owned by neither service directly.
	wellKnownHandler := rest.NewWellKnownHandler(issuer, tenantHostResolver, logger)

	// Dipole MCP Server (read-only event query API for AI agents) resolves its
	// tenant from the caller's API token, so it needs no hostname binding.
	mcpAuditLogger := mcpserver.NewAuditLogger(repos.Events)
	mcpToolService := mcpserver.NewToolService(repos.Events, mcpAuditLogger)
	mcpResourceService := mcpserver.NewResourceService()
	mcpDipoleServer := rest.NewDipoleMCPServer(mcpToolService, mcpResourceService, authenticator, logger)

	// MCP Identity Gateway — still binds ONE tenant at construction, so it is
	// only wired up when that tenant is supplied explicitly. Its downstream MCP
	// client map, event sink, and token audience are all built once at startup;
	// making them per-request is progress.md Fix 11. Until then the Gateway is
	// unavailable rather than silently serving every tenant as one tenant.
	var mcpGatewayServer *rest.MCPGatewayServer
	if gatewayTenantID, err := uuid.Parse(cfg.GatewayTenantID); err == nil && platform.IsValidSubdomain(cfg.GatewayTenantSubdomain) {
		// Guarded above: an invalid subdomain would make MCPGatewayURL return
		// "", leaving gatewayURL as bare "/" — which LeftSideAuth would then use
		// as the expected token audience, silently accepting or rejecting the
		// wrong tokens.
		gatewayURL := platformHosts.MCPGatewayURL(cfg.GatewayTenantSubdomain) + "/"

		gwLeftAuth := mcpgw.NewLeftSideAuth(issuer, gatewayURL)
		gwPolicyFinder := mcpgw.NewDBRightSidePolicyFinder(repos.AccessPolicies)
		gwTokenStore := repos.MCPUserCredentials
		gwRightAuth := mcpgw.NewRightSideAuth(gwPolicyFinder, gwTokenStore, nil, gatewayURL)
		gwEventSink := mcpgw.NewDBEventSink(repos.Events, gatewayTenantID, logger)
		gwEventEmitter := mcpgw.NewGatewayEventEmitter(gwEventSink)
		gwClients := buildMCPServerClients(ctx, repos, gatewayTenantID, logger)
		gwMethodHandler := mcpgw.NewMethodHandler(gwRightAuth, gwClients, gatewayTenantID, gwEventEmitter)

		consentConfigResolver := mcpgw.NewCPConsentConfigResolver(repos.AccessPolicies, repos.CredentialProviders, encKey, gatewayURL)
		consentCallback := mcpgw.NewConsentCallbackService(gwTokenStore, consentConfigResolver)

		mcpGatewayServer = rest.NewMCPGatewayServer(gwLeftAuth, gwMethodHandler, consentCallback, logger, tracingBus)
		logger.Warn("MCP Identity Gateway is bound to a single tenant at startup (pending Fix 11) — it will serve ALL gateway requests as this tenant",
			"tenant_id", gatewayTenantID, "gateway_url", gatewayURL)
	} else {
		logger.Info("MCP Identity Gateway not configured (requires DIPOLE_TENANT_ID + a DNS-valid DIPOLE_TENANT_SUBDOMAIN, until Fix 11 makes it multi-tenant)",
			"tenant_id", cfg.GatewayTenantID, "tenant_subdomain", cfg.GatewayTenantSubdomain)
	}

	// Built as literal patterns rather than via platformHosts: the URL builders
	// validate their input and would (correctly) return "" for a placeholder
	// like "<tenant>".
	logger.Info("MCP services initialized",
		"platform_domain", platformHosts.Domain(),
		"auth_server_pattern", "https://<tenant>-mcp."+platformHosts.Domain(),
		"identity_issuer_pattern", "https://<tenant>-id."+platformHosts.Domain(),
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

	// MFAService is stateless w.r.t. tenant (resolves tenantID per-call from
	// the caller's JWT), so it's constructed unconditionally — MFA works
	// for every tenant, not just the one from DIPOLE_TENANT_ID.
	mfaSvc := service.NewMFAService(repos.UserMFA, encKey)
	mfaHandler := rest.NewMFAHandler(mfaSvc, repos.Users)
	mfaHandler.SetAudit(auditEmitter)
	mfaHandler.RegisterRoutes(mux)

	loginSvc := service.NewLoginService(repos.Users, repos.Tenants, issuer)
	loginSvc.SetMFA(repos.UserMFA, encKey)
	passwordResetSvc := service.NewPasswordResetService(repos.Users, repos.PasswordResetTokens)
	emailSvc := service.NewEmailService(cfg.ResendAPIKey, cfg.EmailFrom)
	// Per-environment: a hardcoded value sends staging users to production.
	passwordResetSvc.SetEmail(emailSvc, cfg.ConsoleURL)
	signupSvc := service.NewSignupService(repos.Tenants, repos.Users, repos.ResourceSets, issuer)
	authHandler := rest.NewAuthHandler(loginSvc, passwordResetSvc, signupSvc)
	authHandler.SetAudit(auditEmitter)
	authHandler.RegisterRoutes(mux)

	// Admin SSO login is still bound to a single tenant at construction
	// (progress.md Fix 10/B7, not yet fixed — it needs a design decision).
	//
	// It can't use either mechanism the rest of the codebase now uses: it runs
	// BEFORE authentication so there's no JWT to read a tenant from, and the
	// admin console is served from one shared hostname rather than a per-tenant
	// subdomain, so the hostname can't supply it either. Listing a tenant's SSO
	// providers on the login page has the same problem. Left exactly as it was
	// (single-tenant) rather than papered over — see Fix 10/B7 for the options.
	if ssoTenantID, err := uuid.Parse(cfg.GatewayTenantID); err == nil && ssoTenantID != uuid.Nil {
		oidcClient := service.NewHTTPOIDCClient()
		ssoSvc := service.NewAdminSSOService(repos.Users, oidcClient)
		rest.NewSSOHandler(repos.IdentityProviders, ssoSvc, issuer, ssoTenantID).RegisterRoutes(mux)
	}

	edgeAPI.RegisterEdgeRoutes(mux)

	// Discovery endpoints are always served: external systems (AWS STS, Azure
	// Entra, GCP WIF) fetch them to verify tokens we sign, independently of
	// whether MCP is in use.
	wellKnown.RegisterRoutes(mux)
	mcpAuth.RegisterRoutes(mux)

	if mcpGateway != nil {
		mcpGateway.RegisterGatewayRoutes(mux)
	}
	if mcpDipole != nil {
		mcpDipole.RegisterMCPServerRoutes(mux)
	}

	handler := applyMiddleware(mux, authenticator, parseCORSOrigins(cfg.CORSAllowedOrigins))

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func applyMiddleware(mux *http.ServeMux, authenticator rest.APIAuthenticator, allowedOrigins []string) http.Handler {
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

	return rest.RequestIDMiddleware(rest.CORSMiddleware(allowedOrigins)(rest.ResourceSetMiddleware(routed)))
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

func parseCORSOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
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

// devBootstrap creates a tenant, an admin user, and an API token so a fresh
// local database is immediately usable.
//
// It is gated behind DIPOLE_DEV_BOOTSTRAP and does nothing otherwise. It used
// to run in EVERY environment, which is how staging ended up with a phantom
// "dev" tenant: with no tenant env vars set it invented one, wrote it back into
// the config, and every MCP service then bound to it — so requests were served
// as a tenant that contained none of the real customer's configuration.
//
// It also creates a well-known password, which must never exist in a deployed
// environment.
func devBootstrap(ctx context.Context, repos *postgres.Repositories, cfg *config.Config, logger *slog.Logger) error {
	if !cfg.DevBootstrap {
		return nil
	}

	const devSubdomain = "dev"

	existing, err := repos.Tenants.GetBySubdomain(ctx, devSubdomain)
	if err != nil {
		return fmt.Errorf("lookup dev tenant: %w", err)
	}
	if existing != nil {
		logger.Info("dev tenant already present", "id", existing.ID, "subdomain", existing.Subdomain)
		return nil
	}

	token, err := service.GenerateAPIToken()
	if err != nil {
		return fmt.Errorf("generate api token: %w", err)
	}

	tenant := &domain.Tenant{
		Name:         "Dev Tenant",
		Subdomain:    devSubdomain,
		APITokenHash: service.HashAPIToken(token),
	}
	if err := repos.Tenants.Create(ctx, tenant); err != nil {
		return fmt.Errorf("create dev tenant: %w", err)
	}

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
		return fmt.Errorf("create dev admin user: %w", err)
	}

	logger.Warn("DEV BOOTSTRAP: created tenant with a well-known password — local development only",
		"tenant_id", tenant.ID,
		"subdomain", devSubdomain,
		"email", "admin@dipole.dev",
		"password", "admin123",
		"api_token", token,
	)

	return nil
}
