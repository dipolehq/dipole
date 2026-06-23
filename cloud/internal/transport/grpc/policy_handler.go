/*
policy_handler.go implements the gRPC PolicyEvaluationService RPCs.

PolicyEvaluationServer is a thin transport layer: it maps protobuf messages
to domain types, delegates to the policy evaluator, and maps results back.
No business logic lives here. Register, ReportHealth, and IngestEvents are
stubs for now (implemented in later jobs).
*/
package grpc

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type PolicyEvaluator interface {
	Evaluate(ctx context.Context, tenantID uuid.UUID, evidence *domain.ClientEvidence, target *domain.ServerTarget) (*domain.EvaluationResult, error)
}

type PolicyEvaluationServer struct {
	pb.UnimplementedPolicyEvaluationServiceServer
	evaluator PolicyEvaluator
	logger    *slog.Logger
}

func NewPolicyEvaluationServer(evaluator PolicyEvaluator, logger *slog.Logger) *PolicyEvaluationServer {
	return &PolicyEvaluationServer{evaluator: evaluator, logger: logger}
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

func (s *PolicyEvaluationServer) Register(_ context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.logger.Info("register called", "tenant_id", req.TenantId)
	return &pb.RegisterResponse{Success: true, ProxyId: uuid.New().String()}, nil
}

func (s *PolicyEvaluationServer) ReportHealth(_ context.Context, req *pb.HealthReport) (*pb.HealthAck, error) {
	s.logger.Debug("health report", "component_id", req.ComponentId, "status", req.Status)
	return &pb.HealthAck{Acknowledged: true}, nil
}

func (s *PolicyEvaluationServer) IngestEvents(stream grpc.ClientStreamingServer[pb.WorkloadEventBatch, pb.EventAck]) error {
	count := 0
	for {
		_, err := stream.Recv()
		if err != nil {
			break
		}
		count++
	}
	return stream.SendAndClose(&pb.EventAck{Accepted: int32(count)})
}

func mapEvidence(c *pb.ClientEvidence) *domain.ClientEvidence {
	if c == nil {
		return &domain.ClientEvidence{}
	}

	ev := &domain.ClientEvidence{
		SourceIP:       c.SourceIp,
		Hostname:       c.Hostname,
		AembitClientID: c.AembitClientId,
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
			Type:     r.Credential.Type,
			Value:    r.Credential.Value,
			Username: r.Credential.Username,
			Password: r.Credential.Password,
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

	return resp
}
