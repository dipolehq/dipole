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
	"github.com/dipolehq/dipole/cloud/internal/service/events"
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

// WorkloadEventPersister abstracts workload event ingestion and persistence.
type WorkloadEventPersister interface {
	Ingest(ctx context.Context, event *events.IncomingWorkloadEvent) error
}

type PolicyEvaluationServer struct {
	pb.UnimplementedPolicyEvaluationServiceServer
	evaluator     PolicyEvaluator
	registrar     ProxyRegistrar
	healthUpdater ProxyHealthUpdater
	caProvider    TenantCAProvider
	eventIngester WorkloadEventPersister
	logger        *slog.Logger
}

func NewPolicyEvaluationServer(evaluator PolicyEvaluator, registrar ProxyRegistrar, healthUpdater ProxyHealthUpdater, caProvider TenantCAProvider, eventIngester WorkloadEventPersister, logger *slog.Logger) *PolicyEvaluationServer {
	return &PolicyEvaluationServer{
		evaluator:     evaluator,
		registrar:     registrar,
		healthUpdater: healthUpdater,
		caProvider:    caProvider,
		eventIngester: eventIngester,
		logger:        logger,
	}
}

func (s *PolicyEvaluationServer) Evaluate(ctx context.Context, req *pb.PolicyEvaluationRequest) (*pb.PolicyEvaluationResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &pb.PolicyEvaluationResponse{Allowed: false, DenyReason: "InvalidTenant"}, nil
	}

	evidence := mapEvidence(req.Client)
	target := &domain.ServerTarget{Host: req.Server.GetHost(), Port: int(req.Server.GetPort())}

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

		for _, ev := range batch.Events {
			incoming := mapWorkloadEvent(tenantID, ev)
			if err := s.eventIngester.Ingest(stream.Context(), incoming); err != nil {
				s.logger.Error("ingest: persist failed", "tenant_id", tenantID, "event_id", ev.EventId, "error", err)
				continue
			}
			accepted++
		}
	}
	return stream.SendAndClose(&pb.EventAck{Accepted: accepted})
}

func mapWorkloadEvent(tenantID uuid.UUID, ev *pb.WorkloadEvent) *events.IncomingWorkloadEvent {
	incoming := &events.IncomingWorkloadEvent{
		TenantID:  tenantID,
		EventType: ev.EventType,
		Severity:  ev.Severity,
		Outcome:   ev.Outcome,
		Payload:   json.RawMessage(ev.Payload),
	}

	if ev.ConnectionId != "" {
		if id, err := uuid.Parse(ev.ConnectionId); err == nil {
			incoming.ConnectionID = id
		}
	}

	if ev.PolicyId != "" {
		if id, err := uuid.Parse(ev.PolicyId); err == nil {
			incoming.PolicyID = &id
		}
	}

	if ev.ClientWorkloadId != "" {
		if id, err := uuid.Parse(ev.ClientWorkloadId); err == nil {
			incoming.ClientWorkloadID = &id
		}
	}

	if ev.ServerWorkloadId != "" {
		if id, err := uuid.Parse(ev.ServerWorkloadId); err == nil {
			incoming.ServerWorkloadID = &id
		}
	}

	if ev.ResourceSetId != "" {
		if id, err := uuid.Parse(ev.ResourceSetId); err == nil {
			incoming.ResourceSetID = &id
		}
	}

	if ev.Timestamp > 0 {
		incoming.Timestamp = time.Unix(ev.Timestamp, 0)
	} else {
		incoming.Timestamp = time.Now()
	}

	return incoming
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
