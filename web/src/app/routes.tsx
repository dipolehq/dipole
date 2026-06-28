import { Routes, Route } from 'react-router-dom'
import { Layout } from '@/shared/components/Layout'

function Placeholder({ title }: { title: string }) {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-semibold text-gray-900">{title}</h1>
    </div>
  )
}

export function AppRoutes() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Placeholder title="Dashboard" />} />
        <Route path="/access-policies" element={<Placeholder title="Access Policies" />} />
        <Route path="/client-workloads" element={<Placeholder title="Client Workloads" />} />
        <Route path="/trust-providers" element={<Placeholder title="Trust Providers" />} />
        <Route path="/access-conditions" element={<Placeholder title="Access Conditions" />} />
        <Route path="/credential-providers" element={<Placeholder title="Credential Providers" />} />
        <Route path="/server-workloads" element={<Placeholder title="Server Workloads" />} />
        <Route path="/reporting" element={<Placeholder title="Reporting" />} />
        <Route path="/edge-components" element={<Placeholder title="Edge Components" />} />
        <Route path="/administration" element={<Placeholder title="Administration" />} />
        <Route path="/help" element={<Placeholder title="Help" />} />
      </Routes>
    </Layout>
  )
}
