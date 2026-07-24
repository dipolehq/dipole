/*
Package policy implements the core policy evaluation engine (PDP).

PolicyEvaluationService evaluates access requests: finds matching policies by
server target, verifies client identity against identifiers (AND logic), runs
trust verification and condition checks, retrieves credentials, and builds
injection config. Fail-closed: any error results in DENY.
*/
package policy

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service/events"
)

type TrustValidator interface {
	Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence) (*domain.TrustProviderResult, error)
}

type ConditionEvaluator interface {
	EvaluateAll(ctx context.Context, conditions []domain.AccessCondition, evidence *domain.ClientEvidence) (bool, string, error)
}

type CredentialBroker interface {
	Retrieve(ctx context.Context, provider *domain.CredentialProvider, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error)
}

type EventEmitter interface {
	Emit(ctx context.Context, eval *events.EvaluationContext) error
}

type PolicyEvaluationService struct {
	repo       repository.AccessPolicyRepository
	trust      TrustValidator
	conditions ConditionEvaluator
	broker     CredentialBroker
	emitter    EventEmitter
	logger     *slog.Logger
}

func NewPolicyEvaluationService(repo repository.AccessPolicyRepository, trust TrustValidator, conditions ConditionEvaluator, broker CredentialBroker, emitter EventEmitter, logger *slog.Logger) *PolicyEvaluationService {
	return &PolicyEvaluationService{
		repo:       repo,
		trust:      trust,
		conditions: conditions,
		broker:     broker,
		emitter:    emitter,
		logger:     logger,
	}
}

func (s *PolicyEvaluationService) Evaluate(ctx context.Context, tenantID uuid.UUID, evidence *domain.ClientEvidence, target *domain.ServerTarget) (*domain.EvaluationResult, error) {
	policies, err := s.repo.FindByServerTarget(ctx, tenantID, target.Host, target.Port)
	if err != nil {
		return nil, err
	}

	var ec eventContext
	ec.tenantID = tenantID
	ec.sourceIP = evidence.SourceIP
	ec.targetHost = target.Host
	ec.targetPort = target.Port

	defer func() {
		s.emitEvents(ctx, &ec)
	}()

	if len(policies) == 0 {
		ec.result = &domain.EvaluationResult{Allowed: false, DenyReason: "NoMatchingPolicy"}
		return ec.result, nil
	}

	for i := range policies {
		ec.candidateClientWorkloads = append(ec.candidateClientWorkloads, policies[i].ClientWorkload)
		ec.candidateServerWorkloads = append(ec.candidateServerWorkloads, policies[i].ServerWorkload)
	}

	var lastTrustResults []domain.TrustProviderResult

	for _, p := range policies {
		if !matchAllIdentifiers(p.ClientWorkload.Identifiers, evidence) {
			continue
		}

		ec.clientWorkload = &p.ClientWorkload
		ec.serverWorkload = &p.ServerWorkload
		ec.policy = &p.Policy
		ec.trustProviders = p.TrustProviders

		// Trust verification (AND logic — all must attest)
		trustFailed := false
		var trustResults []domain.TrustProviderResult
		for _, tp := range p.TrustProviders {
			tp := tp
			result, err := s.trust.Validate(ctx, &tp, evidence)
			if err != nil {
				return nil, err
			}
			trustResults = append(trustResults, *result)
			if !result.Attested {
				trustFailed = true
				break
			}
		}
		if trustFailed {
			lastTrustResults = trustResults
			ec.clientWorkload = nil
			ec.serverWorkload = nil
			ec.policy = nil
			ec.trustProviders = nil
			continue
		}

		// Condition evaluation (AND logic — all must pass)
		if len(p.AccessConditions) > 0 {
			passed, reason, err := s.conditions.EvaluateAll(ctx, p.AccessConditions, evidence)
			if err != nil {
				return nil, err
			}
			if !passed {
				ec.result = &domain.EvaluationResult{Allowed: false, DenyReason: "ConditionFailed:" + reason}
				return ec.result, nil
			}
		}

		// Retrieve credential
		var credential *domain.Credential
		if len(p.CredentialProviders) > 0 {
			cp := &p.CredentialProviders[0]
			ec.credentialProvider = cp
			cred, err := s.broker.Retrieve(ctx, cp, evidence, nil)
			if err != nil {
				ec.result = &domain.EvaluationResult{Allowed: false, DenyReason: "CredentialRetrievalFailed"}
				return ec.result, nil
			}
			credential = cred
		}

		ec.result = &domain.EvaluationResult{
			Allowed:          true,
			Credential:       credential,
			InjectionConfig:  buildInjectionConfig(&p.ServerWorkload),
			PolicyID:         p.Policy.ID,
			ClientWorkloadID: p.ClientWorkload.ID,
			ServerWorkloadID: p.ServerWorkload.ID,
		}
		return ec.result, nil
	}

	if len(lastTrustResults) > 0 {
		ec.result = &domain.EvaluationResult{Allowed: false, DenyReason: "AttestationFailed", TrustResults: lastTrustResults}
		return ec.result, nil
	}

	ec.result = &domain.EvaluationResult{Allowed: false, DenyReason: "NoMatchingPolicy"}
	return ec.result, nil
}

type eventContext struct {
	tenantID                 uuid.UUID
	sourceIP                 string
	targetHost               string
	targetPort               int
	result                   *domain.EvaluationResult
	clientWorkload           *domain.ClientWorkload
	serverWorkload           *domain.ServerWorkload
	policy                   *domain.AccessPolicy
	trustProviders           []domain.TrustProvider
	credentialProvider       *domain.CredentialProvider
	candidateClientWorkloads []domain.ClientWorkload
	candidateServerWorkloads []domain.ServerWorkload
}

func (s *PolicyEvaluationService) emitEvents(ctx context.Context, ec *eventContext) {
	if s.emitter == nil || ec.result == nil {
		return
	}

	evalCtx := &events.EvaluationContext{
		TenantID:                 ec.tenantID,
		SourceIP:                 ec.sourceIP,
		TargetHost:               ec.targetHost,
		TargetPort:               ec.targetPort,
		Result:                   ec.result,
		ClientWorkload:           ec.clientWorkload,
		ServerWorkload:           ec.serverWorkload,
		CandidateClientWorkloads: ec.candidateClientWorkloads,
		CandidateServerWorkloads: ec.candidateServerWorkloads,
	}

	if ec.policy != nil {
		evalCtx.Policy = ec.policy
	}
	if ec.credentialProvider != nil {
		evalCtx.CredentialProvider = ec.credentialProvider
	}
	if ec.trustProviders != nil {
		evalCtx.TrustProviders = ec.trustProviders
	}

	if err := s.emitter.Emit(ctx, evalCtx); err != nil {
		s.logger.Error("failed to emit auth events", "tenant_id", ec.tenantID, "error", err)
	}
}

// matchAllIdentifiers implements Aembit's identifier semantics: values of the
// same type are OR'd together, and distinct types are AND'd. A workload
// matches only if every type group has at least one matching value.
func matchAllIdentifiers(identifiers []domain.Identifier, evidence *domain.ClientEvidence) bool {
	if len(identifiers) == 0 {
		return true
	}

	byType := make(map[string][]domain.Identifier)
	order := make([]string, 0)
	for _, id := range identifiers {
		if _, seen := byType[id.Type]; !seen {
			order = append(order, id.Type)
		}
		byType[id.Type] = append(byType[id.Type], id)
	}

	for _, t := range order {
		matched := false
		for _, id := range byType[t] {
			if matchIdentifier(id, evidence) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func matchIdentifier(id domain.Identifier, ev *domain.ClientEvidence) bool {
	switch id.Type {
	case "k8s_namespace":
		return id.Value == k8sNamespace(ev)
	case "k8s_pod_name":
		return id.Value == k8sPodName(ev)
	case "k8s_pod_name_prefix":
		return strings.HasPrefix(k8sPodName(ev), id.Value)
	case "k8s_service_account_name":
		return id.Value == k8sServiceAccountName(ev)
	case "k8s_service_account_uid":
		return id.Value == k8sServiceAccountUID(ev)
	case "hostname":
		return id.Value == ev.Hostname
	case "source_ip":
		return id.Value == ev.SourceIP
	case "dipole_client_id":
		return id.Value == ev.DipoleClientID
	case "process_name":
		return ev.Process != nil && id.Value == ev.Process.Name
	case "process_path":
		return ev.Process != nil && id.Value == ev.Process.Path
	case "process_command_line":
		return ev.Process != nil && strings.Contains(ev.Process.CommandLine, id.Value)
	case "process_user_name":
		return ev.Process != nil && id.Value == ev.Process.UserName
	case "aws_account_id", "aws_ec2_instance_id", "aws_ecs_task_family",
		"aws_ecs_service_name", "aws_lambda_arn", "aws_region":
		return ev.AWS != nil
	case "azure_subscription_id", "azure_vm_id":
		return ev.Azure != nil
	case "gcp_identity_token":
		return ev.GCP != nil
	case "github_id_token_repository", "github_id_token_subject":
		return ev.GitHub != nil
	case "gitlab_id_token_namespace_path", "gitlab_id_token_project_path",
		"gitlab_id_token_ref_path", "gitlab_id_token_subject":
		return ev.GitLab != nil
	case "terraform_cloud_organization_id", "terraform_cloud_project_id",
		"terraform_cloud_workspace_id":
		return ev.Terraform != nil
	case "oidc_id_token", "oidc_id_token_audience", "oidc_id_token_issuer",
		"oidc_id_token_subject":
		return ev.Environment != nil && ev.Environment["OIDC_ID_TOKEN"] != ""
	case "redirect_uri":
		return id.Value == ev.RedirectURI
	default:
		return false
	}
}

func k8sNamespace(ev *domain.ClientEvidence) string {
	if ev.K8s != nil && ev.K8s.Namespace != "" {
		return ev.K8s.Namespace
	}
	if ev.Environment != nil {
		return ev.Environment["K8S_NAMESPACE"]
	}
	return ""
}

func k8sPodName(ev *domain.ClientEvidence) string {
	if ev.K8s != nil && ev.K8s.PodName != "" {
		return ev.K8s.PodName
	}
	if ev.Environment != nil {
		return ev.Environment["K8S_POD_NAME"]
	}
	return ""
}

func k8sServiceAccountName(ev *domain.ClientEvidence) string {
	if ev.K8s != nil && ev.K8s.ServiceAccountName != "" {
		return ev.K8s.ServiceAccountName
	}
	return ""
}

func k8sServiceAccountUID(ev *domain.ClientEvidence) string {
	if ev.K8s != nil && ev.K8s.ServiceAccountUID != "" {
		return ev.K8s.ServiceAccountUID
	}
	return ""
}

func buildInjectionConfig(sw *domain.ServerWorkload) *domain.InjectionConfig {
	switch sw.AuthMethod {
	case "http_bearer":
		return &domain.InjectionConfig{Method: "http_bearer", HeaderName: "Authorization", Prefix: "Bearer "}
	case "http_basic":
		return &domain.InjectionConfig{Method: "http_basic", HeaderName: "Authorization", Prefix: "Basic "}
	case "http_header":
		return &domain.InjectionConfig{Method: "http_header", HeaderName: sw.AuthConfig["header_name"]}
	case "api_key_header":
		return &domain.InjectionConfig{Method: "api_key_header", HeaderName: sw.AuthConfig["header_name"]}
	case "api_key_query":
		return &domain.InjectionConfig{Method: "api_key_query", QueryParam: sw.AuthConfig["query_param"]}
	case "password_auth":
		return &domain.InjectionConfig{Method: "password_auth"}
	case "sigv4":
		return &domain.InjectionConfig{Method: "sigv4"}
	case "snowflake_jwt":
		return &domain.InjectionConfig{Method: "snowflake_jwt"}
	case "mtls_x509":
		return &domain.InjectionConfig{Method: "mtls_x509"}
	default:
		return &domain.InjectionConfig{Method: "http_bearer", HeaderName: "Authorization", Prefix: "Bearer "}
	}
}
