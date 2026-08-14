import { useState } from 'react'
import { ExistingCredentialProviderSelector } from '@/features/policies/components/tabs/credential-provider/ExistingCredentialProviderSelector'
import { CredentialProviderForm } from '@/shared/components/forms/CredentialProviderForm'
import { fetchCredentialProvider } from '@/features/policies/api/entityApi'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'

interface CredentialProviderTabProps {
  selectedId: string
  onSelect: (id: string) => void
  values: CredentialProviderFormValues
  onChange: (values: CredentialProviderFormValues) => void
  disabled?: boolean
}

export function CredentialProviderTab({
  selectedId,
  onSelect,
  values,
  onChange,
  disabled = false,
}: CredentialProviderTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')

  async function handleSelectExisting(id: string) {
    onSelect(id)
    try {
      const cp = await fetchCredentialProvider(id)
      // Copy all fields from the existing credential provider into the form,
      // preserving any field the fetched entity carries. The spread of `values`
      // provides defaults for any field not present on the response, ensuring
      // the form never has undefined values.
      const config = (cp.config ?? {}) as Record<string, unknown>
      onChange({
        ...values,
        name: cp.name ?? values.name,
        description: cp.description ?? values.description,
        credentialType: cp.type ?? values.credentialType,
        // Type-specific config — spread flat fields from config
        subject: (config.subject as string) ?? values.subject,
        audience: (config.audience as string) ?? values.audience,
        subjectValueType: (config.subjectValueType as string) ?? values.subjectValueType,
        audienceValueType: (config.audienceValueType as string) ?? values.audienceValueType,
        signingAlgorithm: (config.signingAlgorithm as string) ?? values.signingAlgorithm,
        lifetimeSeconds: (config.lifetimeSeconds as number) ?? values.lifetimeSeconds,
        apiKey: (config.apiKey as string) ?? values.apiKey,
        username: (config.username as string) ?? values.username,
        tokenEndpointUrl: (config.tokenEndpointUrl as string) ?? values.tokenEndpointUrl,
        clientId: (config.clientId as string) ?? values.clientId,
        scopes: (config.scopes as string) ?? values.scopes,
        awsIamRoleArn: (config.awsIamRoleArn as string) ?? values.awsIamRoleArn,
        customClaims: (config.customClaims as { name: string; value: string; valueType: string }[]) ?? values.customClaims,
        // Structured fields
        tags: cp.tags ?? values.tags,
      })
    } catch {
      // If fetch fails, just set the ID
    }
    setMode('new')
  }

  return (
    <div className="space-y-6">
      {!disabled && (
        <div className="flex items-center gap-4 border-b border-border pb-3">
          <button
            type="button"
            onClick={() => setMode('new')}
            className={`text-xs font-medium transition-colors ${mode === 'new' ? 'text-gray-700' : 'text-gray-500 hover:text-gray-700'}`}
          >
            Add New
          </button>
          <button
            type="button"
            onClick={() => setMode('existing')}
            className={`text-xs font-medium transition-colors ${mode === 'existing' ? 'text-gray-700' : 'text-gray-500 hover:text-gray-700'}`}
          >
            Select Existing
          </button>
        </div>
      )}

      {!disabled && mode === 'existing' ? (
        <ExistingCredentialProviderSelector
          selectedId={selectedId}
          onSelect={handleSelectExisting}
          onCancel={() => setMode('new')}
        />
      ) : (
        <CredentialProviderForm values={values} onChange={onChange} disabled={disabled} />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Credential Type Config — fully controlled, renders type-specific fields
// ---------------------------------------------------------------------------

import { FloatingInput, FloatingDropdown } from '@/shared/components/FloatingInput'
import { Dropdown } from '@/shared/components/Dropdown'

interface CredentialTypeConfigProps {
  type: string
  values: CredentialProviderFormValues
  onChange: (values: CredentialProviderFormValues) => void
}

export function CredentialTypeConfig({ type, values, onChange }: CredentialTypeConfigProps) {
  function update(partial: Partial<CredentialProviderFormValues>) {
    onChange({ ...values, ...partial })
  }

  switch (type) {
    case 'api_key':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="API Key"
            value={values.apiKey}
            onChange={(v) => update({ apiKey: v })}
            type="password"
          />
        </div>
      )

    case 'username_password':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="Username"
            value={values.username}
            onChange={(v) => update({ username: v })}
          />
          <FloatingInput
            label="Password"
            value={values.password}
            onChange={(v) => update({ password: v })}
            type="password"
          />
        </div>
      )

    case 'dipole_access_token':
      return (
        <div className="space-y-4">
          <ReadOnlyFieldWithCopy label="Audience" value={`<tenant>-api.dipolehq.com`} />
          <FloatingDropdown
            label="Role"
            options={[
              { value: 'superadmin', label: 'SuperAdmin' },
              { value: 'auditor', label: 'Auditor' },
            ]}
            value={values.audience}
            onChange={(v) => update({ audience: v })}
          />
          <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
            <input
              type="number"
              value={String(values.lifetimeSeconds)}
              onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) || 0 })}
              className="flex-1 px-3 py-2 text-sm focus:outline-none"
              placeholder="300"
            />
            <span className="px-3 text-xs text-gray-500 border-l border-border bg-gray-50 h-full flex items-center">
              Seconds
            </span>
          </div>
        </div>
      )

    case 'aws_sts_federation':
      return (
        <div className="space-y-4">
          <ReadOnlyFieldWithCopy
            label="OIDC Issuer URL"
            value="https://<tenant>-id.dipolehq.com"
          />
          <FloatingInput
            label="AWS IAM Role ARN"
            value={values.awsIamRoleArn}
            onChange={(v) => update({ awsIamRoleArn: v })}
          />
          <ReadOnlyFieldWithCopy label="Dipole IdP Token Audience" value="sts.amazonaws.com" />
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
              <input
                type="number"
                value={String(values.lifetimeSeconds)}
                onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) || 0 })}
                className="flex-1 px-3 py-2 text-sm focus:outline-none"
                placeholder="3600"
              />
              <span className="px-3 text-xs text-gray-500 border-l border-border bg-gray-50 h-full flex items-center">
                Seconds
              </span>
            </div>
          </div>
        </div>
      )

    case 'aws_secrets_manager':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="Credential Provider Integration"
            value={values.integrationId}
            onChange={(v) => update({ integrationId: v })}
          />
          <FloatingInput
            label="AWS Secrets Manager Secret ARN"
            value={values.secretArn}
            onChange={(v) => update({ secretArn: v })}
          />
          <FloatingDropdown
            label="Credential Value Type"
            options={[
              { value: 'plain_text', label: 'Plain Text' },
              { value: 'single', label: 'Single Value' },
              { value: 'username_password', label: 'Username/Password' },
            ]}
            value={values.credentialValueType}
            onChange={(v) => update({ credentialValueType: v })}
          />
          {values.credentialValueType === 'single' && (
            <FloatingInput
              label="Secret Key"
              value={values.secretKey}
              onChange={(v) => update({ secretKey: v })}
            />
          )}
          {values.credentialValueType === 'username_password' && (
            <>
              <FloatingInput
                label="Username Key"
                value={values.usernameKey}
                onChange={(v) => update({ usernameKey: v })}
              />
              <FloatingInput
                label="Password Key"
                value={values.passwordKey}
                onChange={(v) => update({ passwordKey: v })}
              />
            </>
          )}
          <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
            <input
              type="checkbox"
              checked={values.privateNetworkAccess}
              onChange={(e) => update({ privateNetworkAccess: e.target.checked })}
              className="w-3.5 h-3.5 rounded border-gray-300"
            />
            Private Network Access
          </label>
        </div>
      )

    case 'azure_entra_wif':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="OIDC Issuer URL"
            value="https://<tenant>-id.dipolehq.com"
            disabled
          />
          <FloatingInput
            label="Audience"
            value={values.audience}
            onChange={(v) => update({ audience: v })}
          />
          <FloatingInput
            label="JWT Token Subject"
            value={values.jwtTokenSubject}
            onChange={(v) => update({ jwtTokenSubject: v })}
          />
          <FloatingInput
            label="JWT Token Scope"
            value={values.jwtTokenScope}
            onChange={(v) => update({ jwtTokenScope: v })}
          />
          <FloatingInput
            label="Azure Tenant ID"
            value={values.azureTenantId}
            onChange={(v) => update({ azureTenantId: v })}
          />
          <FloatingInput
            label="Azure Client ID"
            value={values.azureClientId}
            onChange={(v) => update({ azureClientId: v })}
          />
        </div>
      )

    case 'azure_key_vault':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="Credential Provider Integration"
            value={values.integrationId}
            onChange={(v) => update({ integrationId: v })}
          />
          <FloatingInput
            label="Key Vault URL"
            value={values.keyVaultUrl}
            onChange={(v) => update({ keyVaultUrl: v })}
          />
          <FloatingDropdown
            label="Credential Value Type"
            options={[
              { value: 'single', label: 'Single Value' },
              { value: 'username_password', label: 'Username/Password' },
            ]}
            value={values.credentialValueType}
            onChange={(v) => update({ credentialValueType: v })}
          />
          <FloatingInput
            label="Secret Name"
            value={values.keyVaultSecretName}
            onChange={(v) => update({ keyVaultSecretName: v })}
          />
          {values.credentialValueType === 'username_password' && (
            <>
              <FloatingInput
                label="Username Key"
                value={values.usernameKey}
                onChange={(v) => update({ usernameKey: v })}
              />
              <FloatingInput
                label="Password Key"
                value={values.passwordKey}
                onChange={(v) => update({ passwordKey: v })}
              />
            </>
          )}
          <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
            <input
              type="checkbox"
              checked={values.privateNetworkAccess}
              onChange={(e) => update({ privateNetworkAccess: e.target.checked })}
              className="w-3.5 h-3.5 rounded border-gray-300"
            />
            Private Network Access
          </label>
        </div>
      )

    case 'claude_wif':
      return (
        <div className="space-y-4">
          <ReadOnlyFieldWithCopy
            label="OIDC Issuer URL"
            value="https://<tenant>-id.dipolehq.com"
          />
          <FloatingInput
            label="Federation Rule ID"
            value={values.claudeFederationRuleId}
            onChange={(v) => update({ claudeFederationRuleId: v })}
          />
          <FloatingInput
            label="Organization ID"
            value={values.claudeOrganizationId}
            onChange={(v) => update({ claudeOrganizationId: v })}
          />
          <FloatingInput
            label="Scope"
            value={values.claudeScope}
            onChange={(v) => update({ claudeScope: v })}
          />
          <FloatingInput
            label="Service Account ID"
            value={values.claudeServiceAccountId}
            onChange={(v) => update({ claudeServiceAccountId: v })}
          />
          <FloatingInput
            label="Audience"
            value={values.audience}
            onChange={(v) => update({ audience: v })}
          />
          <FloatingInput
            label="Workspace ID"
            value={values.claudeWorkspaceId}
            onChange={(v) => update({ claudeWorkspaceId: v })}
          />
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
              <input
                type="number"
                value={String(values.lifetimeSeconds)}
                onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) || 0 })}
                className="flex-1 px-3 py-2 text-sm focus:outline-none"
                placeholder="3600"
              />
              <span className="px-3 text-xs text-gray-500 border-l border-border bg-gray-50 h-full flex items-center">
                Seconds
              </span>
            </div>
          </div>
        </div>
      )

    case 'google_wif':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="OIDC Issuer URL"
            value="https://<tenant>-id.dipolehq.com"
            disabled
          />
          <FloatingInput
            label="Subject"
            value={values.subject}
            onChange={(v) => update({ subject: v })}
          />
          <FloatingInput
            label="Audience"
            value={values.audience}
            onChange={(v) => update({ audience: v })}
          />
          <FloatingInput
            label="GCP Service Account Email"
            value={values.gcpServiceAccountEmail}
            onChange={(v) => update({ gcpServiceAccountEmail: v })}
          />
          <FloatingInput
            label="GCP Project Number"
            value={values.gcpProjectNumber}
            onChange={(v) => update({ gcpProjectNumber: v })}
          />
          <FloatingInput
            label="Workload Identity Pool ID"
            value={values.gcpWorkloadIdentityPool}
            onChange={(v) => update({ gcpWorkloadIdentityPool: v })}
          />
          <FloatingInput
            label="WIF Provider ID"
            value={values.gcpWifProvider}
            onChange={(v) => update({ gcpWifProvider: v })}
          />
        </div>
      )

    case 'jwt':
      return (
        <div className="space-y-4">
          <FloatingDropdown
            label="Token Configuration"
            options={[{ value: 'snowflake_key_pair', label: 'Snowflake Key Pair Authentication' }]}
            value="snowflake_key_pair"
            onChange={() => {}}
            disabled
          />
          <div className="grid grid-cols-2 gap-3">
            <FloatingInput
              label="Snowflake Account ID"
              value={values.snowflakeAccountId}
              onChange={(v) => update({ snowflakeAccountId: v })}
              required
            />
            <FloatingInput
              label="Username"
              value={values.snowflakeUsername}
              onChange={(v) => update({ snowflakeUsername: v })}
              required
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">
              Snowflake Alter User Command
            </label>
            <div className="relative">
              <textarea
                readOnly
                rows={5}
                value=""
                placeholder="Snowflake Alter User Command"
                className="w-full px-3 py-2 pr-10 border border-border rounded-md text-sm text-gray-500 bg-gray-50 resize-y focus:outline-none"
              />
              <button
                type="button"
                onClick={() => {}}
                className="absolute top-2 right-2 text-gray-400 hover:text-gray-600"
                title="Copy"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2" />
                  <path
                    d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                    strokeWidth="2"
                  />
                </svg>
              </button>
            </div>
          </div>
        </div>
      )

    case 'jwt_svid':
      return (
        <div className="space-y-4">
          <InputWithBadge
            label="Subject"
            value={values.subject}
            onChange={(v) => update({ subject: v })}
            badgeValue="literal"
            onBadgeChange={() => {}}
          />
          <div className="grid grid-cols-[1fr_auto] gap-3">
            <ReadOnlyFieldWithCopy
              label="Issuer"
              value="https://<tenant>-id.dipolehq.com"
            />
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
              <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
                <input
                  type="number"
                  value={String(values.lifetimeSeconds)}
                  onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) || 0 })}
                  className="w-20 px-3 py-2 text-sm focus:outline-none"
                  placeholder="15"
                />
                <span className="px-3 text-xs text-gray-500 border-l border-border bg-gray-50 h-full flex items-center">
                  Minutes
                </span>
              </div>
            </div>
          </div>
          <FloatingDropdown
            label="Signing Algorithm Type"
            options={[
              { value: 'ES256', label: 'ES256' },
              { value: 'ES384', label: 'ES384' },
              { value: 'RS256', label: 'RS256' },
              { value: 'RS384', label: 'RS384' },
              { value: 'RS512', label: 'RS512' },
            ]}
            value={values.signingAlgorithm}
            onChange={(v) => update({ signingAlgorithm: v })}
          />
          <InputWithBadge
            label="Audience"
            value={values.audience}
            onChange={(v) => update({ audience: v })}
            badgeValue="literal"
            onBadgeChange={() => {}}
          />
        </div>
      )

    case 'managed_gitlab':
      return (
        <div className="space-y-4">
          <FloatingDropdown
            label="Select GitLab Integration"
            options={[]}
            value={values.integrationId}
            onChange={(v) => update({ integrationId: v })}
          />
          <FloatingInput
            label="Service Account Username"
            value={values.gitlabServiceAccountUsername}
            onChange={(v) => update({ gitlabServiceAccountUsername: v })}
          />
          <FloatingInput
            label="GitLab Group IDs or Paths"
            value={values.gitlabGroupId}
            onChange={(v) => update({ gitlabGroupId: v })}
          />
          <FloatingInput
            label="GitLab Project IDs or Paths"
            value={values.gitlabProjectId}
            onChange={(v) => update({ gitlabProjectId: v })}
          />
          <FloatingDropdown
            label="Access Level"
            options={[
              { value: 'no_access', label: 'No access' },
              { value: 'guest', label: 'Guest' },
              { value: 'planner', label: 'Planner' },
              { value: 'reporter', label: 'Reporter' },
              { value: 'developer', label: 'Developer' },
              { value: 'maintainer', label: 'Maintainer' },
              { value: 'owner', label: 'Owner' },
              { value: 'minimal_access', label: 'Minimal access' },
            ]}
            value={values.gitlabAccessLevel}
            onChange={(v) => update({ gitlabAccessLevel: v })}
          />
          <FloatingInput
            label="Scopes"
            value={values.gitlabScopes}
            onChange={(v) => update({ gitlabScopes: v })}
          />
          <div className="grid grid-cols-2 gap-3">
            <FloatingInput
              label="Rotation Time"
              value={String(values.gitlabRotationTime)}
              onChange={(v) => update({ gitlabRotationTime: Number(v) || 1 })}
              type="number"
            />
            <FloatingDropdown
              label="Unit"
              options={[
                { value: 'days', label: 'Days' },
                { value: 'hours', label: 'Hours' },
              ]}
              value={values.gitlabRotationUnit}
              onChange={(v) => update({ gitlabRotationUnit: v })}
            />
          </div>
        </div>
      )

    case 'mcp_user_access_token':
      return (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <div className="flex-1">
              <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
                <span className="px-3 text-xs text-gray-400 bg-gray-50 border-r border-border h-full flex items-center">
                  https://
                </span>
                <input
                  type="text"
                  placeholder="MCP Server URL"
                  value={values.mcpServerUrl}
                  onChange={(e) => update({ mcpServerUrl: e.target.value })}
                  className="flex-1 px-3 py-2 text-sm focus:outline-none"
                />
              </div>
            </div>
            <button
              type="button"
              className="px-4 py-2 text-sm text-gray-500 bg-gray-100 border border-border rounded-md hover:bg-gray-200"
            >
              Discover
            </button>
          </div>
          <ReadOnlyFieldWithCopy
            label="Callback URL"
            value="https://<tenant>-id.dipolehq.com/mcp-auth/userauth/<id>/callback"
          />
          <FloatingInput
            label="Client ID"
            value={values.clientId}
            onChange={(v) => update({ clientId: v })}
          />
          <FloatingInput
            label="Client Secret"
            value={values.clientSecret}
            onChange={(v) => update({ clientSecret: v })}
            type="password"
          />
          <FloatingInput
            label="Scopes"
            value={values.scopes}
            onChange={(v) => update({ scopes: v })}
          />
          <FloatingInput
            label="Resource"
            value={values.mcpResource}
            onChange={(v) => update({ mcpResource: v })}
          />
          <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
            <span className="px-3 text-xs text-gray-400 bg-gray-50 border-r border-border h-full flex items-center">
              https://
            </span>
            <input
              type="text"
              placeholder="Authorization URL"
              value={values.authorizationUrl}
              onChange={(e) => update({ authorizationUrl: e.target.value })}
              className="flex-1 px-3 py-2 text-sm focus:outline-none"
            />
          </div>
          <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
            <span className="px-3 text-xs text-gray-400 bg-gray-50 border-r border-border h-full flex items-center">
              https://
            </span>
            <input
              type="text"
              placeholder="Token URL"
              value={values.tokenEndpointUrl}
              onChange={(e) => update({ tokenEndpointUrl: e.target.value })}
              className="flex-1 px-3 py-2 text-sm focus:outline-none"
            />
          </div>
          <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
            <span className="px-3 text-xs text-gray-400 bg-gray-50 border-r border-border h-full flex items-center">
              https://
            </span>
            <input
              type="text"
              placeholder="Introspection URL"
              value={values.introspectionUrl}
              onChange={(e) => update({ introspectionUrl: e.target.value })}
              className="flex-1 px-3 py-2 text-sm focus:outline-none"
            />
          </div>
          <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
            <input
              type="checkbox"
              checked={values.mcpPkceRequired}
              onChange={(e) => update({ mcpPkceRequired: e.target.checked })}
              className="w-3.5 h-3.5 rounded border-gray-300"
            />
            PKCE Required
          </label>
        </div>
      )

    case 'oauth_authorization_code':
      return (
        <div className="space-y-4">
          <ReadOnlyFieldWithCopy
            label="Callback URL"
            value="https://<tenant>-api.dipolehq.com/api/v2/credential-providers/callback"
          />
          <FloatingInput
            label="Client ID"
            value={values.clientId}
            onChange={(v) => update({ clientId: v })}
          />
          <FloatingInput
            label="Client Secret"
            value={values.clientSecret}
            onChange={(v) => update({ clientSecret: v })}
            type="password"
          />
          <FloatingInput
            label="Scopes"
            value={values.scopes}
            onChange={(v) => update({ scopes: v })}
          />
          <div className="flex items-center gap-2">
            <div className="flex-1">
              <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
                <span className="px-3 text-xs text-gray-400 bg-gray-50 border-r border-border h-full flex items-center">
                  https://
                </span>
                <input
                  type="text"
                  placeholder="OAuth URL"
                  value={values.oauthUrl}
                  onChange={(e) => update({ oauthUrl: e.target.value })}
                  className="flex-1 px-3 py-2 text-sm focus:outline-none"
                />
              </div>
            </div>
            <button
              type="button"
              className="px-4 py-2 text-sm text-gray-500 bg-gray-100 border border-border rounded-md hover:bg-gray-200"
            >
              URL Discovery
            </button>
          </div>
          <div className="space-y-3">
            <FloatingInput
              label="Authorization URL"
              value={values.authorizationUrl}
              onChange={(v) => update({ authorizationUrl: v })}
            />
            <FloatingInput
              label="Token URL"
              value={values.tokenEndpointUrl}
              onChange={(v) => update({ tokenEndpointUrl: v })}
            />
            <FloatingInput
              label="Introspection URL"
              value={values.introspectionUrl}
              onChange={(v) => update({ introspectionUrl: v })}
            />
          </div>
          <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
            <input
              type="checkbox"
              checked={values.oauthPkceRequired}
              onChange={(e) => update({ oauthPkceRequired: e.target.checked })}
              className="w-3.5 h-3.5 rounded border-gray-300"
            />
            PKCE Required
          </label>
          <div className="flex items-center gap-3">
            <span className="text-sm font-medium text-gray-700">Lifetime:</span>
            <input
              type="number"
              value={String(values.oauthLifetimeValue)}
              onChange={(e) => update({ oauthLifetimeValue: Number(e.target.value) || 1 })}
              className="w-20 px-3 py-2 border border-border rounded-md text-sm focus:outline-none"
              placeholder="1"
            />
            <FloatingDropdown
              label="Unit"
              options={[
                { value: 'years', label: 'Years' },
                { value: 'months', label: 'Months' },
                { value: 'days', label: 'Days' },
              ]}
              value={values.oauthLifetimeUnit}
              onChange={(v) => update({ oauthLifetimeUnit: v })}
            />
          </div>
        </div>
      )

    case 'oauth_client_credentials':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="Token Endpoint URL"
            value={values.tokenEndpointUrl}
            onChange={(v) => update({ tokenEndpointUrl: v })}
          />
          <FloatingInput
            label="Client ID"
            value={values.clientId}
            onChange={(v) => update({ clientId: v })}
          />
          <FloatingInput
            label="Client Secret"
            value={values.clientSecret}
            onChange={(v) => update({ clientSecret: v })}
            type="password"
          />
          <FloatingInput
            label="Scopes (Optional)"
            value={values.scopes}
            onChange={(v) => update({ scopes: v })}
          />
          <FloatingDropdown
            label="Credential Style"
            options={[
              { value: 'authorization_header', label: 'Authorization Header' },
              { value: 'post_body', label: 'POST Body' },
            ]}
            value={values.credentialStyle}
            onChange={(v) => update({ credentialStyle: v })}
          />
        </div>
      )

    case 'oidc_id_token':
      return (
        <div className="space-y-4">
          <InputWithBadge
            label="Subject"
            value={values.subject}
            onChange={(v) => update({ subject: v })}
            badgeValue={values.subjectValueType || 'literal'}
            onBadgeChange={(v) => update({ subjectValueType: v })}
          />
          <div className="grid grid-cols-[1fr_auto] gap-3">
            <ReadOnlyFieldWithCopy
              label="Issuer"
              value="https://<tenant>-id.dipolehq.com"
            />
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
              <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
                <input
                  type="number"
                  value={String(values.lifetimeSeconds)}
                  onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) || 0 })}
                  className="w-20 px-3 py-2 text-sm focus:outline-none"
                  placeholder="15"
                />
                <span className="px-3 text-xs text-gray-500 border-l border-border bg-gray-50 h-full flex items-center">
                  Minutes
                </span>
              </div>
            </div>
          </div>
          <FloatingDropdown
            label="Signing Algorithm Type"
            options={[
              { value: 'ES256', label: 'ES256' },
              { value: 'ES384', label: 'ES384' },
              { value: 'RS256', label: 'RS256' },
              { value: 'RS384', label: 'RS384' },
              { value: 'RS512', label: 'RS512' },
            ]}
            value={values.signingAlgorithm}
            onChange={(v) => update({ signingAlgorithm: v })}
          />
          <InputWithBadge
            label="Audience"
            value={values.audience}
            onChange={(v) => update({ audience: v })}
            badgeValue={values.audienceValueType || 'literal'}
            onBadgeChange={(v) => update({ audienceValueType: v })}
          />
        </div>
      )

    case 'vault_client_token':
      return (
        <div className="space-y-5">
          <div>
            <h5 className="text-sm font-semibold text-gray-600 mb-3">JSON Web Token (JWT)</h5>
            <div className="space-y-4">
              <InputWithBadge
                label="Subject"
                value={values.subject}
                onChange={(v) => update({ subject: v })}
                badgeValue="literal"
                onBadgeChange={() => {}}
              />
              <div className="grid grid-cols-[1fr_auto] gap-3">
                <ReadOnlyFieldWithCopy
                  label="Issuer"
                  value="https://<tenant>-id.dipolehq.com"
                />
                <div>
                  <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
                  <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
                    <input
                      type="number"
                      value={String(values.lifetimeSeconds)}
                      onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) || 0 })}
                      className="w-20 px-3 py-2 text-sm focus:outline-none"
                      placeholder="1"
                    />
                    <span className="px-3 text-xs text-gray-500 border-l border-border bg-gray-50 h-full flex items-center">
                      Minutes
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className="border border-border rounded-lg p-5">
            <h5 className="text-sm font-semibold text-gray-600 mb-4">Vault Authentication</h5>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <FloatingInput
                  label="Host"
                  value={values.vaultHost}
                  onChange={(v) => update({ vaultHost: v })}
                />
                <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
                  <input
                    type="text"
                    placeholder="8200"
                    value={values.vaultPort}
                    onChange={(e) => update({ vaultPort: e.target.value })}
                    className="flex-1 px-3 py-2 text-sm focus:outline-none"
                  />
                  <label className="flex items-center gap-1.5 px-3 border-l border-border cursor-pointer">
                    <input
                      type="checkbox"
                      checked={values.vaultTls}
                      onChange={(e) => update({ vaultTls: e.target.checked })}
                      className="w-3.5 h-3.5 rounded border-gray-300"
                    />
                    <span className="text-xs text-gray-600">TLS</span>
                  </label>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <FloatingInput
                  label="Authentication Path"
                  value={values.vaultAuthPath}
                  onChange={(v) => update({ vaultAuthPath: v })}
                />
                <FloatingInput
                  label="Role"
                  value={values.vaultRole}
                  onChange={(v) => update({ vaultRole: v })}
                />
              </div>
              <FloatingInput
                label="Namespace"
                value={values.vaultNamespace}
                onChange={(v) => update({ vaultNamespace: v })}
              />
              <div className="flex items-center gap-4">
                <div className="flex-1">
                  <FloatingDropdown
                    label="Forwarding Configuration"
                    options={[
                      { value: 'none', label: 'No Forwarding (default)' },
                      { value: 'unconditional', label: 'Unconditional Forwarding' },
                      { value: 'conditional', label: 'Conditional Forwarding' },
                    ]}
                    value={values.vaultForwardingConfig}
                    onChange={(v) => update({ vaultForwardingConfig: v })}
                  />
                </div>
                <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer shrink-0">
                  <input
                    type="checkbox"
                    checked={values.privateNetworkAccess}
                    onChange={(e) => update({ privateNetworkAccess: e.target.checked })}
                    className="w-4 h-4 rounded border-gray-300"
                  />
                  <span>Private Network Access</span>
                </label>
              </div>
            </div>
          </div>
        </div>
      )

    case 'x509_svid':
      return (
        <div className="space-y-4">
          <InputWithBadge
            label="Subject"
            value={values.subject}
            onChange={(v) => update({ subject: v })}
            badgeValue="literal"
            onBadgeChange={() => {}}
          />
          <InputWithBadge
            label="Spiffe ID"
            value={values.spiffeId}
            onChange={(v) => update({ spiffeId: v })}
            badgeValue="literal"
            onBadgeChange={() => {}}
          />
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px] w-fit">
              <input
                type="number"
                value={String(values.lifetimeSeconds)}
                onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) || 0 })}
                className="w-20 px-3 py-2 text-sm focus:outline-none"
                placeholder="15"
              />
              <span className="px-3 text-xs text-gray-500 border-l border-border bg-gray-50 h-full flex items-center">
                Minutes
              </span>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FloatingDropdown
              label="Key Usage"
              options={[
                { value: 'digital_signature', label: 'Digital Signature' },
                { value: 'key_encipherment', label: 'Key Encipherment' },
              ]}
              value={values.keyUsage}
              onChange={(v) => update({ keyUsage: v })}
            />
            <FloatingDropdown
              label="Extended Key Usage"
              options={[
                { value: 'id-kp-serverAuth', label: 'id-kp-serverAuth' },
                { value: 'id-kp-clientAuth', label: 'id-kp-clientAuth' },
              ]}
              value={values.extendedKeyUsage[0] || ''}
              onChange={(v) => update({ extendedKeyUsage: [v] })}
            />
          </div>
        </div>
      )

    case 'openai_wif':
      return (
        <div className="space-y-4">
          <ReadOnlyFieldWithCopy
            label="OIDC Issuer URL"
            value="https://<tenant>-id.dipolehq.com"
          />
          <FloatingInput
            label="Identity Provider ID"
            value={values.openaiIdentityProviderId}
            onChange={(v) => update({ openaiIdentityProviderId: v })}
          />
          <FloatingInput
            label="Service Account ID"
            value={values.openaiServiceAccountId}
            onChange={(v) => update({ openaiServiceAccountId: v })}
          />
          <FloatingInput
            label="Audience"
            value={values.audience}
            onChange={(v) => update({ audience: v })}
          />
        </div>
      )

    default:
      return (
        <p className="text-xs text-gray-400 italic">
          Unknown credential type. Please select a valid type.
        </p>
      )
  }
}

function ReadOnlyFieldWithCopy({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <label className="block text-[11px] text-gray-500 mb-1">{label}</label>
      <div className="flex items-center border border-border rounded-md overflow-hidden bg-gray-50">
        <input
          type="text"
          readOnly
          value={value}
          className="flex-1 px-3 py-2 text-sm text-gray-500 bg-gray-50 focus:outline-none"
        />
        <button
          type="button"
          onClick={() => navigator.clipboard.writeText(value)}
          className="px-3 py-2 border-l border-border text-gray-400 hover:text-gray-600"
          title="Copy"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2" />
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" strokeWidth="2" />
          </svg>
        </button>
      </div>
    </div>
  )
}

function InputWithBadge({
  label,
  value,
  onChange,
  badgeValue,
  onBadgeChange,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  badgeValue: string
  onBadgeChange?: (v: string) => void
}) {
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1">
        <FloatingInput label={label} value={value} onChange={onChange} />
      </div>
      <div className="w-28 shrink-0">
        <Dropdown
          options={[
            { value: 'literal', label: 'Literal' },
            { value: 'dynamic', label: 'Dynamic' },
          ]}
          value={badgeValue}
          onChange={(v) => onBadgeChange?.(v)}
        />
      </div>
    </div>
  )
}
