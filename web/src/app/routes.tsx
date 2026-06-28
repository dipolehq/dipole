import { Routes, Route } from 'react-router-dom'
import { IdentityListPage } from '@/features/identities/components/IdentityListPage'

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<IdentityListPage />} />
    </Routes>
  )
}
