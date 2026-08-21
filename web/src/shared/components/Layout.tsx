import { type ReactNode, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { Sidebar } from '@/shared/components/Sidebar'

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
      <main className="flex-1 ml-[204px] h-screen overflow-hidden flex flex-col">{children}</main>
    </div>
  )
}
