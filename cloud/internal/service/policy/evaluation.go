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
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type TrustValidator interface {
	Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence) (*domain.TrustProviderResult, error)
}

type ConditionEvaluator interface {
	EvaluateAll(ctx context.Context, conditions []domain.AccessCondition, evidence *domain.ClientEvidence) (bool, string, error)
}

type CredentialBroker interface {
	// tenantID is required so that credential types which mint a token we sign
	// use the requesting tenant's identity host as the `iss` claim. Cloud
	// federation providers (AWS STS, Azure Entra, GCP WIF) fetch OIDC discovery
	// from `iss` and reject a mismatch.
	Retrieve(ctx context.Context, tenantID uuid.UUID, provider *domain.CredentialProvider, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error)
}

type PolicyEvaluationService struct {
	repo       repository.AccessPolicyRepository
	trust      TrustValidator
	conditions ConditionEvaluator
	broker     CredentialBroker
	events     *events.EventBus
	logger     *slog.Logger
}

func NewPolicyEvaluationService(repo repository.AccessPolicyRepository, trust TrustValidator, conditions ConditionEvaluator, broker CredentialBroker, eventBus *events.EventBus, logger *slog.Logger) *PolicyEvaluationService {
	return &PolicyEvaluationService{
		repo:       repo,
		trust:      trust,
		conditions: conditions,
		broker:     broker,
		events:     eventBus,
		logger:     logger,
	}
}

func (s *PolicyEvaluationService) Evaluate(ctx context.Context, tenantID uuid.UUID, evidence *domain.ClientEvidence, target *domain.ServerTarget) (*domain.EvaluationResult, error) {
	policies, err := s.repo.FindByServerTarget(ctx, tenantID, target.Host, target.Port, target.ResourceSetID)
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

		if len(p.TrustProviders) == 0 {
			ec.result = &domain.EvaluationResult{Allowed: false, DenyReason: "NoTrustProvider"}
			return ec.result, nil
		}

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
			cred, err := s.broker.Retrieve(ctx, tenantID, cp, evidence, nil)
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
	if s.events == nil || ec.result == nil {
		return
	}

	contextID := uuid.New()

	severity := "Info"
	if !ec.result.Allowed {
		severity = "Warn"
		if ec.result.DenyReason == "InternalError" {
			severity = "Error"
		}
	}

	clientWL := ""
	var clientWLID *uuid.UUID
	if ec.clientWorkload != nil {
		clientWL = ec.clientWorkload.Name
		clientWLID = &ec.clientWorkload.ID
	}

	serverWL := ""
	var serverWLID *uuid.UUID
	if ec.serverWorkload != nil {
		serverWL = ec.serverWorkload.Name
		serverWLID = &ec.serverWorkload.ID
	}

	policyName := ""
	var policyID *uuid.UUID
	if ec.policy != nil {
		policyName = ec.policy.Name
		policyID = &ec.policy.ID
	}

	var trustResults []events.TrustResult
	if ec.result != nil {
		for _, tr := range ec.result.TrustResults {
			trustResults = append(trustResults, events.TrustResult{
				ProviderID:    tr.ProviderID,
				ProviderName:  tr.ProviderName,
				Attested:      tr.Attested,
				Reason:        tr.Reason,
				Attribute:     tr.Attribute,
				ExpectedValue: tr.ExpectedValue,
				ActualValue:   tr.ActualValue,
			})
		}
	}

	credType := ""
	credName := ""
	var credProviderID *uuid.UUID
	credMaxAge := 0
	if ec.credentialProvider != nil {
		credType = ec.credentialProvider.Type
		credName = ec.credentialProvider.Name
		credProviderID = &ec.credentialProvider.ID
		if ec.result.Allowed {
			credMaxAge = 60
		}
	}

	candidateCW := make([]events.WorkloadRef, 0, len(ec.candidateClientWorkloads))
	for _, cw := range ec.candidateClientWorkloads {
		candidateCW = append(candidateCW, events.WorkloadRef{ID: cw.ID, Name: cw.Name})
	}

	candidateSW := make([]events.WorkloadRef, 0, len(ec.candidateServerWorkloads))
	for _, sw := range ec.candidateServerWorkloads {
		candidateSW = append(candidateSW, events.WorkloadRef{ID: sw.ID, Name: sw.Name})
	}

	base := events.SecurityEvent{
		TenantID:                 ec.tenantID,
		ContextID:                contextID,
		SourceIP:                 ec.sourceIP,
		Source:                   "edge_proxy",
		Severity:                 "Info",
		Allowed:                  ec.result.Allowed,
		DenyReason:               ec.result.DenyReason,
		ClientWorkload:           clientWL,
		ClientWorkloadID:         clientWLID,
		ServerWorkload:           serverWL,
		ServerWorkloadID:         serverWLID,
		ServerHost:               ec.targetHost,
		ServerPort:               ec.targetPort,
		PolicyID:                 policyID,
		PolicyName:               policyName,
		TrustResults:             trustResults,
		CredentialType:           credType,
		CredentialName:           credName,
		CredentialProviderID:     credProviderID,
		CredentialMaxAge:         credMaxAge,
		CandidateClientWorkloads: candidateCW,
		CandidateServerWorkloads: candidateSW,
	}

	request := base
	request.EventType = "access.request"
	if !ec.result.Allowed {
		request.Severity = "Error"
	}
	s.events.EmitSecurity(ctx, request)

	auth := base
	auth.EventType = "access.authorization"
	auth.Severity = severity
	s.events.EmitSecurity(ctx, auth)

	if ec.credentialProvider != nil {
		cred := base
		cred.EventType = "access.credential"
		if !ec.result.Allowed {
			cred.Severity = "Error"
		}
		s.events.EmitSecurity(ctx, cred)
	}

	discovery := base
	discovery.EventType = "access.discovery"
	if !ec.result.Allowed && ec.clientWorkload == nil {
		discovery.Severity = "Error"
	}
	if err := s.events.EmitSecurity(ctx, discovery); err != nil {
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
	case "aws_account_id":
		return ev.AWS != nil && extractAWSField(ev.AWS.InstanceIdentityDocument, "accountId") == id.Value
	case "aws_ec2_instance_id":
		return ev.AWS != nil && extractAWSField(ev.AWS.InstanceIdentityDocument, "instanceId") == id.Value
	case "aws_ecs_task_family":
		return ev.AWS != nil && extractAWSField(ev.AWS.InstanceIdentityDocument, "ecsTaskFamily") == id.Value
	case "aws_ecs_service_name":
		return ev.AWS != nil && extractAWSField(ev.AWS.InstanceIdentityDocument, "ecsServiceName") == id.Value
	case "aws_lambda_arn":
		return ev.AWS != nil && extractAWSField(ev.AWS.InstanceIdentityDocument, "lambdaArn") == id.Value
	case "aws_region":
		return ev.AWS != nil && extractAWSField(ev.AWS.InstanceIdentityDocument, "region") == id.Value
	case "azure_subscription_id":
		return ev.Azure != nil && extractAWSField(ev.Azure.Encoding, "subscriptionId") == id.Value
	case "azure_vm_id":
		return ev.Azure != nil && extractAWSField(ev.Azure.Encoding, "vmId") == id.Value
	case "gcp_identity_token":
		return ev.GCP != nil && extractJWTClaim(ev.GCP.IdentityToken, "email") == id.Value
	case "github_id_token_repository":
		return ev.GitHub != nil && extractJWTClaim(ev.GitHub.IdentityToken, "repository") == id.Value
	case "github_id_token_subject":
		return ev.GitHub != nil && extractJWTClaim(ev.GitHub.IdentityToken, "sub") == id.Value
	case "gitlab_id_token_namespace_path":
		return ev.GitLab != nil && extractJWTClaim(ev.GitLab.IdentityToken, "namespace_path") == id.Value
	case "gitlab_id_token_project_path":
		return ev.GitLab != nil && extractJWTClaim(ev.GitLab.IdentityToken, "project_path") == id.Value
	case "gitlab_id_token_ref_path":
		return ev.GitLab != nil && extractJWTClaim(ev.GitLab.IdentityToken, "ref_path") == id.Value
	case "gitlab_id_token_subject":
		return ev.GitLab != nil && extractJWTClaim(ev.GitLab.IdentityToken, "sub") == id.Value
	case "terraform_cloud_organization_id":
		return ev.Terraform != nil && extractJWTClaim(ev.Terraform.IdentityToken, "terraform_organization_id") == id.Value
	case "terraform_cloud_project_id":
		return ev.Terraform != nil && extractJWTClaim(ev.Terraform.IdentityToken, "terraform_project_id") == id.Value
	case "terraform_cloud_workspace_id":
		return ev.Terraform != nil && extractJWTClaim(ev.Terraform.IdentityToken, "terraform_workspace_id") == id.Value
	case "oidc_id_token":
		return ev.Environment != nil && ev.Environment["OIDC_ID_TOKEN"] != ""
	case "oidc_id_token_audience":
		return ev.Environment != nil && extractJWTClaim(ev.Environment["OIDC_ID_TOKEN"], "aud") == id.Value
	case "oidc_id_token_issuer":
		return ev.Environment != nil && extractJWTClaim(ev.Environment["OIDC_ID_TOKEN"], "iss") == id.Value
	case "oidc_id_token_subject":
		return ev.Environment != nil && extractJWTClaim(ev.Environment["OIDC_ID_TOKEN"], "sub") == id.Value
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

// extractAWSField decodes a base64-encoded JSON identity document and returns
// the string value of the specified field. Returns empty string on any failure.
func extractAWSField(encodedDoc string, field string) string {
	if encodedDoc == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(encodedDoc)
	if err != nil {
		raw = []byte(encodedDoc)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	v, ok := doc[field]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// extractJWTClaim decodes a JWT (without signature verification — that happens
// in the trust validation step) and returns the string value of the specified
// claim from the payload. Returns empty string on any failure.
func extractJWTClaim(token string, claim string) string {
	if token == "" {
		return ""
	}
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return ""
	}
	payload := parts[1]
	if pad := len(payload) % 4; pad != 0 {
		payload += strings.Repeat("=", 4-pad)
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return ""
	}
	v, ok := claims[claim]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
