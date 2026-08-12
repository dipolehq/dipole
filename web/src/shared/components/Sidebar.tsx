import { useState } from 'react'
import { NavLink, useNavigate, useLocation } from 'react-router-dom'
import {
  LayoutGrid,
  ArrowLeftRight,
  Monitor,
  Server,
  CheckCircle,
  Lock,
  Clock,
  Settings,
  BarChart3,
  Users,
  BookOpen,
  HelpCircle,
  User,
  ChevronRight,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { WorkspaceSwitcher } from '@/shared/components/WorkspaceSwitcher'
import { TenantSwitcher } from '@/shared/components/TenantSwitcher'

interface NavChild {
  label: string
  path: string
}

interface NavItemConfig {
  label: string
  path?: string
  icon: LucideIcon
  children?: NavChild[]
}

interface NavSectionConfig {
  title: string
  items: NavItemConfig[]
}

const NAV_SECTIONS: NavSectionConfig[] = [
  {
    title: 'Overview',
    items: [
      { label: 'Dashboard', path: '/', icon: LayoutGrid },
      { label: 'Access Policies', path: '/access-policies', icon: ArrowLeftRight },
    ],
  },
  {
    title: 'Resources',
    items: [
      { label: 'Client Workloads', path: '/client-workloads', icon: Monitor },
      { label: 'Trust Providers', path: '/trust-providers', icon: CheckCircle },
      { label: 'Access Conditions', path: '/access-conditions', icon: Clock },
      { label: 'Credential Providers', path: '/credential-providers', icon: Lock },
      { label: 'Server Workloads', path: '/server-workloads', icon: Server },
    ],
  },
  {
    title: 'Manage',
    items: [
      {
        label: 'Reporting',
        icon: BarChart3,
        children: [
          { label: 'Access Authorization', path: '/reporting/access-authorization' },
          { label: 'Audit Logs', path: '/reporting/audit' },
          { label: 'Workload Events', path: '/reporting/workload-events' },
          { label: 'Compliance', path: '/reporting/compliance' },
          { label: 'MCP Tracing', path: '/reporting/mcp-tracing' },
        ],
      },
      {
        label: 'Administration',
        icon: Users,
        children: [
          { label: 'Users', path: '/administration/users' },
          { label: 'Roles', path: '/administration/roles' },
          { label: 'Resource Sets', path: '/administration/resource-sets' },
          { label: 'Identity Providers', path: '/administration/identity-providers' },
          { label: 'Log Streams', path: '/administration/log-streams' },
          { label: 'Sign-On Policy', path: '/administration/sign-on-policy' },
          { label: 'Global Policy Compliance', path: '/administration/global-policy-compliance' },
        ],
      },
      {
        label: 'Edge Components',
        icon: Settings,
        children: [
          { label: 'Agent Controllers', path: '/edge-components/agent-controllers' },
          { label: 'Agent Proxies', path: '/edge-components/agent-proxies' },
          { label: 'Deploy Edge', path: '/edge-components/deploy' },
          { label: 'TLS Decrypt', path: '/edge-components/tls-decrypt' },
        ],
      },
    ],
  },
  {
    title: 'Support',
    items: [
      { label: 'Documentation', path: '/help', icon: BookOpen },
      { label: 'Help', path: '/help/contact', icon: HelpCircle },
    ],
  },
]

function NavItem({ item }: { item: NavItemConfig }) {
  const location = useLocation()
  const [isOpen, setIsOpen] = useState(() => {
    if (!item.children) return false
    return item.children.some((child) => location.pathname === child.path)
  })

  const hasChildren = !!item.children
  const Icon = item.icon

  if (!hasChildren) {
    return (
      <NavLink
        to={item.path!}
        end
        className={({ isActive }) =>
          `flex items-center gap-2.5 px-3 py-[7px] rounded-[6px] text-[13px] font-medium transition-all duration-200 select-none ${
            isActive
              ? 'bg-gray-100 text-gray-900'
              : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
          }`
        }
      >
        <Icon size={16} strokeWidth={1.5} className="shrink-0" />
        <span className="truncate">{item.label}</span>
      </NavLink>
    )
  }

  const isChildActive = item.children!.some((child) => location.pathname === child.path)

  return (
    <div className="flex flex-col w-full">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center justify-between px-3 py-[7px] rounded-[6px] text-[13px] font-medium transition-all duration-200 select-none w-full ${
          isChildActive
            ? 'text-gray-900'
            : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
        }`}
      >
        <div className="flex items-center gap-2.5">
          <Icon size={16} strokeWidth={1.5} className="shrink-0" />
          <span className="truncate">{item.label}</span>
        </div>
        <ChevronRight
          size={14}
          strokeWidth={2}
          className={`text-gray-500 transition-transform duration-200 ${isOpen ? 'rotate-90' : ''}`}
        />
      </button>

      <div
        className={`grid transition-[grid-template-rows,opacity] duration-300 ease-in-out ${
          isOpen ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'
        }`}
      >
        <div className="overflow-hidden min-h-0 relative flex flex-col gap-0.5 mt-0.5">
          <div
            className="absolute top-0 bottom-0 border-l border-gray-200"
            style={{ left: '19px' }}
          />
          {item.children!.map((child) => (
            <NavLink
              key={child.path}
              to={child.path}
              end
              className={({ isActive }) =>
                `flex items-center ml-[30px] mr-2 px-2.5 py-[6px] rounded-[6px] text-[12px] font-medium transition-all duration-200 select-none ${
                  isActive
                    ? 'bg-gray-100 text-gray-900'
                    : 'text-gray-500 hover:bg-gray-50 hover:text-gray-800'
                }`
              }
            >
              <span className="truncate">{child.label}</span>
            </NavLink>
          ))}
        </div>
      </div>
    </div>
  )
}

export function Sidebar() {
  const navigate = useNavigate()

  function getUserDisplayName(): string {
    try {
      const raw = sessionStorage.getItem('dipole_user')
      if (raw) {
        const user = JSON.parse(raw) as { name?: string; email?: string }
        return user.name || user.email || 'User'
      }
    } catch {
      /* ignore */
    }
    return 'User'
  }

  return (
    <aside className="fixed left-0 top-0 h-screen w-51 bg-white flex flex-col border-r border-gray-200">
      <div className="px-4 flex items-center gap-2 h-[60px] shrink-0 border-b border-gray-200 select-none pointer-events-none">
        <img src="/brand_black.svg" alt="Dipole" className="h-17 translate-x-1" />
      </div>

      <div className="px-3 mt-5 mb-3">
        <TenantSwitcher />
      </div>

      <div className="px-3 mb-3">
        <div className="px-3 mb-1.5 text-[10px] font-semibold text-gray-400 uppercase tracking-wider">
          Resource Set
        </div>
        <WorkspaceSwitcher />
      </div>

      <nav className="flex-1 px-3 py-1 overflow-y-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]">
        {NAV_SECTIONS.map((section, sectionIdx) => (
          <div key={section.title} className={sectionIdx > 0 ? 'mt-5' : 'mt-2'}>
            <div className="px-3 mb-1.5 text-[10px] font-semibold text-gray-400 uppercase tracking-wider">
              {section.title}
            </div>
            <div className="flex flex-col gap-0.5">
              {section.items.map((item) => (
                <NavItem key={item.label} item={item} />
              ))}
            </div>
          </div>
        ))}
      </nav>

      <div className="px-3 border-t border-gray-200 h-[52px] flex items-center relative group">
        <button className="flex items-center gap-2.5 w-full px-3 py-2 rounded-[6px] text-[13px] text-gray-600 hover:bg-gray-50 transition-colors">
          <User size={16} strokeWidth={1.5} className="shrink-0" />
          <span className="truncate">{getUserDisplayName()}</span>
        </button>

        <div className="absolute bottom-full left-0 right-0 px-3 pb-2 hidden group-hover:block z-50">
          <div className="bg-white rounded-lg shadow-xl border border-gray-200 py-1.5">
            <button
              onClick={() => navigate('/profile')}
              className="w-full text-left px-4 py-2 text-[13px] text-gray-600 hover:text-gray-900 hover:bg-gray-50 transition-colors"
            >
              Profile
            </button>
            <button
              onClick={() => {
                sessionStorage.removeItem('dipole_token')
                sessionStorage.removeItem('dipole_user')
                window.location.href = '/login'
              }}
              className="w-full text-left px-4 py-2 text-[13px] text-gray-600 hover:text-gray-900 hover:bg-gray-50 transition-colors"
            >
              Sign out
            </button>
          </div>
        </div>
      </div>
    </aside>
  )
}
