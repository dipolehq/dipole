import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Copy, ArrowRight } from 'lucide-react'
import { Dropdown } from '@/shared/components/Dropdown'
import { useAgentControllers, useRegenerateDeviceCode } from '@/features/edge-components/hooks/useAgentControllers'

type Environment = 'kubernetes' | 'ubuntu' | 'redhat' | 'windows'

const ENVIRONMENTS: { value: Environment; label: string; subtitle: string; icon: string }[] = [
  { value: 'kubernetes', label: 'kubernetes', subtitle: 'Kubernetes', icon: '⎈' },
  { value: 'ubuntu', label: 'Ubuntu', subtitle: 'Ubuntu Linux', icon: '⊙' },
  { value: 'redhat', label: 'Red Hat', subtitle: 'Red Hat Enterprise Linux', icon: '🎩' },
  { value: 'windows', label: 'Microsoft', subtitle: 'Windows', icon: '⊞' },
]

// TODO: Wire to real tenant ID from auth context
const TENANT_ID = '<YOUR_TENANT_ID>'

export function DeployEdgePanel() {
  const [selectedEnv, setSelectedEnv] = useState<Environment>('kubernetes')
  const [selectedController, setSelectedController] = useState('')
  const [deviceCode, setDeviceCode] = useState('')

  const { data: controllersData } = useAgentControllers(1, 100)
  const controllers = controllersData?.items ?? []
  const regenerateCode = useRegenerateDeviceCode()

  const AGENT_CONTROLLER_OPTIONS = [
    { value: '', label: 'Select Existing' },
    ...controllers.map((ac) => ({ value: ac.id, label: ac.name })),
  ]

  async function handleGenerateCode() {
    if (!selectedController) return
    try {
      const code = await regenerateCode.mutateAsync(selectedController)
      setDeviceCode(code)
    } catch (err) {
      console.error('Failed to generate device code', err)
    }
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text)
  }

  const helmRepoCmd = 'helm repo add dipole https://helm.dipolehq.com'
  const helmInstallCmd = deviceCode
    ? `helm install dipole dipole/dipole -n dipole --create-namespace --set tenant=${TENANT_ID},agentController.deviceCode=${deviceCode}`
    : `helm install dipole dipole/dipole -n dipole --create-namespace --set tenant=${TENANT_ID}`

  return (
    <div className="flex h-full">
      {/* Left sidebar — Environment selector */}
      <div className="w-[200px] border-r border-border p-4 space-y-2 shrink-0">
        <p className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-3">Environments</p>
        {ENVIRONMENTS.map((env) => (
          <button
            key={env.value}
            type="button"
            onClick={() => setSelectedEnv(env.value)}
            className={`w-full text-left px-3 py-4 rounded-lg border transition-colors ${
              selectedEnv === env.value
                ? 'border-orange-500 bg-orange-50'
                : 'border-border hover:border-gray-300 hover:bg-gray-50'
            }`}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <EnvironmentLogo env={env.value} />
                <span className={`text-sm font-semibold ${selectedEnv === env.value ? 'text-gray-900' : 'text-gray-700'}`}>{env.label}</span>
              </div>
              <ArrowRight size={12} className={selectedEnv === env.value ? 'text-orange-500' : 'text-gray-400'} />
            </div>
            <p className="text-[11px] text-gray-500 mt-1 ml-7">{env.subtitle}</p>
          </button>
        ))}
      </div>

      {/* Main content */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {selectedEnv === 'kubernetes' ? (
          <KubernetesDeployment
            selectedController={selectedController}
            onControllerChange={setSelectedController}
            deviceCode={deviceCode}
            onGenerateCode={handleGenerateCode}
            isGenerating={regenerateCode.isPending}
            helmRepoCmd={helmRepoCmd}
            helmInstallCmd={helmInstallCmd}
            onCopy={copyToClipboard}
            controllerOptions={AGENT_CONTROLLER_OPTIONS}
          />
        ) : selectedEnv === 'ubuntu' ? (
          <UbuntuDeployment
            selectedController={selectedController}
            onControllerChange={setSelectedController}
            deviceCode={deviceCode}
            onGenerateCode={handleGenerateCode}
            isGenerating={regenerateCode.isPending}
            onCopy={copyToClipboard}
            controllerOptions={AGENT_CONTROLLER_OPTIONS}
          />
        ) : selectedEnv === 'redhat' ? (
          <RedHatDeployment
            selectedController={selectedController}
            onControllerChange={setSelectedController}
            deviceCode={deviceCode}
            onGenerateCode={handleGenerateCode}
            isGenerating={regenerateCode.isPending}
            onCopy={copyToClipboard}
            controllerOptions={AGENT_CONTROLLER_OPTIONS}
          />
        ) : selectedEnv === 'windows' ? (
          <WindowsDeployment
            selectedController={selectedController}
            onControllerChange={setSelectedController}
            deviceCode={deviceCode}
            onGenerateCode={handleGenerateCode}
            isGenerating={regenerateCode.isPending}
            onCopy={copyToClipboard}
            controllerOptions={AGENT_CONTROLLER_OPTIONS}
          />
        ) : (
          <div className="flex items-center justify-center h-64">
            <p className="text-sm text-gray-400">{ENVIRONMENTS.find((e) => e.value === selectedEnv)?.subtitle} deployment — coming soon</p>
          </div>
        )}
      </div>
    </div>
  )
}

function EnvironmentLogo({ env }: { env: Environment }) {
  switch (env) {
    case 'kubernetes':
      return (
        <svg className="w-5 h-5" viewBox="0 0 32 32" fill="none">
          <path d="M15.9 2a1.5 1.5 0 00-.7.2l-11 6.3a1.5 1.5 0 00-.7 1.3v12.4a1.5 1.5 0 00.7 1.3l11 6.3a1.5 1.5 0 001.4 0l11-6.3a1.5 1.5 0 00.7-1.3V9.8a1.5 1.5 0 00-.7-1.3l-11-6.3a1.5 1.5 0 00-.7-.2z" fill="#326CE5"/>
          <path d="M16 8.5l.1.7.3.6-.1.7v.5l.1.4.1.3-.1.5-.1.3.1.5.2.4v.4l-.1.3-.2.5.1.4.2.3v.7l-.3.3-.4.2-.3.3-.2.3-.4.2-.4.1-.4.2-.2.2-.5.1-.4.2h-.4l-.3-.1-.5.1-.4-.1-.3-.1-.4.1-.4-.2-.3-.2-.3-.2-.3-.3-.3-.3-.2-.4.1-.4v-.5l-.2-.4.1-.5v-.4l-.2-.4.1-.5.1-.3-.1-.5.1-.4.1-.3-.1-.5v-.4l.2-.5.1-.6.3-.6.1-.7z" fill="#fff"/>
        </svg>
      )
    case 'ubuntu':
      return (
        <svg className="w-5 h-5" viewBox="0 0 32 32" fill="none">
          <circle cx="16" cy="16" r="14" fill="#E95420"/>
          <circle cx="9.5" cy="16" r="2.5" fill="#fff"/>
          <circle cx="19.8" cy="10.2" r="2.5" fill="#fff"/>
          <circle cx="19.8" cy="21.8" r="2.5" fill="#fff"/>
          <path d="M11.5 14.5l6.5-3.5M11.5 17.5l6.5 3.5" stroke="#fff" strokeWidth="1.2"/>
        </svg>
      )
    case 'redhat':
      return (
        <svg className="w-5 h-5" viewBox="0 0 32 32" fill="none">
          <path d="M16 4C9.4 4 4 9.4 4 16s5.4 12 12 12 12-5.4 12-12S22.6 4 16 4z" fill="#EE0000"/>
          <path d="M20.5 12c-1.5-1-3.5-1.2-5-.5-1.8.8-2.5 2.5-2 4.2.4 1.5 2 2.8 3.5 3 2 .3 3.8-.5 4.5-2 .5-1 .3-2.2-.3-3-.5-.7-1.5-1.2-2.2-1.2" fill="#fff"/>
          <ellipse cx="13" cy="10" rx="3" ry="2.5" fill="#EE0000"/>
          <path d="M11 9.5c.5-1 2-1.5 3.5-1s2 1.8 1.5 2.5" stroke="#fff" strokeWidth=".5"/>
        </svg>
      )
    case 'windows':
      return (
        <svg className="w-5 h-5" viewBox="0 0 32 32" fill="none">
          <path d="M3 7.5l11.5-1.6v11H3V7.5z" fill="#F25022"/>
          <path d="M15.5 5.7L29 3.5v13.4H15.5V5.7z" fill="#7FBA00"/>
          <path d="M3 17.9h11.5v11L3 27.3v-9.4z" fill="#00A4EF"/>
          <path d="M15.5 17.9H29V31l-13.5-1.9V17.9z" fill="#FFB900"/>
        </svg>
      )
  }
}

function KubernetesDeployment({
  selectedController,
  onControllerChange,
  deviceCode,
  onGenerateCode,
  isGenerating,
  helmRepoCmd,
  helmInstallCmd,
  onCopy,
  controllerOptions,
}: {
  selectedController: string
  onControllerChange: (v: string) => void
  deviceCode: string
  onGenerateCode: () => void
  isGenerating: boolean
  helmRepoCmd: string
  helmInstallCmd: string
  onCopy: (text: string) => void
  controllerOptions: { value: string; label: string }[]
}) {
  const controllerSelected = selectedController !== ''

  return (
    <>
      {/* Prerequisites */}
      <div className="border border-border rounded-lg p-5">
        <h3 className="text-sm font-semibold text-orange-600 mb-2">Prerequisites</h3>
        <p className="text-xs text-gray-600 mb-3">
          You must have the following to deploy Dipole Edge Components on Kubernetes:
        </p>
        <ul className="list-disc list-inside text-xs text-gray-600 space-y-1">
          <li>A running Kubernetes cluster.</li>
          <li>Helm installed on your system.</li>
        </ul>
      </div>

      {/* Step 1: Prepare Edge Components */}
      <div className="border border-border rounded-lg p-5">
        <div className="flex items-center gap-2 mb-3">
          <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">1</span>
          <h3 className="text-sm font-semibold text-gray-900">Prepare Edge Components</h3>
        </div>

        <p className="text-xs text-gray-600 mb-1">
          1. Click <span className="font-semibold">New Agent Controller</span> or select an existing Agent Controller.
        </p>
        <p className="text-xs text-gray-500 mb-4">
          See <Link to="/edge-components/agent_controllers/new" className="text-orange-500 hover:text-orange-600">Create an Agent Controller</Link>.
        </p>

        <div className="flex items-center gap-3 mb-5">
          <Link
            to="/edge-components/agent_controllers/new"
            className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            <Plus size={12} />
            New Agent Controller
          </Link>
          <span className="text-xs text-gray-500 font-medium">OR</span>
          <div className="w-52">
            <Dropdown
              options={controllerOptions}
              value={selectedController}
              onChange={onControllerChange}
              placeholder="Select Existing"
            />
          </div>
        </div>

        {controllerSelected && (
          <>
            <p className="text-xs text-gray-600 mb-1">
              2. Click <span className="font-semibold">Create Device Code</span>.
            </p>
            <p className="text-xs text-gray-500 mb-3">
              The Device code authorizes the Agent Controller in your Kubernetes cluster to access Dipole Cloud and is valid for 15 minutes.
            </p>

            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={onGenerateCode}
                disabled={isGenerating}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-60"
              >
                {isGenerating ? 'Generating\u2026' : 'Create Device Code'}
              </button>
              {deviceCode && (
                <div className="px-4 py-2 border border-border rounded-md text-sm font-mono tracking-widest text-gray-700 bg-gray-50">
                  {deviceCode}
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {/* Message when no controller selected */}
      {!controllerSelected && (
        <p className="text-xs text-orange-500 text-center py-4">
          To continue, complete the preceding step: Add or create an Agent Controller.
        </p>
      )}

      {/* Steps 2 & 3 only visible after controller is selected */}
      {controllerSelected && (
        <>
          {/* Step 2: Install Helm Chart */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">2</span>
              <h3 className="text-sm font-semibold text-gray-900">Install Dipole Edge Helm Chart</h3>
            </div>

            <p className="text-xs text-gray-600 mb-2">
              1. From a terminal within your Kubernetes context, add the Dipole Helm chart repository:
            </p>
            <CodeBlock text={helmRepoCmd} onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              2. Install the Dipole Helm chart:
            </p>
            <CodeBlock text={helmInstallCmd} onCopy={onCopy} />
          </div>

          {/* Step 3: Annotate Client Workloads */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">3</span>
              <h3 className="text-sm font-semibold text-gray-900">Annotate Client Workloads</h3>
            </div>

            <p className="text-xs text-gray-600 mb-3">
              To enable Dipole Edge to manage Client Workloads, annotate them appropriately in your YAML configuration, Helm chart, or ArgoCD:
            </p>
            <CodeBlock text='dipolehq.com/agent-inject: "enabled"' onCopy={onCopy} />
            <p className="text-xs text-gray-500 mt-3">
              This annotation ensures that the Dipole Edge Components can intercept network requests from your client workloads.
            </p>
          </div>

          {/* Additional Configurations */}
          <div className="border border-border rounded-lg p-5 bg-gray-50">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Additional Configurations</h3>
            <p className="text-xs text-gray-600">
              For further configuration options, refer to the official documentation:{' '}
              <a href="#" className="text-orange-500 hover:text-orange-600">Deploying Kubernetes Configurations</a>.
            </p>
          </div>
        </>
      )}
    </>
  )
}

function UbuntuDeployment({
  selectedController,
  onControllerChange,
  deviceCode,
  onGenerateCode,
  isGenerating,
  onCopy,
  controllerOptions,
}: {
  selectedController: string
  onControllerChange: (v: string) => void
  deviceCode: string
  onGenerateCode: () => void
  isGenerating: boolean
  onCopy: (text: string) => void
  controllerOptions: { value: string; label: string }[]
}) {
  const controllerSelected = selectedController !== ''

  return (
    <>
      {/* Prerequisites */}
      <div className="border border-border rounded-lg p-5">
        <h3 className="text-sm font-semibold text-orange-600 mb-2">Prerequisites</h3>
        <p className="text-xs text-gray-600 mb-4">
          Use the following table to make sure that the operating system and platform you're deploying to your virtual machine (VM) is supported:
        </p>

        <div className="border border-border rounded-md overflow-hidden mb-4">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-50 border-b border-border">
                <th className="px-4 py-2 text-left font-semibold text-gray-600">Operating system</th>
                <th className="px-4 py-2 text-left font-semibold text-gray-600">Edge Component versions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              <tr>
                <td className="px-4 py-2 text-gray-700">Ubuntu 20.04 LTS</td>
                <td className="px-4 py-2 text-gray-700">
                  Agent Controller 1.32.3502<br />Agent Proxy 1.32.4999
                </td>
              </tr>
              <tr>
                <td className="px-4 py-2 text-gray-700">Ubuntu 22.04 LTS</td>
                <td className="px-4 py-2 text-gray-700">
                  Agent Controller 1.32.3502<br />Agent Proxy 1.32.4999
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <p className="text-xs text-gray-600 mb-2">
          Dipole recommends deploying Agent Controller on a standalone virtual machine and not collocating it with Agent Proxy. See{' '}
          <a href="#" className="text-orange-500 hover:text-orange-600">About Colocating Edge Components</a>.
        </p>
        <p className="text-xs text-gray-600 mb-2">
          Make sure the VM is running and accessible via SSH.
        </p>
        <CodeBlock text="ssh -i <PATH TO YOUR SSH KEY> <YOUR USERNAME ON REMOTE HOST>@<HOSTNAME>" onCopy={onCopy} />
      </div>

      {/* Step 1: Prepare Edge Components */}
      <div className="border border-border rounded-lg p-5">
        <div className="flex items-center gap-2 mb-3">
          <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">1</span>
          <h3 className="text-sm font-semibold text-gray-900">Prepare Edge Components</h3>
        </div>

        <p className="text-xs text-gray-600 mb-1">
          1. Click <span className="font-semibold">New Agent Controller</span> or select an existing Agent Controller.
        </p>
        <p className="text-xs text-gray-500 mb-4">
          See <Link to="/edge-components/agent_controllers/new" className="text-orange-500 hover:text-orange-600">Create an Agent Controller</Link>.
        </p>

        <div className="flex items-center gap-3 mb-5">
          <Link
            to="/edge-components/agent_controllers/new"
            className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            <Plus size={12} />
            New Agent Controller
          </Link>
          <span className="text-xs text-gray-500 font-medium">OR</span>
          <div className="w-52">
            <Dropdown
              options={controllerOptions}
              value={selectedController}
              onChange={onControllerChange}
              placeholder="Select Existing"
            />
          </div>
        </div>

        {controllerSelected && (
          <>
            <p className="text-xs text-gray-600 mb-1">
              2. Click <span className="font-semibold">Create Device Code</span>.
            </p>
            <p className="text-xs text-gray-500 mb-3">
              The Device code authorizes the Agent Controller in your VM to access Dipole Cloud and is valid for 15 minutes.
            </p>

            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={onGenerateCode}
                disabled={isGenerating}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-60"
              >
                {isGenerating ? 'Generating\u2026' : 'Create Device Code'}
              </button>
              {deviceCode && (
                <div className="px-4 py-2 border border-border rounded-md text-sm font-mono tracking-widest text-gray-700 bg-gray-50">
                  {deviceCode}
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {/* Message when no controller selected */}
      {!controllerSelected && (
        <p className="text-xs text-orange-500 text-center py-4">
          To continue, complete the preceding step: Add or create an Agent Controller.
        </p>
      )}

      {/* Steps 2 & 3 only visible after controller is selected */}
      {controllerSelected && (
        <>
          {/* Step 2: Install the Agent Controller */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">2</span>
              <h3 className="text-sm font-semibold text-gray-900">Install the Agent Controller</h3>
            </div>

            <p className="text-xs text-gray-600 mb-2">
              1. Run the following command to download the latest version of Agent Controller:
            </p>
            <CodeBlock text="wget https://releases.dipolehq.com/agent_controller/1.32.3502/linux/amd64/dipole_agent_controller_linux_amd64_1.32.3502.tar.gz" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              2. Extract the Archive:
            </p>
            <CodeBlock text="tar xf dipole_agent_controller_linux_amd64_1.32.3502.tar.gz" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              3. Navigate to the unpacked directory:
            </p>
            <CodeBlock text="cd dipole_agent_controller_linux_amd64_1.32.3502" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              4. Run the installer to enable Device Code based registration:
            </p>
            <CodeBlock text={`sudo DIPOLE_TENANT_ID=${TENANT_ID} ./install`} onCopy={onCopy} />
          </div>

          {/* Step 3: Install the Agent Proxy */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">3</span>
              <h3 className="text-sm font-semibold text-gray-900">Install the Agent Proxy</h3>
            </div>

            <p className="text-xs text-gray-600 mb-2">
              1. On the system where you would like to install the Agent Proxy, run the following command to download the latest version:
            </p>
            <CodeBlock text="wget https://releases.dipolehq.com/agent_proxy/1.32.4999/linux/amd64/dipole_agent_proxy_linux_amd64_1.32.4999.tar.gz" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              2. Extract the Archive:
            </p>
            <CodeBlock text="tar xf dipole_agent_proxy_linux_amd64_1.32.4999.tar.gz" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              3. Navigate to the unpacked directory:
            </p>
            <CodeBlock text="cd dipole_agent_proxy_linux_amd64_1.32.4999" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              4. Run the installer:
            </p>
            <CodeBlock text="sudo DIPOLE_AGENT_CONTROLLER=http://<AGENT CONTROLLER HOST>:5000 ./install" onCopy={onCopy} />
          </div>

          {/* Additional Configurations */}
          <div className="border border-border rounded-lg p-5 bg-gray-50">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Additional Configurations</h3>
            <p className="text-xs text-gray-600 mb-1">
              For further customization and advanced configurations, see{' '}
              <a href="#" className="text-orange-500 hover:text-orange-600">Dipole Virtual Machine Deployment Guide</a>.
            </p>
            <p className="text-xs text-gray-600">
              This guide always uses the latest version of Agent Controller and Proxy. For other versions, see{' '}
              <a href="#" className="text-orange-500 hover:text-orange-600">Dipole Releases</a>.
            </p>
          </div>
        </>
      )}
    </>
  )
}

function RedHatDeployment({
  selectedController,
  onControllerChange,
  deviceCode,
  onGenerateCode,
  isGenerating,
  onCopy,
  controllerOptions,
}: {
  selectedController: string
  onControllerChange: (v: string) => void
  deviceCode: string
  onGenerateCode: () => void
  isGenerating: boolean
  onCopy: (text: string) => void
  controllerOptions: { value: string; label: string }[]
}) {
  const controllerSelected = selectedController !== ''

  return (
    <>
      {/* Prerequisites */}
      <div className="border border-border rounded-lg p-5">
        <h3 className="text-sm font-semibold text-orange-600 mb-2">Prerequisites</h3>
        <p className="text-xs text-gray-600 mb-4">
          Use the following table to make sure that the operating system and platform you're deploying to your virtual machine (VM) is supported:
        </p>

        <div className="border border-border rounded-md overflow-hidden mb-4">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-50 border-b border-border">
                <th className="px-4 py-2 text-left font-semibold text-gray-600">Operating system</th>
                <th className="px-4 py-2 text-left font-semibold text-gray-600">Edge Component versions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              <tr>
                <td className="px-4 py-2 text-gray-700">Red Hat 8.9</td>
                <td className="px-4 py-2 text-gray-700">
                  Agent Controller 1.32.3502<br />Agent Proxy 1.32.4999
                </td>
              </tr>
              <tr>
                <td className="px-4 py-2 text-gray-700">Red Hat 9.3</td>
                <td className="px-4 py-2 text-gray-700">
                  Agent Controller 1.32.3502<br />Agent Proxy 1.32.4999
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <p className="text-xs text-gray-600 mb-2">
          Dipole recommends deploying Agent Controller on a standalone virtual machine and not collocating it with Agent Proxy. See{' '}
          <a href="#" className="text-orange-500 hover:text-orange-600">About Colocating Edge Components</a>.
        </p>
        <p className="text-xs text-gray-600 mb-2">
          Make sure the VM is running and accessible via SSH.
        </p>
        <CodeBlock text="ssh -i <PATH TO YOUR SSH KEY> <YOUR USERNAME ON REMOTE HOST>@<HOSTNAME>" onCopy={onCopy} />
      </div>

      {/* Step 1: Prepare Edge Components */}
      <div className="border border-border rounded-lg p-5">
        <div className="flex items-center gap-2 mb-3">
          <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">1</span>
          <h3 className="text-sm font-semibold text-gray-900">Prepare Edge Components</h3>
        </div>

        <p className="text-xs text-gray-600 mb-1">
          1. Click <span className="font-semibold">New Agent Controller</span> or select an existing Agent Controller.
        </p>
        <p className="text-xs text-gray-500 mb-4">
          See <Link to="/edge-components/agent_controllers/new" className="text-orange-500 hover:text-orange-600">Create an Agent Controller</Link>.
        </p>

        <div className="flex items-center gap-3 mb-5">
          <Link
            to="/edge-components/agent_controllers/new"
            className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            <Plus size={12} />
            New Agent Controller
          </Link>
          <span className="text-xs text-gray-500 font-medium">OR</span>
          <div className="w-52">
            <Dropdown
              options={controllerOptions}
              value={selectedController}
              onChange={onControllerChange}
              placeholder="Select Existing"
            />
          </div>
        </div>

        {controllerSelected && (
          <>
            <p className="text-xs text-gray-600 mb-1">
              2. Click <span className="font-semibold">Create Device Code</span>.
            </p>
            <p className="text-xs text-gray-500 mb-3">
              The Device code authorizes the Agent Controller in your VM to access Dipole Cloud and is valid for 15 minutes.
            </p>

            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={onGenerateCode}
                disabled={isGenerating}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-60"
              >
                {isGenerating ? 'Generating\u2026' : 'Create Device Code'}
              </button>
              {deviceCode && (
                <div className="px-4 py-2 border border-border rounded-md text-sm font-mono tracking-widest text-gray-700 bg-gray-50">
                  {deviceCode}
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {/* Message when no controller selected */}
      {!controllerSelected && (
        <p className="text-xs text-orange-500 text-center py-4">
          To continue, complete the preceding step: Add or create an Agent Controller.
        </p>
      )}

      {/* Steps 2 & 3 only visible after controller is selected */}
      {controllerSelected && (
        <>
          {/* Step 2: Install the Agent Controller */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">2</span>
              <h3 className="text-sm font-semibold text-gray-900">Install the Agent Controller</h3>
            </div>

            <p className="text-xs text-gray-600 mb-2">
              1. To prevent errors during installation, install the necessary dependencies:
            </p>
            <CodeBlock text="sudo yum install -y libicu wget" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              2. On the system where you would like to install the Agent Controller, run the following command to download the latest version:
            </p>
            <CodeBlock text="wget https://releases.dipolehq.com/agent_controller/1.32.3502/linux/amd64/dipole_agent_controller_linux_amd64_1.32.3502.tar.gz" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              3. Extract the Archive:
            </p>
            <CodeBlock text="tar xf dipole_agent_controller_linux_amd64_1.32.3502.tar.gz" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              4. Navigate to the unpacked directory:
            </p>
            <CodeBlock text="cd dipole_agent_controller_linux_amd64_1.32.3502" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              5. Run the installer to enable Device Code based registration:
            </p>
            <CodeBlock text={`sudo DIPOLE_TENANT_ID=${TENANT_ID} ./install`} onCopy={onCopy} />
          </div>

          {/* Step 3: Install the Agent Proxy */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">3</span>
              <h3 className="text-sm font-semibold text-gray-900">Install the Agent Proxy</h3>
            </div>

            <p className="text-xs text-gray-600 mb-2">
              1. To prevent errors during installation, install the necessary dependencies:
            </p>
            <CodeBlock text="sudo yum install -y wget iptables" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              2. On the system where you would like to install the Agent Proxy, run the following command to download the latest version:
            </p>
            <CodeBlock text="wget https://releases.dipolehq.com/agent_proxy/1.32.4999/linux/amd64/dipole_agent_proxy_linux_amd64_1.32.4999.tar.gz" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              3. Extract the Archive:
            </p>
            <CodeBlock text="tar xf dipole_agent_proxy_linux_amd64_1.32.4999.tar.gz" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              4. Navigate to the unpacked directory:
            </p>
            <CodeBlock text="cd dipole_agent_proxy_linux_amd64_1.32.4999" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              5. Run the installer:
            </p>
            <CodeBlock text="sudo DIPOLE_AGENT_CONTROLLER=http://<AGENT CONTROLLER HOST>:5000 ./install" onCopy={onCopy} />
          </div>

          {/* Additional Configurations */}
          <div className="border border-border rounded-lg p-5 bg-gray-50">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Additional Configurations</h3>
            <p className="text-xs text-gray-600 mb-1">
              For further customization and advanced configurations, see{' '}
              <a href="#" className="text-orange-500 hover:text-orange-600">Dipole Virtual Machine Deployment Guide</a>.
            </p>
            <p className="text-xs text-gray-600 mb-1">
              For <span className="font-semibold">SELinux configurations</span>, see{' '}
              <a href="#" className="text-orange-500 hover:text-orange-600">Dipole SELinux Deployment Guide</a>.
            </p>
            <p className="text-xs text-gray-600">
              This guide always uses the latest version of Agent Controller and Proxy. For other versions, see{' '}
              <a href="#" className="text-orange-500 hover:text-orange-600">Dipole Releases</a>.
            </p>
          </div>
        </>
      )}
    </>
  )
}

function WindowsDeployment({
  selectedController,
  onControllerChange,
  deviceCode,
  onGenerateCode,
  isGenerating,
  onCopy,
  controllerOptions,
}: {
  selectedController: string
  onControllerChange: (v: string) => void
  deviceCode: string
  onGenerateCode: () => void
  isGenerating: boolean
  onCopy: (text: string) => void
  controllerOptions: { value: string; label: string }[]
}) {
  const controllerSelected = selectedController !== ''

  return (
    <>
      {/* Prerequisites */}
      <div className="border border-border rounded-lg p-5">
        <h3 className="text-sm font-semibold text-orange-600 mb-2">Prerequisites</h3>
        <p className="text-xs text-gray-600 mb-4">
          Use the following table to make sure that the operating system and platform you're deploying to your virtual machine (VM) is supported:
        </p>

        <div className="border border-border rounded-md overflow-hidden mb-4">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-50 border-b border-border">
                <th className="px-4 py-2 text-left font-semibold text-gray-600">Operating system</th>
                <th className="px-4 py-2 text-left font-semibold text-gray-600">Edge Component versions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              <tr>
                <td className="px-4 py-2 text-gray-700">Windows Server 2019</td>
                <td className="px-4 py-2 text-gray-700">
                  Agent Controller 1.32.3502<br />Agent Proxy 1.32.4999
                </td>
              </tr>
              <tr>
                <td className="px-4 py-2 text-gray-700">Windows Server 2022</td>
                <td className="px-4 py-2 text-gray-700">
                  Agent Controller 1.32.3502<br />Agent Proxy 1.32.4999
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <p className="text-xs text-gray-600 mb-2">
          Dipole recommends deploying Agent Controller on a standalone virtual machine and not collocating it with Agent Proxy. See{' '}
          <a href="#" className="text-orange-500 hover:text-orange-600">About Colocating Edge Components</a>.
        </p>
        <p className="text-xs text-gray-600 mb-2">
          Make sure the VM is running and accessible via RDP or SSH.
        </p>
        <CodeBlock text="ssh -i <PATH TO YOUR SSH KEY> <YOUR USERNAME ON REMOTE HOST>@<HOSTNAME>" onCopy={onCopy} />
      </div>

      {/* Step 1: Prepare Edge Components */}
      <div className="border border-border rounded-lg p-5">
        <div className="flex items-center gap-2 mb-3">
          <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">1</span>
          <h3 className="text-sm font-semibold text-gray-900">Prepare Edge Components</h3>
        </div>

        <p className="text-xs text-gray-600 mb-1">
          1. Click <span className="font-semibold">New Agent Controller</span> or select an existing Agent Controller.
        </p>
        <p className="text-xs text-gray-500 mb-4">
          See <Link to="/edge-components/agent_controllers/new" className="text-orange-500 hover:text-orange-600">Create an Agent Controller</Link>.
        </p>

        <div className="flex items-center gap-3 mb-5">
          <Link
            to="/edge-components/agent_controllers/new"
            className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            <Plus size={12} />
            New Agent Controller
          </Link>
          <span className="text-xs text-gray-500 font-medium">OR</span>
          <div className="w-52">
            <Dropdown
              options={controllerOptions}
              value={selectedController}
              onChange={onControllerChange}
              placeholder="Select Existing"
            />
          </div>
        </div>

        {controllerSelected && (
          <>
            <p className="text-xs text-gray-600 mb-1">
              2. Click <span className="font-semibold">Create Device Code</span>.
            </p>
            <p className="text-xs text-gray-500 mb-3">
              The Device code authorizes the Agent Controller in your VM to access Dipole Cloud and is valid for 15 minutes.
            </p>

            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={onGenerateCode}
                disabled={isGenerating}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-60"
              >
                {isGenerating ? 'Generating\u2026' : 'Create Device Code'}
              </button>
              {deviceCode && (
                <div className="px-4 py-2 border border-border rounded-md text-sm font-mono tracking-widest text-gray-700 bg-gray-50">
                  {deviceCode}
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {/* Message when no controller selected */}
      {!controllerSelected && (
        <p className="text-xs text-orange-500 text-center py-4">
          To continue, complete the preceding step: Add or create an Agent Controller.
        </p>
      )}

      {/* Steps 2 & 3 only visible after controller is selected */}
      {controllerSelected && (
        <>
          {/* Step 2: Install the Agent Controller */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">2</span>
              <h3 className="text-sm font-semibold text-gray-900">Install the Agent Controller</h3>
            </div>

            <p className="text-xs text-gray-600 mb-2">
              1. Download the latest Agent Controller MSI installer:
            </p>
            <CodeBlock text="https://releases.dipolehq.com/agent_controller/1.32.3502/windows/amd64/dipole_agent_controller_windows_amd64_1.32.3502.msi" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              2. Run the MSI installer with the following command in an elevated PowerShell prompt:
            </p>
            <CodeBlock text={`msiexec /i dipole_agent_controller_windows_amd64_1.32.3502.msi TENANT_ID=${TENANT_ID} /quiet`} onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              3. Verify the Agent Controller service is running:
            </p>
            <CodeBlock text="Get-Service DipoleAgentController" onCopy={onCopy} />
          </div>

          {/* Step 3: Install the Agent Proxy */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-orange-500 text-orange-500 text-xs font-bold">3</span>
              <h3 className="text-sm font-semibold text-gray-900">Install the Agent Proxy</h3>
            </div>

            <p className="text-xs text-gray-600 mb-2">
              1. Download the latest Agent Proxy MSI installer:
            </p>
            <CodeBlock text="https://releases.dipolehq.com/agent_proxy/1.32.4999/windows/amd64/dipole_agent_proxy_windows_amd64_1.32.4999.msi" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              2. Run the MSI installer with the following command in an elevated PowerShell prompt:
            </p>
            <CodeBlock text="msiexec /i dipole_agent_proxy_windows_amd64_1.32.4999.msi AGENT_CONTROLLER=http://<AGENT CONTROLLER HOST>:5000 /quiet" onCopy={onCopy} />

            <p className="text-xs text-gray-600 mt-4 mb-2">
              3. Verify the Agent Proxy service is running:
            </p>
            <CodeBlock text="Get-Service DipoleAgentProxy" onCopy={onCopy} />
          </div>

          {/* Additional Configurations */}
          <div className="border border-border rounded-lg p-5 bg-gray-50">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Additional Configurations</h3>
            <p className="text-xs text-gray-600 mb-1">
              For further customization and advanced configurations, see{' '}
              <a href="#" className="text-orange-500 hover:text-orange-600">Dipole Windows Deployment Guide</a>.
            </p>
            <p className="text-xs text-gray-600">
              This guide always uses the latest version of Agent Controller and Proxy. For other versions, see{' '}
              <a href="#" className="text-orange-500 hover:text-orange-600">Dipole Releases</a>.
            </p>
          </div>
        </>
      )}
    </>
  )
}

function CodeBlock({ text, onCopy }: { text: string; onCopy: (text: string) => void }) {
  return (
    <div className="flex items-center gap-2 bg-gray-50 border border-border rounded-md px-4 py-2.5">
      <div className="flex-1 overflow-x-auto scrollbar-hide">
        <code className="text-xs text-gray-700 font-mono whitespace-nowrap">{text}</code>
      </div>
      <button
        type="button"
        onClick={() => onCopy(text)}
        className="text-gray-400 hover:text-gray-600 transition-colors shrink-0"
        title="Copy to clipboard"
      >
        <Copy size={14} />
      </button>
    </div>
  )
}
