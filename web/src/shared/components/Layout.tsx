import { type ReactNode, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { Sidebar } from '@/shared/components/Sidebar'
import { PageHeader } from '@/shared/components/PageHeader'
import { PageHeaderActionsProvider, usePageHeaderActionsValue } from '@/shared/context/PageHeaderActionsContext'

interface LayoutProps {
  children: ReactNode
}

const PAGE_TITLES: Record<string, string> = {
  '/': 'Dashboard',
  '/access-policies': 'Access Policies',
  '/client-workloads': 'Client Workloads',
  '/server-workloads': 'Server Workloads',
  '/trust-providers': 'Trust Providers',
  '/access-conditions': 'Access Conditions',
  '/credential-providers': 'Credential Providers',
  '/reporting/access-authorization': 'Access Authorization',
  '/reporting/audit': 'Audit Logs',
  '/reporting/workload-events': 'Workload Events',
  '/reporting/compliance': 'Compliance',
  '/reporting/mcp-logs': 'MCP Logs',
  '/edge-components': 'Edge Components',
  '/edge-components/agent-controllers': 'Agent Controllers',
  '/edge-components/agent-proxies': 'Agent Proxies',
  '/edge-components/deploy': 'Deploy',
  '/edge-components/tls-decrypt': 'TLS Decrypt',
  '/administration/users': 'Users',
  '/administration/roles': 'Roles',
  '/administration/resource-sets': 'Resource Sets',
  '/administration/identity-providers': 'Identity Providers',
  '/administration/log-streams': 'Log Streams',
  '/administration/sign-on-policy': 'Sign-On Policy',
  '/administration/global-policy-compliance': 'Global Policy Compliance',
  '/profile': 'Profile',
  '/docs': 'Documentation',
  '/support': 'Support',
}

function getPageTitle(pathname: string): string {
  if (PAGE_TITLES[pathname]) return PAGE_TITLES[pathname]
  // Match detail/create pages by prefix
  for (const [path, title] of Object.entries(PAGE_TITLES)) {
    if (pathname.startsWith(path + '/')) return title
  }
  return 'Workload IAM'
}

export function Layout({ children }: LayoutProps) {
  const location = useLocation()

  useEffect(() => {
    document.title = `Dipole | ${getPageTitle(location.pathname)}`
  }, [location.pathname])

  return (
    <div className="h-screen bg-surface-secondary overflow-hidden flex">
      <Sidebar />
      <PageHeaderActionsProvider>
        <main className="flex-1 ml-[204px] h-screen overflow-hidden flex flex-col">
          <PageHeaderBar />
          {children}
        </main>
      </PageHeaderActionsProvider>
    </div>
  )
}

function PageHeaderBar() {
  const actions = usePageHeaderActionsValue()
  const height = actions ? 'h-[60px]' : 'h-[44px]'

  return (
    <div className={`flex items-center justify-between px-4 shrink-0 ${height} border-b border-border bg-white`}>
      <PageHeader />
      {actions}
    </div>
  )
}
