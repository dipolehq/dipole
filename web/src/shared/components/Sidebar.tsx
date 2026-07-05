import { NavLink, useNavigate } from 'react-router-dom'
import {
  LayoutGrid,
  Shield,
  Monitor,
  CheckCircle,
  Clock,
  Lock,
  Server,
  BarChart3,
  Settings,
  Users,
  HelpCircle,
  User,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

interface NavItemLink {
  label: string
  path: string
  icon: LucideIcon
  type?: undefined
}

interface NavItemDivider {
  type: 'divider'
}

type NavItem = NavItemLink | NavItemDivider

const NAV_ITEMS: NavItem[] = [
  { label: 'Dashboard', path: '/', icon: LayoutGrid },
  { label: 'Access Policies', path: '/access-policies', icon: Shield },
  { type: 'divider' },
  { label: 'Client Workloads', path: '/client-workloads', icon: Monitor },
  { label: 'Trust Providers', path: '/trust-providers', icon: CheckCircle },
  { label: 'Access Conditions', path: '/access-conditions', icon: Clock },
  { label: 'Credential Providers', path: '/credential-providers', icon: Lock },
  { label: 'Server Workloads', path: '/server-workloads', icon: Server },
  { type: 'divider' },
  { label: 'Reporting', path: '/reporting', icon: BarChart3 },
  { label: 'Edge Components', path: '/edge-components', icon: Settings },
  { label: 'Administration', path: '/administration', icon: Users },
  { label: 'Help', path: '/help', icon: HelpCircle },
]

export function Sidebar() {
  const navigate = useNavigate()

  return (
    <aside className="fixed left-0 top-0 h-screen w-51 bg-[#28282B] flex flex-col">
      <div className="px-4 pt-5 pb-6 flex items-center gap-2">
        <img src="/dipole_logo.svg" alt="Dipole" className="h-9 w-9" />
        <span className="text-white font-semibold text-xl tracking-tight">Dipole</span>
      </div>

      <nav className="flex-1 px-3 py-1 space-y-px overflow-y-auto">
        {NAV_ITEMS.map((item, idx) => {
          if (item.type === 'divider') {
            return <div key={idx} className="my-3 border-t border-white/10" />
          }
          const Icon = item.icon
          return (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md text-[13px] font-medium transition-colors ${
                  isActive
                    ? 'bg-white/10 text-white'
                    : 'text-gray-400 hover:bg-white/5 hover:text-gray-200'
                }`
              }
            >
              <Icon size={15} strokeWidth={1.75} className="shrink-0" />
              <span>{item.label}</span>
            </NavLink>
          )
        })}
      </nav>

      <div className="px-3 py-3 border-t border-white/10 relative group">
        <button className="flex items-center gap-2.5 w-full px-3 py-2 rounded text-sm text-gray-300 hover:bg-white/5 hover:text-white transition-colors">
          <User size={16} className="shrink-0" />
          <span>John Doe</span>
        </button>

        <div className="absolute bottom-full left-0 right-0 px-3 pb-1 hidden group-hover:block z-50">
          <div className="bg-[#333336] rounded-lg shadow-lg py-1">
            <button className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:text-white hover:bg-white/5 transition-colors">
              Sign out
            </button>
            <button onClick={() => navigate('/profile')} className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:text-white hover:bg-white/5 transition-colors">
              Profile
            </button>
          </div>
        </div>
      </div>
    </aside>
  )
}
