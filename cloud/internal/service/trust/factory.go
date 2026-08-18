/*
factory.go creates a fully-configured TrustValidationService with all
type-specific validators registered. Called during Cloud service initialization.
*/
package trust

import (
	"log/slog"

	"github.com/dipolehq/dipole/cloud/internal/repository"
)

func NewConfiguredService(logger *slog.Logger, controllerRepo repository.AgentControllerRepository, encryptionKey []byte) *TrustValidationService {
	cache := NewJWKSCache(logger)
	svc := NewTrustValidationService(cache, logger)

	svc.RegisterValidator("kubernetes_service_account", &K8sValidator{})
	svc.RegisterValidator("github_action", &GitHubValidator{})
	svc.RegisterValidator("gitlab_job", &OIDCValidator{})
	svc.RegisterValidator("terraform_cloud", &OIDCValidator{})
	svc.RegisterValidator("oidc_id_token", &OIDCValidator{})
	svc.RegisterValidator("aws_metadata_service", &AWSIMDSValidator{})
	svc.RegisterValidator("aws_role", NewAWSRoleValidator())
	svc.RegisterValidator("azure_metadata", &AzureValidator{})
	svc.RegisterValidator("gcp_identity_token", &GCPValidator{})
	svc.RegisterValidator("kerberos", NewKerberosValidator(controllerRepo, encryptionKey))
	svc.RegisterValidator("certificate_signed_attestation", &CertificateValidator{})
	svc.RegisterValidator("samlv2_response", NewSAMLValidator())

	return svc
}
