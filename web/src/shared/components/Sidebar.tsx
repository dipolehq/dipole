import { NavLink, useNavigate } from 'react-router-dom'
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
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { WorkspaceSwitcher } from '@/shared/components/WorkspaceSwitcher'

interface NavItemLink {
  label: string
  path: string
  icon: LucideIcon
}

interface NavSection {
  title: string
  items: NavItemLink[]
}

const NAV_SECTIONS: NavSection[] = [
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
      { label: 'Reporting', path: '/reporting', icon: BarChart3 },
      { label: 'Administration', path: '/administration', icon: Users },
      { label: 'Edge Components', path: '/edge-components', icon: Settings },
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

export function Sidebar() {
  const navigate = useNavigate()

  function getUserDisplayName(): string {
    try {
      const raw = sessionStorage.getItem('dipole_user')
      if (raw) {
        const user = JSON.parse(raw) as { name?: string; email?: string }
        return user.name || user.email || 'User'
      }
    } catch { /* ignore */ }
    return 'User'
  }

  return (
    <aside className="fixed left-0 top-0 h-screen w-51 bg-[#28282B] flex flex-col">
      <div className="px-4 flex items-center gap-2 h-[60px] shrink-0 border-b border-white/10 select-none pointer-events-none">
        <img src="/dipole_logo.svg" alt="Dipole" className="h-9 w-9" />
        <span className="text-white font-semibold text-xl tracking-tight">Dipole</span>
      </div>

      <div className="px-3 mt-5 mb-3">
        <div className="px-3 mb-1.5 text-[10px] font-semibold text-gray-500 uppercase tracking-wider">
          Resource Set
        </div>
        <WorkspaceSwitcher />
      </div>

      <nav className="flex-1 px-3 py-1 overflow-y-auto">
        {NAV_SECTIONS.map((section, sectionIdx) => (
          <div key={section.title} className={sectionIdx > 0 ? 'mt-6' : 'mt-2'}>
            <div className="px-3 mb-1.5 text-[10px] font-semibold text-gray-500 uppercase tracking-wider">
              {section.title}
            </div>
            <div className="space-y-0.5">
              {section.items.map((item) => {
                const Icon = item.icon
                return (
                  <NavLink
                    key={item.path + item.label}
                    to={item.path}
                    end
                    className={({ isActive }) =>
                      `flex items-center gap-3 px-3 py-2 rounded-md text-[13px] font-medium transition-colors ${
                        isActive
                          ? 'bg-white/5 text-white'
                          : 'text-gray-400 hover:bg-white/5 hover:text-gray-200'
                      }`
                    }
                  >
                    <Icon size={15} strokeWidth={1.75} className="shrink-0" />
                    <span>{item.label}</span>
                  </NavLink>
                )
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="px-3 border-t border-white/10 h-[45px] flex items-center relative group">
        <button className="flex items-center gap-2.5 w-full px-3 rounded text-sm text-gray-300">
          <User size={16} className="shrink-0" />
          <span>{getUserDisplayName()}</span>
        </button>

        <div className="absolute bottom-full left-0 right-0 px-3 pb-2 hidden group-hover:block z-50">
          <div className="bg-[#333336] rounded-lg shadow-lg py-2">
            <button onClick={() => navigate('/profile')} className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:text-white hover:bg-white/5 transition-colors">
              Profile
            </button>
            <button onClick={() => { sessionStorage.removeItem('dipole_token'); sessionStorage.removeItem('dipole_user'); window.location.href = '/login' }} className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:text-white hover:bg-white/5 transition-colors">
              Sign out
            </button>
          </div>
        </div>
      </div>
    </aside>
  )
}
