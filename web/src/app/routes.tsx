import { Routes, Route } from 'react-router-dom'
import { Layout } from '@/shared/components/Layout'
import { DashboardPage } from '@/features/dashboard/components/DashboardPage'
import { PoliciesPage } from '@/features/policies/components/PoliciesPage'
import { CreatePolicyPage } from '@/features/policies/components/CreatePolicyPage'
import { PolicyDetailPage } from '@/features/policies/components/PolicyDetailPage'
import {
  ClientWorkloadsPage,
  CreateClientWorkloadPage,
  ClientWorkloadDetailPage,
} from '@/features/client-workloads'
import {
  ServerWorkloadsPage,
  CreateServerWorkloadPage,
  ServerWorkloadDetailPage,
} from '@/features/server-workloads'
import {
  TrustProvidersPage,
  CreateTrustProviderPage,
  TrustProviderDetailPage,
} from '@/features/trust-providers'
import {
  AccessConditionsPage,
  CreateAccessConditionPage,
  AccessConditionDetailPage,
} from '@/features/access-conditions'
import {
  CredentialProvidersPage,
  CreateCredentialProviderPage,
  CredentialProviderDetailPage,
} from '@/features/credential-providers'
import { ReportingAccessAuthPage, ReportingAuditPage, ReportingWorkloadEventsPage, ReportingCompliancePage, ReportingMcpTracingPage, ReportingMcpLogsPage } from '@/features/reporting'
import {
  EdgeAgentControllersPage,
  EdgeAgentProxiesPage,
  EdgeDeployPage,
  EdgeTlsDecryptPage,
  CreateAgentControllerPage,
  AgentControllerDetailPage,
} from '@/features/edge-components'
import { HelpPage } from '@/features/help'
import { ProfilePage } from '@/features/profile'
import {
  AdminUsersPage,
  AdminRolesPage,
  AdminResourceSetsPage,
  AdminIdentityProvidersPage,
  AdminLogStreamsPage,
  AdminSignOnPolicyPage,
  AdminGlobalPolicyCompliancePage,
  CreateUserPage,
  UserDetailPage,
  CreateRolePage,
  RoleDetailPage,
  CreateResourceSetPage,
  ResourceSetDetailPage,
  CreateIdentityProviderPage,
  IdentityProviderDetailPage,
  CreateLogStreamPage,
  LogStreamDetailPage,
} from '@/features/administration'

import { LoginPage } from '@/features/auth/components/LoginPage'
import { SignupPage } from '@/features/auth/components/SignupPage'
import { ForgotPasswordPage } from '@/features/auth/components/ForgotPasswordPage'
import { ResetPasswordPage } from '@/features/auth/components/ResetPasswordPage'
import { SSOCallbackPage } from '@/features/auth/components/SSOCallbackPage'
import { AuthGuard } from '@/features/auth/components/AuthGuard'
import { ErrorBoundary } from '@/shared/components/ErrorBoundary'

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/auth/sso/callback" element={<SSOCallbackPage />} />
      <Route path="/forgot-password" element={<ForgotPasswordPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />
      <Route path="/*" element={
        <AuthGuard>
          <ErrorBoundary>
            <Layout>
            <Routes>
        <Route path="/" element={<DashboardPage />} />
        <Route path="/access-policies" element={<PoliciesPage />} />
        <Route path="/access-policies/new" element={<CreatePolicyPage />} />
        <Route path="/access-policies/:id" element={<PolicyDetailPage />} />
        <Route path="/client-workloads" element={<ClientWorkloadsPage />} />
        <Route path="/client-workloads/new" element={<CreateClientWorkloadPage />} />
        <Route path="/client-workloads/:id" element={<ClientWorkloadDetailPage />} />
        <Route path="/trust-providers" element={<TrustProvidersPage />} />
        <Route path="/trust-providers/new" element={<CreateTrustProviderPage />} />
        <Route path="/trust-providers/:id" element={<TrustProviderDetailPage />} />
        <Route path="/access-conditions" element={<AccessConditionsPage />} />
        <Route path="/access-conditions/new" element={<CreateAccessConditionPage />} />
        <Route path="/access-conditions/:id" element={<AccessConditionDetailPage />} />
        <Route path="/credential-providers" element={<CredentialProvidersPage />} />
        <Route path="/credential-providers/new" element={<CreateCredentialProviderPage />} />
        <Route path="/credential-providers/:id" element={<CredentialProviderDetailPage />} />
        <Route path="/server-workloads" element={<ServerWorkloadsPage />} />
        <Route path="/server-workloads/new" element={<CreateServerWorkloadPage />} />
        <Route path="/server-workloads/:id" element={<ServerWorkloadDetailPage />} />
        <Route path="/reporting/access-authorization" element={<ReportingAccessAuthPage />} />
        <Route path="/reporting/audit" element={<ReportingAuditPage />} />
        <Route path="/reporting/workload-events" element={<ReportingWorkloadEventsPage />} />
        <Route path="/reporting/compliance" element={<ReportingCompliancePage />} />
        <Route path="/reporting/mcp-tracing" element={<ReportingMcpTracingPage />} />
        <Route path="/reporting/mcp-logs" element={<ReportingMcpLogsPage />} />
        <Route path="/edge-components" element={<EdgeAgentControllersPage />} />
        <Route path="/edge-components/agent-controllers" element={<EdgeAgentControllersPage />} />
        <Route path="/edge-components/agent-proxies" element={<EdgeAgentProxiesPage />} />
        <Route path="/edge-components/deploy" element={<EdgeDeployPage />} />
        <Route path="/edge-components/tls-decrypt" element={<EdgeTlsDecryptPage />} />
        <Route
          path="/edge-components/agent_controllers/new"
          element={<CreateAgentControllerPage />}
        />
        <Route
          path="/edge-components/agent_controllers/:id"
          element={<AgentControllerDetailPage />}
        />
        <Route path="/administration/users" element={<AdminUsersPage />} />
        <Route path="/administration/roles" element={<AdminRolesPage />} />
        <Route path="/administration/resource-sets" element={<AdminResourceSetsPage />} />
        <Route path="/administration/identity-providers" element={<AdminIdentityProvidersPage />} />
        <Route path="/administration/log-streams" element={<AdminLogStreamsPage />} />
        <Route path="/administration/sign-on-policy" element={<AdminSignOnPolicyPage />} />
        <Route path="/administration/global-policy-compliance" element={<AdminGlobalPolicyCompliancePage />} />
        <Route path="/administration/new" element={<CreateUserPage />} />
        <Route path="/administration/users/new" element={<CreateUserPage />} />
        <Route path="/administration/users/:id" element={<UserDetailPage />} />
        <Route path="/administration/roles/new" element={<CreateRolePage />} />
        <Route path="/administration/roles/:id" element={<RoleDetailPage />} />
        <Route path="/administration/resource_sets/new" element={<CreateResourceSetPage />} />
        <Route path="/administration/resource_sets/:id" element={<ResourceSetDetailPage />} />
        <Route
          path="/administration/identity_providers/new"
          element={<CreateIdentityProviderPage />}
        />
        <Route
          path="/administration/identity_providers/:id"
          element={<IdentityProviderDetailPage />}
        />
        <Route path="/administration/log_streams/new" element={<CreateLogStreamPage />} />
        <Route path="/administration/log_streams/:id" element={<LogStreamDetailPage />} />
        <Route path="/help" element={<HelpPage />} />
        <Route path="/profile" element={<ProfilePage />} />
      </Routes>
    </Layout>
    </ErrorBoundary>
    </AuthGuard>
    } />
    </Routes>
  )
}
