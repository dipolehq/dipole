import { Link } from 'react-router-dom'
import { ExternalLink, Play, Sparkles, MonitorPlay, ArrowRight } from 'lucide-react'

export function GettingStartedPanel() {
  return (
    <div className="flex flex-col h-full space-y-6">
      <div className="bg-orange-50 rounded-lg p-4 grid grid-cols-3 gap-5 border border-orange-200">
        {/* Introduction */}
        <div className="flex flex-col">
          <h3 className="text-sm font-semibold text-gray-900 mb-2">Introduction to Dipole</h3>
          <p className="text-xs text-gray-600 leading-relaxed flex-1">
            To gain a better understanding of Dipole and Workload Identity and Access Management (IAM), please watch the An Introduction To Dipole and IAM video.
          </p>
          <a href="#" className="inline-flex items-center gap-1.5 text-xs text-orange-500 hover:text-orange-600 font-medium mt-4">
            <Play size={12} />
            How Does Dipole Work?
          </a>
        </div>

        {/* QuickStart */}
        <div className="flex flex-col">
          <h3 className="text-sm font-semibold text-gray-900 mb-2">QuickStart</h3>
          <p className="text-xs text-gray-600 leading-relaxed flex-1">
            Follow the QuickStart guide to set up a sandbox workloads and explore Dipole's capabilities. Explore our detailed{' '}
            <a href="#" className="text-orange-500 hover:text-orange-600 font-semibold">Documentation</a>{' '}
            to dive deeper into Dipole's features and configurations.
          </p>
          <a href="#" className="inline-flex items-center gap-1.5 text-xs text-orange-500 hover:text-orange-600 font-medium mt-4">
            <Sparkles size={12} />
            QuickStart
          </a>
        </div>

        {/* Demo */}
        <div className="flex flex-col">
          <h3 className="text-sm font-semibold text-gray-900 mb-2">Demo</h3>
          <p className="text-xs text-gray-600 leading-relaxed flex-1">
            We're happy to provide a one-on-one demo that will help you better understand Dipole as well answer any questions you may have.
          </p>
          <a href="#" className="inline-flex items-center gap-1.5 text-xs text-orange-500 hover:text-orange-600 font-medium mt-4">
            <MonitorPlay size={12} />
            Request a Demo
          </a>
        </div>
      </div>

      {/* Bottom cards — 2x2 grid */}
      <div className="grid grid-cols-2 gap-5 mt-2">
        {/* Deploy Dipole Edge */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="text-base font-semibold text-gray-900 mb-3">Deploy Dipole Edge</h3>
          <p className="text-xs text-gray-600 mb-2 leading-relaxed">
            Dipole Edge is a set of helper components which operate as the interface between your workload traffic and Dipole's Access Policies.
          </p>
          <p className="text-xs text-gray-600 mb-4 leading-relaxed">
            Deploying Dipole Edge to your live environment allows you to capture the interactions between your workloads as Events in the Workload Event Log.
          </p>
          <Link
            to="/edge-components"
            className="inline-flex items-center justify-center gap-1.5 w-[170px] h-[36px] bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            Deploy Dipole Edge
            <ArrowRight size={11} />
          </Link>
        </div>

        {/* Access Policies */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="text-base font-semibold text-gray-900 mb-3">Access Policies</h3>
          <p className="text-xs text-gray-600 mb-2 leading-relaxed">
            Access Policies specify the conditions that must be satisfied for Dipole to grant Client Workloads access to Server Workloads.
          </p>
          <p className="text-xs text-gray-600 mb-4 leading-relaxed">
            To see how this works without installing into your live environment, we need to set up some sample Workloads and a sample Access Policy.
          </p>
          <Link
            to="/access-policies/new"
            className="inline-flex items-center justify-center gap-1.5 w-[170px] h-[36px] bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            Add an Access Policy
            <ArrowRight size={11} />
          </Link>
        </div>

        {/* Reporting & Logs */}
        <div className="border border-border rounded-lg p-4 flex flex-col">
          <h3 className="text-base font-semibold text-gray-900 mb-3">Reporting & Logs</h3>
          <p className="text-xs text-gray-600 mb-2 leading-relaxed">
            Dipole captures detailed logs for every access decision and workload interaction. View Access Authorization Events, Workload Events, and Audit Logs to understand what's happening across your environment.
          </p>
          <p className="text-xs text-gray-600 leading-relaxed flex-1">
            Use these logs to troubleshoot denied requests, verify policy evaluations, and maintain a full audit trail of all workload-to-workload communication.
          </p>
          <Link
            to="/reporting"
            className="inline-flex items-center justify-center gap-1.5 w-[170px] h-[36px] bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors mt-4"
          >
            Explore Reporting
            <ArrowRight size={11} />
          </Link>
        </div>

        {/* Dashboard & Metrics */}
        <div className="border border-border rounded-lg p-4 flex flex-col">
          <h3 className="text-base font-semibold text-gray-900 mb-3">Dashboard & Metrics</h3>
          <p className="text-xs text-gray-600 mb-2 leading-relaxed">
            The Dashboard provides a real-time overview of your Dipole environment — including total access decisions, credential injections, active policies, and workload health at a glance.
          </p>
          <p className="text-xs text-gray-600 leading-relaxed flex-1">
            Monitor trends over time, spot anomalies in access patterns, and track the health of your Edge Components from a single unified view.
          </p>
          <Link
            to="/"
            className="inline-flex items-center justify-center gap-1.5 w-[170px] h-[36px] bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors mt-4"
          >
            View Dashboard
            <ArrowRight size={11} />
          </Link>
        </div>
      </div>

      {/* Quick links footer */}
      <div className="flex items-center justify-end gap-6 pt-4 mt-auto">
        <a href="#" className="inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-gray-700 font-medium">
          <ExternalLink size={11} />
          Support
        </a>
        <a href="#" className="inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-gray-700 font-medium">
          <ExternalLink size={11} />
          Status
        </a>
        <a href="#" className="inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-gray-700 font-medium">
          <ExternalLink size={11} />
          Documentation
        </a>
      </div>
    </div>
  )
}
