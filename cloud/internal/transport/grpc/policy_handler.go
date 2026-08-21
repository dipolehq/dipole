// policy_handler.go implements the gRPC PolicyEvaluationService RPCs, mapping protobuf messages to domain types and delegating to the policy evaluator, registration, health, event, and PKI services.
package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/service/pki"
	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type PolicyEvaluator interface {
	Evaluate(ctx context.Context, tenantID uuid.UUID, evidence *domain.ClientEvidence, target *domain.ServerTarget) (*domain.EvaluationResult, error)
}

type ProxyRegistrar interface {
	RegisterProxy(ctx context.Context, tenantID uuid.UUID, registrationToken string) (proxy *domain.AgentProxy, sessionToken string, sessionTokenExpiresAt time.Time, err error)
	RefreshSessionToken(ctx context.Context, tenantID, proxyID uuid.UUID) (sessionToken string, sessionTokenExpiresAt time.Time, err error)
}

type ProxyHealthUpdater interface {
	UpdateHealth(ctx context.Context, tenantID uuid.UUID, proxyID uuid.UUID, status string) error
}

type TenantCAProvider interface {
	GetOrCreateTenantCA(ctx context.Context, tenantID uuid.UUID) (certPEM string, keyPEM string, err error)
}

type LeafCertSigner interface {
	SignCSR(ctx context.Context, req pki.SignCSRRequest) (*pki.SignCSRResponse, error)
	IsTlsDecryptEnabled(ctx context.Context, tenantID uuid.UUID) (bool, error)
}

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

func (s *PolicyEvaluationServer) SetLeafSigner(signer LeafCertSigner) {
	s.leafSigner = signer
}

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

	proxy, sessionToken, sessionTokenExpiresAt, err := s.registrar.RegisterProxy(ctx, tenantID, req.RegistrationToken)
	if err != nil {
		var regErr *domain.RegistrationError
		if errors.As(err, &regErr) {
			s.logger.Warn("register: registration failed", "tenant_id", tenantID, "error", err)
			return &pb.RegisterResponse{Success: false, ErrorMessage: regErr.Error()}, nil
		}
		s.logger.Error("register: registration failed", "tenant_id", tenantID, "error", err)
		return &pb.RegisterResponse{Success: false, ErrorMessage: "registration failed"}, nil
	}

	s.logger.Info("proxy registered", "tenant_id", tenantID, "proxy_id", proxy.ID)

	resp := &pb.RegisterResponse{
		Success:               true,
		ProxyId:               proxy.ID.String(),
		SessionToken:          sessionToken,
		SessionTokenExpiresAt: sessionTokenExpiresAt.Unix(),
	}

	certPEM, _, err := s.caProvider.GetOrCreateTenantCA(ctx, tenantID)
	if err != nil {
		s.logger.Error("register: tenant CA generation failed", "tenant_id", tenantID, "error", err)
		return resp, nil
	}

	resp.CaCertPem = certPEM
	return resp, nil
}

func (s *PolicyEvaluationServer) RefreshSessionToken(ctx context.Context, _ *pb.RefreshSessionTokenRequest) (*pb.RefreshSessionTokenResponse, error) {
	verified, ok := VerifiedProxyFromContext(ctx)
	if !ok {
		s.logger.Error("refresh-session-token: no verified proxy identity in context (session auth interceptor did not run)")
		return &pb.RefreshSessionTokenResponse{Success: false, ErrorMessage: "authentication required"}, nil
	}

	sessionToken, expiresAt, err := s.registrar.RefreshSessionToken(ctx, verified.TenantID, verified.ProxyID)
	if err != nil {
		s.logger.Error("refresh-session-token: refresh failed", "tenant_id", verified.TenantID, "proxy_id", verified.ProxyID, "error", err)
		return &pb.RefreshSessionTokenResponse{Success: false, ErrorMessage: "session token refresh failed"}, nil
	}

	s.logger.Info("session token refreshed", "tenant_id", verified.TenantID, "proxy_id", verified.ProxyID)

	return &pb.RefreshSessionTokenResponse{
		Success:               true,
		SessionToken:          sessionToken,
		SessionTokenExpiresAt: expiresAt.Unix(),
	}, nil
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
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.logger.Error("ingest: stream receive failed", "tenant_id", tenantID, "proxy_id", verified.ProxyID, "error", err)
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

	if ev.EventId != "" {
		if id, err := uuid.Parse(ev.EventId); err == nil {
			w.EventID = id
		}
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

func (s *PolicyEvaluationServer) SignLeafCertificate(ctx context.Context, req *pb.SignLeafRequest) (*pb.SignLeafResponse, error) {
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

	if s.leafSigner == nil || s.hostnameAuthz == nil {
		s.logger.Error("sign-leaf: required dependencies not configured")
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "TLS Decrypt not configured"}, nil
	}

	enabled, err := s.leafSigner.IsTlsDecryptEnabled(ctx, tenantID)
	if err != nil {
		s.logger.Error("sign-leaf: failed to check TLS Decrypt status", "tenant_id", tenantID, "error", err)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "failed to verify TLS Decrypt status"}, nil
	}
	if !enabled {
		s.logger.Warn("sign-leaf: TLS Decrypt not enabled for tenant", "tenant_id", tenantID)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "TLS Decrypt is not enabled for this tenant"}, nil
	}

	authorized, err := s.hostnameAuthz.IsHostnameAuthorizedForTLS(ctx, tenantID, req.Hostname)
	if err != nil {
		s.logger.Error("sign-leaf: authorization check failed", "tenant_id", tenantID, "hostname", req.Hostname, "error", err)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "authorization check failed"}, nil
	}
	if !authorized {
		s.logger.Warn("sign-leaf: hostname not authorized", "tenant_id", tenantID, "hostname", req.Hostname, "proxy_id", proxyID)
		return &pb.SignLeafResponse{Success: false, ErrorMessage: "hostname not authorized for TLS interception"}, nil
	}

	resp, err := s.leafSigner.SignCSR(ctx, pki.SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   req.CsrDer,
		Hostname: req.Hostname,
	})
	if err != nil {
		s.logger.Error("sign-leaf: signing failed", "tenant_id", tenantID, "hostname", req.Hostname, "error", err)
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
