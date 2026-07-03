import { Routes, Route } from 'react-router-dom'
import { Layout } from '@/shared/components/Layout'
import { PageHeader } from '@/shared/components/PageHeader'
import { DashboardPage } from '@/features/dashboard/components/DashboardPage'
import { PoliciesPage } from '@/features/policies/components/PoliciesPage'
import { CreatePolicyPage } from '@/features/policies/components/CreatePolicyPage'
import { PolicyDetailPage } from '@/features/policies/components/PolicyDetailPage'
import { ClientWorkloadsPage, CreateClientWorkloadPage, ClientWorkloadDetailPage } from '@/features/client-workloads'
import { ServerWorkloadsPage, CreateServerWorkloadPage, ServerWorkloadDetailPage } from '@/features/server-workloads'
import { TrustProvidersPage, CreateTrustProviderPage, TrustProviderDetailPage } from '@/features/trust-providers'

function Placeholder() {
  return (
    <div className="p-6">
      <PageHeader />
    </div>
  )
}

export function AppRoutes() {
  return (
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
        <Route path="/access-conditions" element={<Placeholder />} />
        <Route path="/credential-providers" element={<Placeholder />} />
        <Route path="/server-workloads" element={<ServerWorkloadsPage />} />
        <Route path="/server-workloads/new" element={<CreateServerWorkloadPage />} />
        <Route path="/server-workloads/:id" element={<ServerWorkloadDetailPage />} />
        <Route path="/reporting" element={<Placeholder />} />
        <Route path="/edge-components" element={<Placeholder />} />
        <Route path="/administration" element={<Placeholder />} />
        <Route path="/help" element={<Placeholder />} />
      </Routes>
    </Layout>
  )
}
