export const POLICY_TABLE_COLUMNS = [
  { key: 'number', label: '#' },
  { key: 'name', label: 'Access Policy Name' },
  { key: 'clientWorkload', label: 'Client Workload' },
  { key: 'trustProvider', label: 'Trust Provider' },
  { key: 'accessCondition', label: 'Access Condition' },
  { key: 'credentialProvider', label: 'Credential Provider' },
  { key: 'serverWorkload', label: 'Server Workload' },
  { key: 'active', label: 'Status' },
] as const

export const PAGE_SIZE_OPTIONS = [15, 25, 50] as const

export const CLIENT_IDENTIFIER_TYPES = [
  { value: 'k8s_namespace', label: 'Kubernetes Namespace' },
  { value: 'k8s_pod_name', label: 'Kubernetes Pod Name' },
  { value: 'k8s_pod_name_prefix', label: 'Kubernetes Pod Name Prefix' },
  { value: 'k8s_service_account_name', label: 'Kubernetes Service Account Name' },
  { value: 'k8s_service_account_uid', label: 'Kubernetes Service Account UID' },
  { value: 'hostname', label: 'Hostname' },
  { value: 'source_ip', label: 'Source IP' },
  { value: 'process_name', label: 'Process Name' },
  { value: 'process_path', label: 'Process Path' },
  { value: 'process_command_line', label: 'Process Command Line' },
  { value: 'process_user_name', label: 'Process User Name' },
  { value: 'aws_account_id', label: 'AWS Account ID' },
  { value: 'aws_ec2_instance_id', label: 'AWS EC2 Instance ID' },
  { value: 'aws_ecs_task_family', label: 'AWS ECS Task Family' },
  { value: 'aws_ecs_service_name', label: 'AWS ECS Service Name' },
  { value: 'aws_lambda_arn', label: 'AWS Lambda ARN' },
  { value: 'aws_region', label: 'AWS Region' },
  { value: 'azure_subscription_id', label: 'Azure Subscription ID' },
  { value: 'azure_vm_id', label: 'Azure VM ID' },
  { value: 'gcp_identity_token', label: 'GCP Identity Token' },
  { value: 'github_id_token_repository', label: 'GitHub ID Token Repository' },
  { value: 'github_id_token_subject', label: 'GitHub ID Token Subject' },
  { value: 'gitlab_id_token_project_path', label: 'GitLab ID Token Project Path' },
  { value: 'gitlab_id_token_namespace_path', label: 'GitLab ID Token Namespace Path' },
  { value: 'gitlab_id_token_ref_path', label: 'GitLab ID Token Ref Path' },
  { value: 'gitlab_id_token_subject', label: 'GitLab ID Token Subject' },
  { value: 'terraform_cloud_id_token_organization_id', label: 'Terraform Cloud Organization ID' },
  { value: 'terraform_cloud_id_token_project_id', label: 'Terraform Cloud Project ID' },
  { value: 'terraform_cloud_id_token_workspace_id', label: 'Terraform Cloud Workspace ID' },
  { value: 'oidc_id_token_issuer', label: 'OIDC ID Token Issuer' },
  { value: 'oidc_id_token_subject', label: 'OIDC ID Token Subject' },
  { value: 'oidc_id_token_audience', label: 'OIDC ID Token Audience' },
  { value: 'redirect_uri', label: 'Redirect URI' },
  { value: 'dipole_client_id', label: 'Dipole Client ID' },
] as const

export const APP_PROTOCOLS = [
  { value: 'HTTP', label: 'HTTP' },
  { value: 'OAuth', label: 'OAuth' },
  { value: 'PostgreSQL', label: 'PostgreSQL' },
  { value: 'MySQL', label: 'MySQL' },
  { value: 'Redis', label: 'Redis' },
  { value: 'Snowflake', label: 'Snowflake' },
  { value: 'OracleDatabase', label: 'Oracle Database' },
  { value: 'Redshift', label: 'Redshift' },
  { value: 'MCP', label: 'MCP' },
] as const

export const TRANSPORT_PROTOCOLS = [
  { value: 'TCP', label: 'TCP' },
] as const

export const AUTH_METHODS = [
  { value: 'no_auth', label: 'No Authentication' },
  { value: 'http_auth', label: 'HTTP Authentication' },
  { value: 'api_key', label: 'API Key' },
  { value: 'mtls_x509', label: 'mTLS Authentication' },
  { value: 'password_auth', label: 'Password Authentication' },
] as const

export const TRUST_PROVIDER_TYPES = [
  { value: 'aws_metadata', label: 'AWS Metadata Service' },
  { value: 'aws_role', label: 'AWS Role' },
  { value: 'azure_metadata', label: 'Azure Instance Metadata Service' },
  { value: 'gcp_identity_token', label: 'GCP Identity Token' },
  { value: 'kubernetes_service_account', label: 'Kubernetes Service Account' },
  { value: 'github_action', label: 'GitHub Action ID Token' },
  { value: 'gitlab_job', label: 'GitLab Job ID Token' },
  { value: 'terraform_cloud', label: 'Terraform Cloud Identity Token' },
  { value: 'kerberos', label: 'Kerberos' },
  { value: 'oidc_id_token', label: 'OIDC ID Token' },
  { value: 'samlv2_response', label: 'SAMLv2 Response' },
  { value: 'certificate_signed', label: 'Certificate Signed Attestation' },
] as const
