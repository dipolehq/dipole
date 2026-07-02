import { Routes, Route } from 'react-router-dom'
import { Layout } from '@/shared/components/Layout'
import { PageHeader } from '@/shared/components/PageHeader'
import { DashboardPage } from '@/features/dashboard/components/DashboardPage'
import { PoliciesPage } from '@/features/policies/components/PoliciesPage'
import { CreatePolicyPage } from '@/features/policies/components/CreatePolicyPage'
import { PolicyDetailPage } from '@/features/policies/components/PolicyDetailPage'

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
        <Route path="/client-workloads" element={<Placeholder />} />
        <Route path="/trust-providers" element={<Placeholder />} />
        <Route path="/access-conditions" element={<Placeholder />} />
        <Route path="/credential-providers" element={<Placeholder />} />
        <Route path="/server-workloads" element={<Placeholder />} />
        <Route path="/reporting" element={<Placeholder />} />
        <Route path="/edge-components" element={<Placeholder />} />
        <Route path="/administration" element={<Placeholder />} />
        <Route path="/help" element={<Placeholder />} />
      </Routes>
    </Layout>
  )
}
