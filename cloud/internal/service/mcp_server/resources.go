/*
resources.go provides static MCP resources (metadata) for AI agents.

ResourceService returns valid filter values (enums) that help AI agents
construct correct tool calls. All content is static and tenant-independent.
*/
package mcp_server

import "fmt"

type ResourceService struct{}

func NewResourceService() *ResourceService {
	return &ResourceService{}
}

func (s *ResourceService) ListResources() any {
	return map[string]any{
		"resources": []map[string]any{
			{"uri": "dipole://audit-log-severities", "name": "Audit Log Severities", "mimeType": "application/json"},
			{"uri": "dipole://audit-log-categories", "name": "Audit Log Categories", "mimeType": "application/json"},
			{"uri": "dipole://auth-event-types", "name": "Authorization Event Types", "mimeType": "application/json"},
			{"uri": "dipole://auth-event-severities", "name": "Authorization Event Severities", "mimeType": "application/json"},
			{"uri": "dipole://workload-event-types", "name": "Workload Event Types", "mimeType": "application/json"},
			{"uri": "dipole://workload-event-severities", "name": "Workload Event Severities", "mimeType": "application/json"},
		},
	}
}

func (s *ResourceService) ReadResource(uri string) (any, error) {
	content, ok := resourceContent[uri]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", uri)
	}
	return map[string]any{
		"contents": []map[string]any{
			{"uri": uri, "mimeType": "application/json", "text": content},
		},
	}, nil
}

var resourceContent = map[string]any{
	"dipole://audit-log-severities": []map[string]any{
		{"value": "Info", "description": "Routine administrative activity."},
		{"value": "Warn", "description": "Events that may indicate a configuration issue or unusual activity."},
		{"value": "Alert", "description": "High-importance events requiring attention."},
	},
	"dipole://audit-log-categories": []string{
		"Unknown", "Tenant", "Users", "Authentication", "Workloads",
		"AccessPolicies", "Agents", "CredentialProvider", "TrustProvider",
		"AgentControllers", "LogStreams", "Integrations", "AccessConditions",
		"Roles", "IdentityProviders", "ResourceSets", "Analysis", "Routing",
		"SignOnPolicies", "PkiSettings", "StandaloneCertificateAuthorities",
		"CredentialProviderIntegrations", "DiscoveryIntegration", "GlobalPolicyCompliance",
	},
	"dipole://auth-event-types": []map[string]any{
		{"value": "Request", "description": "Captures the incoming access request and associated metadata."},
		{"value": "Authorization", "description": "Records the outcome of Access Policy evaluation, including Trust Provider and Access Condition results."},
		{"value": "Credential", "description": "Records the result of credential retrieval from the Credential Provider."},
	},
	"dipole://auth-event-severities": []map[string]any{
		{"value": "Error", "description": "Authorization failed due to an internal error."},
		{"value": "Alert", "description": "High-priority authorization event requiring attention."},
		{"value": "Warn", "description": "Concerning authorization activity."},
		{"value": "Info", "description": "Standard, expected authorization activity."},
	},
	"dipole://workload-event-types": []string{
		"Redshift", "HTTP", "MySQL", "Postgres", "Redis", "Snowflake", "TCP", "OracleDatabase", "MCP",
	},
	"dipole://workload-event-severities": []map[string]any{
		{"value": "Error", "description": "Workload connection failed or encountered an error."},
		{"value": "Alert", "description": "High-priority workload event requiring attention."},
		{"value": "Warn", "description": "Concerning workload activity."},
		{"value": "Info", "description": "Standard, expected workload activity."},
	},
}
