// factory.go creates a fully-configured TrustValidationService with all type-specific validators registered.
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
	oidcValidator := NewOIDCValidator(encryptionKey)
	svc.RegisterValidator("gitlab_job", oidcValidator)
	svc.RegisterValidator("terraform_cloud", oidcValidator)
	svc.RegisterValidator("oidc_id_token", oidcValidator)
	svc.RegisterValidator("aws_metadata_service", &AWSIMDSValidator{})
	svc.RegisterValidator("aws_role", NewAWSRoleValidator())
	svc.RegisterValidator("azure_metadata", NewAzureValidator())
	svc.RegisterValidator("gcp_identity_token", &GCPValidator{})
	svc.RegisterValidator("kerberos", NewKerberosValidator(controllerRepo, encryptionKey))
	svc.RegisterValidator("certificate_signed_attestation", &CertificateValidator{})
	svc.RegisterValidator("samlv2_response", NewSAMLValidator())

	return svc
}
