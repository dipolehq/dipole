/*
factory.go creates a fully-configured TrustValidationService with all
type-specific validators registered. Called during Cloud service initialization.
*/
package trust

import "log/slog"

func NewConfiguredService(logger *slog.Logger) *TrustValidationService {
	cache := NewJWKSCache(logger)
	svc := NewTrustValidationService(cache, logger)

	svc.RegisterValidator("kubernetes_service_account", &K8sValidator{})
	svc.RegisterValidator("github_action", &GitHubValidator{})
	svc.RegisterValidator("gitlab_job", &OIDCValidator{})
	svc.RegisterValidator("terraform_cloud", &OIDCValidator{})
	svc.RegisterValidator("oidc_id_token", &OIDCValidator{})

	return svc
}
