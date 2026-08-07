/*
edge_sdk.go auto-generates Edge SDK Client ID and Audience for Trust Providers.

GenerateEdgeSDKClientID creates the identifier used by CLI/Edge API:

	dipole:<region>:<tenantSubdomain>:identity:<type_short>:<providerUUID>

GenerateEdgeSDKAudience creates the OIDC audience value:

	https://<tenantSubdomain>-id.dipolehq.com

Both are immutable after Trust Provider creation.
*/
package trust

import (
	"fmt"

	"github.com/google/uuid"
)

func GenerateEdgeSDKClientID(region, tenantSubdomain, tpType string, tpID uuid.UUID) string {
	return fmt.Sprintf("dipole:%s:%s:identity:%s:%s", region, tenantSubdomain, typeShortName(tpType), tpID.String())
}

func GenerateEdgeSDKAudience(region, tenantSubdomain string) string {
	return fmt.Sprintf("https://%s-id.dipolehq.com", tenantSubdomain)
}

func typeShortName(tpType string) string {
	switch tpType {
	case "kubernetes_service_account":
		return "kubernetes_serviceaccount_token"
	case "github_action":
		return "github_idtoken"
	case "gitlab_job":
		return "gitlab_idtoken"
	case "oidc_id_token":
		return "oidc_id_token"
	case "terraform_cloud":
		return "terraform_cloud_idtoken"
	case "aws_metadata_service":
		return "aws_metadata"
	case "aws_role":
		return "aws_role"
	case "kerberos":
		return "kerberos"
	case "certificate_signed_attestation":
		return "certificate"
	case "samlv2_response":
		return "saml"
	default:
		return tpType
	}
}
