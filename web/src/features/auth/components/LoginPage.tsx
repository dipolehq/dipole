import { CloudCanvas } from '@/features/auth/components/CloudCanvas'

export function LoginPage() {
  return (
    <div className="h-screen w-screen bg-[#050505] overflow-hidden">
      <CloudCanvas />

      {/* Centered content on top of canvas */}
      <div className="fixed inset-0 z-10 flex flex-col items-center justify-center pointer-events-none">
        <div className="flex items-center gap-6 mb-6 -mt-16 select-none">
          <img src="/dipole_logo.svg" alt="Dipole" className="h-32 w-32" draggable={false} />
          <span className="text-white text-8xl tracking-tight" style={{ fontFamily: 'Tomorrow, sans-serif', fontWeight: 700 }}>Dipole</span>
        </div>

        <p className="text-white text-lg mb-12 select-none">Identity & Access Management Fabric For Agents & Workloads</p>

        <button
          type="button"
          className="px-8 py-3 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors pointer-events-auto"
        >
          Login Using Email
        </button>
      </div>
    </div>
  )
}
