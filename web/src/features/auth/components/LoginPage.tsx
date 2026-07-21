export function LoginPage() {
  return (
    <div className="h-screen w-screen bg-[#28282B] flex flex-col items-center justify-center">
      <div className="flex items-center gap-6 mb-6 -mt-24 select-none pointer-events-none">
        <img src="/dipole_logo.svg" alt="Dipole" className="h-32 w-32" draggable={false} />
        <span className="text-white text-8xl tracking-tight" style={{ fontFamily: 'Tomorrow, sans-serif', fontWeight: 700 }}>Dipole</span>
      </div>

      <p className="text-gray-400 text-lg mb-12 select-none pointer-events-none">Identity & Access Management Fabric For Agents & Workloads</p>

      <button
        type="button"
        className="px-8 py-3 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
      >
        Login Using Email
      </button>
    </div>
  )
}
