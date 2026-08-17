/*
policy_handler.go implements the gRPC PolicyEvaluationService RPCs.

PolicyEvaluationServer is a thin transport layer: it maps protobuf messages
to domain types, delegates to the policy evaluator, and maps results back.
No business logic lives here. Register delegates to the registration service
and returns the per-tenant CA from the CA provider. ReportHealth delegates to
the agent proxy repository. IngestEvents persists workload events via the
event ingester (which handles header redaction before storage).
*/
package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/service/pki"
	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

// PolicyEvaluator abstracts the policy evaluation service.
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, tenantID uuid.UUID, evidence *domain.ClientEvidence, target *domain.ServerTarget) (*domain.EvaluationResult, error)
}

// ProxyRegistrar abstracts the registration service for proxy registration.
// It returns the newly registered proxy plus a bearer session token the
// proxy must present (via gRPC metadata) on every later call.
type ProxyRegistrar interface {
	RegisterProxy(ctx context.Context, tenantID uuid.UUID, registrationToken string) (*domain.AgentProxy, string, error)
}

// ProxyHealthUpdater abstracts the repository for updating proxy health.
type ProxyHealthUpdater interface {
	UpdateHealth(ctx context.Context, tenantID uuid.UUID, proxyID uuid.UUID, status string) error
}

// TenantCAProvider abstracts the PKI service for per-tenant CA generation.
type TenantCAProvider interface {
	GetOrCreateTenantCA(ctx context.Context, tenantID uuid.UUID) (certPEM string, keyPEM string, err error)
}

// LeafCertSigner abstracts the PKI service for signing leaf certificate CSRs.
type LeafCertSigner interface {
	SignCSR(ctx context.Context, req pki.SignCSRRequest) (*pki.SignCSRResponse, error)
	IsTlsDecryptEnabled(ctx context.Context, tenantID uuid.UUID) bool
}

// HostnameAuthorizer checks whether a hostname is authorized for TLS interception.
type HostnameAuthorizer interface {
	IsHostnameAuthorizedForTLS(ctx context.Context, tenantID uuid.UUID, hostname string) (bool, error)
}

type PolicyEvaluationServer struct {
	pb.UnimplementedPolicyEvaluationServiceServer
	evaluator     PolicyEvaluator
	registrar     ProxyRegistrar
	healthUpdater ProxyHealthUpdater
	caProvider    TenantCAProvider
	leafSigner    LeafCertSigner
	hostnameAuthz HostnameAuthorizer
	events        *events.EventBus
	logger        *slog.Logger
}

func NewPolicyEvaluationServer(evaluator PolicyEvaluator, registrar ProxyRegistrar, healthUpdater ProxyHealthUpdater, caProvider TenantCAProvider, eventBus *events.EventBus, logger *slog.Logger) *PolicyEvaluationServer {
	return &PolicyEvaluationServer{
		evaluator:     evaluator,
		registrar:     registrar,
		healthUpdater: healthUpdater,
		caProvider:    caProvider,
		events:        eventBus,
		logger:        logger,
	}
}

// SetLeafSigner configures the leaf certificate signing service.
func (s *PolicyEvaluationServer) SetLeafSigner(signer LeafCertSigner) {
	s.leafSigner = signer
}

// SetHostnameAuthorizer configures the hostname authorization checker.
func (s *PolicyEvaluationServer) SetHostnameAuthorizer(authz HostnameAuthorizer) {
	s.hostnameAuthz = authz
}

func (s *PolicyEvaluationServer) Evaluate(ctx context.Context, req *pb.PolicyEvaluationRequest) (*pb.PolicyEvaluationResponse, error) {
	verified, ok := VerifiedProxyFromContext(ctx)
	if !ok {
		s.logger.Error("evaluate: no verified proxy identity in context (session auth interceptor did not run)")
		return &pb.PolicyEvaluationResponse{Allowed: false, DenyReason: "Unauthenticated"}, nil
	}
	tenantID := verified.TenantID

	if req.TenantId != "" && req.TenantId != tenantID.String() {
		s.logger.Warn("evaluate: request tenant_id does not match session identity, rejecting as a possible spoofing attempt",
			"session_tenant_id", tenantID, "claimed_tenant_id", req.TenantId, "proxy_id", verified.ProxyID)
		return &pb.PolicyEvaluationResponse{Allowed: false, DenyReason: "IdentityMismatch"}, nil
	}

	evidence := mapEvidence(req.Client)
	target := &domain.ServerTarget{Host: req.Server.GetHost(), Port: int(req.Server.GetPort())}

	if req.ResourceSetId != "" {
		if id, err := uuid.Parse(req.ResourceSetId); err == nil {
			target.ResourceSetID = &id
		}
	}

	result, err := s.evaluator.Evaluate(ctx, tenantID, evidence, target)
	if err != nil {
		s.logger.Error("evaluation error", "tenant_id", tenantID, "error", err)
		return &pb.PolicyEvaluationResponse{Allowed: false, DenyReason: "InternalError"}, nil
	}

	return mapResponse(result), nil
}

func (s *PolicyEvaluationServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		s.logger.Warn("register: invalid tenant_id", "tenant_id", req.TenantId, "error", err)
		return &pb.RegisterResponse{Success: false, ErrorMessage: "invalid tenant_id"}, nil
	}

	proxy, sessionToken, err := s.registrar.RegisterProxy(ctx, tenantID, req.RegistrationToken)
	if err != nil {
		var regErr *domain.RegistrationError
		if errors.As(err, &regErr) {
			// RegistrationError messages are deliberately safe, specific
			// validation feedback for a legitimate proxy (e.g. "invalid or
			// already consumed registration token") — they never wrap
			// internal error detail, so returning them verbatim is fine.
			s.logger.Warn("register: registration failed", "tenant_id", tenantID, "error", err)
			return &pb.RegisterResponse{Success: false, ErrorMessage: regErr.Error()}, nil
		}
		// Any other error (persistence, session token generation/storage,
		// controller lookup, etc.) may wrap raw repository/database error
		// text. Register runs with no caller authentication at all — it is
		// the RPC that establishes it — so this detail must never cross
		// the wire, only be logged server-side.
		s.logger.Error("register: registration failed", "tenant_id", tenantID, "error", err)
		return &pb.RegisterResponse{Success: false, ErrorMessage: "registration failed"}, nil
	}

	s.logger.Info("proxy registered", "tenant_id", tenantID, "proxy_id", proxy.ID)

	resp := &pb.RegisterResponse{Success: true, ProxyId: proxy.ID.String(), SessionToken: sessionToken}

	certPEM, _, err := s.caProvider.GetOrCreateTenantCA(ctx, tenantID)
	if err != nil {
		s.logger.Error("register: tenant CA generation failed", "tenant_id", tenantID, "error", err)
		return resp, nil
	}

	resp.CaCertPem = certPEM
	return resp, nil
}

func (s *PolicyEvaluationServer) ReportHealth(ctx context.Context, req *pb.HealthReport) (*pb.HealthAck, error) {
	verified, ok := VerifiedProxyFromContext(ctx)
	if !ok {
		s.logger.Error("health: no verified proxy identity in context (session auth interceptor did not run)")
		return &pb.HealthAck{Acknowledged: false}, nil
	}
	tenantID, proxyID := verified.TenantID, verified.ProxyID

	if req.ComponentId != "" && req.ComponentId != proxyID.String() {
		s.logger.Warn("health: request component_id does not match session identity, rejecting as a possible spoofing attempt",
			"session_proxy_id", proxyID, "claimed_component_id", req.ComponentId)
		return &pb.HealthAck{Acknowledged: false}, nil
	}

	if err := s.healthUpdater.UpdateHealth(ctx, tenantID, proxyID, req.Status); err != nil {
		s.logger.Error("health: update failed", "tenant_id", tenantID, "proxy_id", proxyID, "error", err)
		return &pb.HealthAck{Acknowledged: false}, nil
	}

	s.logger.Debug("health report acknowledged", "tenant_id", tenantID, "proxy_id", proxyID, "status", req.Status)
	return &pb.HealthAck{Acknowledged: true}, nil
}

func (s *PolicyEvaluationServer) IngestEvents(stream grpc.ClientStreamingServer[pb.WorkloadEventBatch, pb.EventAck]) error {
	verified, ok := VerifiedProxyFromContext(stream.Context())
	if !ok {
		s.logger.Error("ingest: no verified proxy identity in context (session auth interceptor did not run)")
		return stream.SendAndClose(&pb.EventAck{Accepted: 0})
	}
	tenantID := verified.TenantID

	var accepted int32
	for {
		batch, err := stream.Recv()
		if err != nil {
			break
		}

		if batch.TenantId != "" && batch.TenantId != tenantID.String() {
			s.logger.Warn("ingest: batch tenant_id does not match session identity, rejecting this batch as a possible spoofing attempt",
				"session_tenant_id", tenantID, "claimed_tenant_id", batch.TenantId, "proxy_id", verified.ProxyID)
			continue
		}

		mapped := make([]events.WorkloadEvent, 0, len(batch.Events))
		for _, ev := range batch.Events {
			mapped = append(mapped, mapWorkloadEvent(ev))
		}

		if err := s.events.IngestFromProxy(stream.Context(), tenantID, mapped); err != nil {
			s.logger.Error("ingest: persist failed", "tenant_id", tenantID, "batch_size", len(mapped), "error", err)
			continue
		}
		accepted += int32(len(mapped))
	}
	return stream.SendAndClose(&pb.EventAck{Accepted: accepted})
}

func mapWorkloadEvent(ev *pb.WorkloadEvent) events.WorkloadEvent {
	w := events.WorkloadEvent{
		EventType: ev.EventType,
		Severity:  ev.Severity,
		Outcome:   ev.Outcome,
		Payload:   json.RawMessage(ev.Payload),
	}

	if ev.ConnectionId != "" {
		if id, err := uuid.Parse(ev.ConnectionId); err == nil {
			w.ConnectionID = id
		}
	}

	if ev.PolicyId != "" {
		if id, err := uuid.Parse(ev.PolicyId); err == nil {
			w.PolicyID = &id
		}
	}

	if ev.ClientWorkloadId != "" {
		if id, err := uuid.Parse(ev.ClientWorkloadId); err == nil {
			w.ClientWorkloadID = &id
		}
	}

	if ev.ServerWorkloadId != "" {
		if id, err := uuid.Parse(ev.ServerWorkloadId); err == nil {
			w.ServerWorkloadID = &id
		}
	}

	if ev.ResourceSetId != "" {
		if id, err := uuid.Parse(ev.ResourceSetId); err == nil {
			w.ResourceSetID = &id
		}
	}

	if ev.Timestamp > 0 {
		w.Timestamp = time.Unix(ev.Timestamp, 0)
	}

	return w
}

func mapEvidence(c *pb.ClientEvidence) *domain.ClientEvidence {
	if c == nil {
		return &domain.ClientEvidence{}
	}

	ev := &domain.ClientEvidence{
		SourceIP:       c.SourceIp,
		Hostname:       c.Hostname,
		DipoleClientID: c.DipoleClientId,
		RedirectURI:    c.RedirectUri,
	}

	if c.K8S != nil {
		ev.K8s = &domain.K8sEvidence{
			ServiceAccountToken: c.K8S.ServiceAccountToken,
			Namespace:           c.K8S.Namespace,
			PodName:             c.K8S.PodName,
			ServiceAccountName:  c.K8S.ServiceAccountName,
			ServiceAccountUID:   c.K8S.ServiceAccountUid,
		}
	}

	if c.Aws != nil {
		ev.AWS = &domain.AWSEvidence{
			InstanceIdentityDocument:  c.Aws.InstanceIdentityDocument,
			InstanceIdentitySignature: c.Aws.InstanceIdentitySignature,
		}
		if c.Aws.StsGetCallerIdentity != nil {
			ev.AWS.STSGetCallerIdentity = &domain.STSGetCallerIdentity{
				SignedHeaders: c.Aws.StsGetCallerIdentity.SignedHeaders,
				Region:        c.Aws.StsGetCallerIdentity.Region,
			}
		}
	}

	if c.Azure != nil {
		ev.Azure = &domain.AzureEvidence{
			Encoding:  c.Azure.Encoding,
			Signature: c.Azure.Signature,
			Nonce:     c.Azure.Nonce,
		}
	}

	if c.Gcp != nil {
		ev.GCP = &domain.GCPEvidence{IdentityToken: c.Gcp.IdentityToken}
	}

	if c.Github != nil {
		ev.GitHub = &domain.IdentityTokenEvidence{IdentityToken: c.Github.IdentityToken}
	}
	if c.Gitlab != nil {
		ev.GitLab = &domain.IdentityTokenEvidence{IdentityToken: c.Gitlab.IdentityToken}
	}
	if c.Terraform != nil {
		ev.Terraform = &domain.IdentityTokenEvidence{IdentityToken: c.Terraform.IdentityToken}
	}

	if c.Process != nil {
		ev.Process = &domain.ProcessEvidence{
			Name:        c.Process.Name,
			Path:        c.Process.Path,
			CommandLine: c.Process.CommandLine,
			UserName:    c.Process.UserName,
		}
	}

	ev.Environment = c.Environment
	return ev
}

func mapResponse(r *domain.EvaluationResult) *pb.PolicyEvaluationResponse {
	resp := &pb.PolicyEvaluationResponse{
		Allowed:    r.Allowed,
		DenyReason: r.DenyReason,
	}

	if r.Credential != nil {
		resp.Credential = &pb.Credential{
			Type:               r.Credential.Type,
			Value:              r.Credential.Value,
			Username:           r.Credential.Username,
			Password:           r.Credential.Password,
			AwsAccessKeyId:     r.Credential.AWSAccessKeyID,
			AwsSecretAccessKey: r.Credential.AWSSecretAccessKey,
			AwsSessionToken:    r.Credential.AWSSessionToken,
			ExpiresAt:          r.Credential.ExpiresAt,
		}
	}

	if r.InjectionConfig != nil {
		resp.Injection = &pb.InjectionConfig{
			Method:     r.InjectionConfig.Method,
			HeaderName: r.InjectionConfig.HeaderName,
			Prefix:     r.InjectionConfig.Prefix,
			QueryParam: r.InjectionConfig.QueryParam,
		}
	}

	if r.PolicyID != uuid.Nil {
		resp.PolicyId = r.PolicyID.String()
	}
	if r.ClientWorkloadID != uuid.Nil {
		resp.ClientWorkloadId = r.ClientWorkloadID.String()
	}
	if r.ServerWorkloadID != uuid.Nil {
		resp.ServerWorkloadId = r.ServerWorkloadID.String()
	}

	for _, tr := range r.TrustResults {
		resp.TrustResults = append(resp.TrustResults, &pb.TrustProviderResult{
			ProviderId:    tr.ProviderID.String(),
			ProviderName:  tr.ProviderName,
			Attested:      tr.Attested,
			Reason:        tr.Reason,
			Attribute:     tr.Attribute,
			ExpectedValue: tr.ExpectedValue,
			ActualValue:   tr.ActualValue,
		})
	}

	return resp
}

// SignLeafCertificate handles the proxy's request to sign a CSR for TLS interception.
// Security flow:
//  1. Authenticate the proxy — the session auth interceptor has already
//     verified the caller's identity from its bearer session token before
//     this handler runs; tenant_id/proxy_id come from that verified
//     identity, not from the request body.
//  2. Authorize the hostname (check active policy with TLS server workload)
//  3. Validate and sign the CSR
func (s *PolicyEvaluationServer) SignLeafCertificate(ctx context.Context, req *pb.SignLeafRequest) (*pb.SignLeafResponse, error) {
	// 1. Require a verified caller identity. This is the sole source of
	// truth for tenant_id/proxy_id — the corresponding request fields are
	// unauthenticated client input and are never used for authorization.
	verified, ok := VerifiedProxyFromContext(ctx)
	if !ok {
		s.logger.Error("sign-leaf: no verified proxy identity in context (session auth interceptor did not run)")
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "authentication required"}, nil
	}
	tenantID, proxyID := verified.TenantID, verified.ProxyID

	if req.ProxyId != "" && req.ProxyId != proxyID.String() {
		s.logger.Warn("sign-leaf: request proxy_id does not match session identity, rejecting as a possible spoofing attempt",
			"session_proxy_id", proxyID, "claimed_proxy_id", req.ProxyId)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "identity mismatch"}, nil
	}
	if req.TenantId != "" && req.TenantId != tenantID.String() {
		s.logger.Warn("sign-leaf: request tenant_id does not match session identity, rejecting as a possible spoofing attempt",
			"session_tenant_id", tenantID, "claimed_tenant_id", req.TenantId, "proxy_id", proxyID)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "identity mismatch"}, nil
	}

	if req.Hostname == "" {
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "hostname is required"}, nil
	}

	if len(req.CsrDer) == 0 {
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "CSR is required"}, nil
	}

	// 2. Check if signing infrastructure is configured
	if s.leafSigner == nil || s.hostnameAuthz == nil {
		s.logger.Error("sign-leaf: required dependencies not configured")
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "TLS Decrypt not configured"}, nil
	}

	// 3. Verify TLS Decrypt is enabled (CA must already exist).
	// We do NOT lazily generate the CA here — a signing request must not
	// trigger CA creation. The admin must explicitly enable TLS Decrypt first.
	if !s.leafSigner.IsTlsDecryptEnabled(ctx, tenantID) {
		s.logger.Warn("sign-leaf: TLS Decrypt not enabled for tenant", "tenant_id", tenantID)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "TLS Decrypt is not enabled for this tenant"}, nil
	}

	// 4. Authorize hostname — only sign for hostnames in active policies with TLS
	authorized, err := s.hostnameAuthz.IsHostnameAuthorizedForTLS(ctx, tenantID, req.Hostname)
	if err != nil {
		s.logger.Error("sign-leaf: authorization check failed", "tenant_id", tenantID, "hostname", req.Hostname, "error", err)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "authorization check failed"}, nil
	}
	if !authorized {
		s.logger.Warn("sign-leaf: hostname not authorized", "tenant_id", tenantID, "hostname", req.Hostname, "proxy_id", proxyID)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "hostname not authorized for TLS interception"}, nil
	}

	// 5. Sign the CSR
	resp, err := s.leafSigner.SignCSR(ctx, pki.SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   req.CsrDer,
		Hostname: req.Hostname,
	})
	if err != nil {
		s.logger.Error("sign-leaf: signing failed", "tenant_id", tenantID, "hostname", req.Hostname, "error", err)
		// The underlying error (CSR parsing, CA decrypt, x509 signing
		// failures, etc.) is logged above but never returned to the
		// caller.
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "certificate signing failed"}, nil
	}

	s.logger.Info("sign-leaf: certificate issued",
		"tenant_id", tenantID,
		"hostname", req.Hostname,
		"proxy_id", proxyID,
		"not_after", resp.NotAfter.Format(time.RFC3339),
	)

	return &pb.SignLeafResponse{
		CertDer:  resp.CertDER,
		NotAfter: resp.NotAfter.Unix(),
		Success:  true,
	}, nil
}
