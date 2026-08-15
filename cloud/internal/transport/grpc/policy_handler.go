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
type ProxyRegistrar interface {
	RegisterProxy(ctx context.Context, tenantID uuid.UUID, registrationToken string) (*domain.AgentProxy, error)
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

// ProxyAuthenticator verifies that a proxy belongs to a tenant.
// Used to prevent cross-tenant certificate signing.
type ProxyAuthenticator interface {
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AgentProxy, error)
}

type PolicyEvaluationServer struct {
	pb.UnimplementedPolicyEvaluationServiceServer
	evaluator     PolicyEvaluator
	registrar     ProxyRegistrar
	healthUpdater ProxyHealthUpdater
	caProvider    TenantCAProvider
	leafSigner    LeafCertSigner
	hostnameAuthz HostnameAuthorizer
	proxyAuth     ProxyAuthenticator
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

// SetProxyAuthenticator configures the proxy identity verifier.
func (s *PolicyEvaluationServer) SetProxyAuthenticator(auth ProxyAuthenticator) {
	s.proxyAuth = auth
}

func (s *PolicyEvaluationServer) Evaluate(ctx context.Context, req *pb.PolicyEvaluationRequest) (*pb.PolicyEvaluationResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &pb.PolicyEvaluationResponse{Allowed: false, DenyReason: "InvalidTenant"}, nil
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

	proxy, err := s.registrar.RegisterProxy(ctx, tenantID, req.RegistrationToken)
	if err != nil {
		var regErr *domain.RegistrationError
		if errors.As(err, &regErr) {
			s.logger.Warn("register: registration failed", "tenant_id", tenantID, "error", err)
		} else {
			s.logger.Error("register: registration failed", "tenant_id", tenantID, "error", err)
		}
		return &pb.RegisterResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	s.logger.Info("proxy registered", "tenant_id", tenantID, "proxy_id", proxy.ID)

	resp := &pb.RegisterResponse{Success: true, ProxyId: proxy.ID.String()}

	certPEM, keyPEM, err := s.caProvider.GetOrCreateTenantCA(ctx, tenantID)
	if err != nil {
		s.logger.Error("register: tenant CA generation failed", "tenant_id", tenantID, "error", err)
		return resp, nil
	}

	resp.CaCertPem = certPEM
	resp.CaKeyPem = keyPEM
	return resp, nil
}

func (s *PolicyEvaluationServer) ReportHealth(ctx context.Context, req *pb.HealthReport) (*pb.HealthAck, error) {
	proxyID, err := uuid.Parse(req.ComponentId)
	if err != nil {
		s.logger.Warn("health: invalid component_id", "component_id", req.ComponentId, "error", err)
		return &pb.HealthAck{Acknowledged: false}, nil
	}

	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		s.logger.Warn("health: invalid tenant_id", "tenant_id", req.TenantId, "error", err)
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
	var accepted int32
	for {
		batch, err := stream.Recv()
		if err != nil {
			break
		}

		tenantID, err := uuid.Parse(batch.TenantId)
		if err != nil {
			s.logger.Warn("ingest: invalid tenant_id", "tenant_id", batch.TenantId)
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
//  1. Authenticate the proxy (tenant_id from the authenticated context)
//  2. Authorize the hostname (check active policy with TLS server workload)
//  3. Validate and sign the CSR
func (s *PolicyEvaluationServer) SignLeafCertificate(ctx context.Context, req *pb.SignLeafRequest) (*pb.SignLeafResponse, error) {
	// 1. Parse and validate tenant identity
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		s.logger.Warn("sign-leaf: invalid tenant_id", "tenant_id", req.TenantId)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "invalid tenant_id"}, nil
	}

	if req.Hostname == "" {
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "hostname is required"}, nil
	}

	if len(req.CsrDer) == 0 {
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "CSR is required"}, nil
	}

	if req.ProxyId == "" {
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "proxy_id is required"}, nil
	}

	// 2. Check if signing infrastructure is configured
	if s.leafSigner == nil || s.hostnameAuthz == nil || s.proxyAuth == nil {
		s.logger.Error("sign-leaf: required dependencies not configured")
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "TLS Decrypt not configured"}, nil
	}

	// 3. Authenticate the proxy — verify it belongs to the claimed tenant.
	// This prevents cross-tenant certificate issuance: a proxy from Tenant A
	// cannot request certs signed by Tenant B's CA.
	proxyID, err := uuid.Parse(req.ProxyId)
	if err != nil {
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "invalid proxy_id"}, nil
	}

	proxy, err := s.proxyAuth.GetByID(ctx, tenantID, proxyID)
	if err != nil {
		s.logger.Error("sign-leaf: proxy lookup failed", "tenant_id", tenantID, "proxy_id", proxyID, "error", err)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "proxy authentication failed"}, nil
	}
	if proxy == nil {
		s.logger.Warn("sign-leaf: proxy not found for tenant", "tenant_id", tenantID, "proxy_id", proxyID)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "proxy not registered for this tenant"}, nil
	}

	// 4. Verify TLS Decrypt is enabled (CA must already exist).
	// We do NOT lazily generate the CA here — a signing request must not
	// trigger CA creation. The admin must explicitly enable TLS Decrypt first.
	if !s.leafSigner.IsTlsDecryptEnabled(ctx, tenantID) {
		s.logger.Warn("sign-leaf: TLS Decrypt not enabled for tenant", "tenant_id", tenantID)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "TLS Decrypt is not enabled for this tenant"}, nil
	}

	// 5. Authorize hostname — only sign for hostnames in active policies with TLS
	authorized, err := s.hostnameAuthz.IsHostnameAuthorizedForTLS(ctx, tenantID, req.Hostname)
	if err != nil {
		s.logger.Error("sign-leaf: authorization check failed", "tenant_id", tenantID, "hostname", req.Hostname, "error", err)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "authorization check failed"}, nil
	}
	if !authorized {
		s.logger.Warn("sign-leaf: hostname not authorized", "tenant_id", tenantID, "hostname", req.Hostname, "proxy_id", req.ProxyId)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "hostname not authorized for TLS interception"}, nil
	}

	// 6. Sign the CSR
	resp, err := s.leafSigner.SignCSR(ctx, pki.SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   req.CsrDer,
		Hostname: req.Hostname,
	})
	if err != nil {
		s.logger.Error("sign-leaf: signing failed", "tenant_id", tenantID, "hostname", req.Hostname, "error", err)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	s.logger.Info("sign-leaf: certificate issued",
		"tenant_id", tenantID,
		"hostname", req.Hostname,
		"proxy_id", req.ProxyId,
		"not_after", resp.NotAfter.Format(time.RFC3339),
	)

	return &pb.SignLeafResponse{
		CertDer:  resp.CertDER,
		NotAfter: resp.NotAfter.Unix(),
		Success:  true,
	}, nil
}
