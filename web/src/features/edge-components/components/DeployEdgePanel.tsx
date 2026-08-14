import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Copy } from 'lucide-react'
import { SiKubernetes, SiUbuntu, SiRedhat } from '@icons-pack/react-simple-icons'
import { Dropdown } from '@/shared/components/Dropdown'
import { useAgentControllers, useRegenerateDeviceCode } from '@/features/edge-components/hooks/useAgentControllers'

type Environment = 'kubernetes' | 'ubuntu' | 'redhat' | 'windows'

const ENVIRONMENTS: { value: Environment; label: string; subtitle: string; icon: string }[] = [
  { value: 'kubernetes', label: 'Kubernetes', subtitle: 'Kubernetes', icon: '⎈' },
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
    <>
      {/* Tab bar — replaces left sidebar */}
      <div className="flex items-center gap-1 px-4 py-3 shrink-0">
        {ENVIRONMENTS.map((env) => (
          <button
            key={env.value}
            type="button"
            onClick={() => setSelectedEnv(env.value)}
            className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              selectedEnv === env.value
                ? 'bg-gray-100 text-gray-900'
                : 'text-gray-600 hover:bg-gray-50 hover:text-gray-800'
            }`}
          >
            <EnvironmentLogo env={env.value} />
            <span>{env.label}</span>
          </button>
        ))}
      </div>

      {/* Content box */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="p-6 space-y-6">
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
    </>
  )
}

function EnvironmentLogo({ env }: { env: Environment }) {
  switch (env) {
    case 'kubernetes':
      return <SiKubernetes size={18} color="#326CE5" />
    case 'ubuntu':
      return <SiUbuntu size={18} color="#E95420" />
    case 'redhat':
      return <SiRedhat size={18} color="#EE0000" />
    case 'windows':
      return (
        <svg className="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none">
          <path d="M0 3.5L9.8 2.1v9.4H0V3.5z" fill="#0078D4"/>
          <path d="M10.9 1.9L24 0v11.5H10.9V1.9z" fill="#0078D4"/>
          <path d="M0 12.5h9.8v9.4L0 20.5v-8z" fill="#0078D4"/>
          <path d="M10.9 12.5H24V24l-13.1-1.9v-9.6z" fill="#0078D4"/>
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
        <h3 className="text-sm font-semibold text-gray-700 mb-2">Prerequisites</h3>
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
          <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">1</span>
          <h3 className="text-sm font-semibold text-gray-900">Prepare Edge Components</h3>
        </div>

        <p className="text-xs text-gray-600 mb-1">
          1. Click <span className="font-semibold">New Agent Controller</span> or select an existing Agent Controller.
        </p>
        <p className="text-xs text-gray-500 mb-4">
          See <Link to="/edge-components/agent_controllers/new" className="text-gray-600 hover:text-gray-700">Create an Agent Controller</Link>.
        </p>

        <div className="flex items-center gap-3 mb-5">
          <Link
            to="/edge-components/agent_controllers/new"
            className="flex items-center gap-1.5 px-4 py-2 border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 rounded-md transition-colors"
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
                className="px-4 py-2 border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 rounded-md transition-colors disabled:opacity-60"
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
        <p className="text-xs text-gray-600 text-center py-4">
          To continue, complete the preceding step: Add or create an Agent Controller.
        </p>
      )}

      {/* Steps 2 & 3 only visible after controller is selected */}
      {controllerSelected && (
        <>
          {/* Step 2: Install Helm Chart */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">2</span>
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
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">3</span>
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
          <div className="bg-gray-50 rounded-md p-4">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Additional Configurations</h3>
            <p className="text-xs text-gray-600">
              For further configuration options, refer to the official documentation:{' '}
              <a href="#" className="text-gray-600 hover:text-gray-700">Deploying Kubernetes Configurations</a>.
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
        <h3 className="text-sm font-semibold text-gray-700 mb-2">Prerequisites</h3>
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
          <a href="#" className="text-gray-600 hover:text-gray-700">About Colocating Edge Components</a>.
        </p>
        <p className="text-xs text-gray-600 mb-2">
          Make sure the VM is running and accessible via SSH.
        </p>
        <CodeBlock text="ssh -i <PATH TO YOUR SSH KEY> <YOUR USERNAME ON REMOTE HOST>@<HOSTNAME>" onCopy={onCopy} />
      </div>

      {/* Step 1: Prepare Edge Components */}
      <div className="border border-border rounded-lg p-5">
        <div className="flex items-center gap-2 mb-3">
          <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">1</span>
          <h3 className="text-sm font-semibold text-gray-900">Prepare Edge Components</h3>
        </div>

        <p className="text-xs text-gray-600 mb-1">
          1. Click <span className="font-semibold">New Agent Controller</span> or select an existing Agent Controller.
        </p>
        <p className="text-xs text-gray-500 mb-4">
          See <Link to="/edge-components/agent_controllers/new" className="text-gray-600 hover:text-gray-700">Create an Agent Controller</Link>.
        </p>

        <div className="flex items-center gap-3 mb-5">
          <Link
            to="/edge-components/agent_controllers/new"
            className="flex items-center gap-1.5 px-4 py-2 border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 rounded-md transition-colors"
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
                className="px-4 py-2 border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 rounded-md transition-colors disabled:opacity-60"
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
        <p className="text-xs text-gray-600 text-center py-4">
          To continue, complete the preceding step: Add or create an Agent Controller.
        </p>
      )}

      {/* Steps 2 & 3 only visible after controller is selected */}
      {controllerSelected && (
        <>
          {/* Step 2: Install the Agent Controller */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">2</span>
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
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">3</span>
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
          <div className="bg-gray-50 rounded-md p-4">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Additional Configurations</h3>
            <p className="text-xs text-gray-600 mb-1">
              For further customization and advanced configurations, see{' '}
              <a href="#" className="text-gray-600 hover:text-gray-700">Dipole Virtual Machine Deployment Guide</a>.
            </p>
            <p className="text-xs text-gray-600">
              This guide always uses the latest version of Agent Controller and Proxy. For other versions, see{' '}
              <a href="#" className="text-gray-600 hover:text-gray-700">Dipole Releases</a>.
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
        <h3 className="text-sm font-semibold text-gray-700 mb-2">Prerequisites</h3>
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
          <a href="#" className="text-gray-600 hover:text-gray-700">About Colocating Edge Components</a>.
        </p>
        <p className="text-xs text-gray-600 mb-2">
          Make sure the VM is running and accessible via SSH.
        </p>
        <CodeBlock text="ssh -i <PATH TO YOUR SSH KEY> <YOUR USERNAME ON REMOTE HOST>@<HOSTNAME>" onCopy={onCopy} />
      </div>

      {/* Step 1: Prepare Edge Components */}
      <div className="border border-border rounded-lg p-5">
        <div className="flex items-center gap-2 mb-3">
          <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">1</span>
          <h3 className="text-sm font-semibold text-gray-900">Prepare Edge Components</h3>
        </div>

        <p className="text-xs text-gray-600 mb-1">
          1. Click <span className="font-semibold">New Agent Controller</span> or select an existing Agent Controller.
        </p>
        <p className="text-xs text-gray-500 mb-4">
          See <Link to="/edge-components/agent_controllers/new" className="text-gray-600 hover:text-gray-700">Create an Agent Controller</Link>.
        </p>

        <div className="flex items-center gap-3 mb-5">
          <Link
            to="/edge-components/agent_controllers/new"
            className="flex items-center gap-1.5 px-4 py-2 border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 rounded-md transition-colors"
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
                className="px-4 py-2 border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 rounded-md transition-colors disabled:opacity-60"
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
        <p className="text-xs text-gray-600 text-center py-4">
          To continue, complete the preceding step: Add or create an Agent Controller.
        </p>
      )}

      {/* Steps 2 & 3 only visible after controller is selected */}
      {controllerSelected && (
        <>
          {/* Step 2: Install the Agent Controller */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">2</span>
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
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">3</span>
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
          <div className="bg-gray-50 rounded-md p-4">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Additional Configurations</h3>
            <p className="text-xs text-gray-600 mb-1">
              For further customization and advanced configurations, see{' '}
              <a href="#" className="text-gray-600 hover:text-gray-700">Dipole Virtual Machine Deployment Guide</a>.
            </p>
            <p className="text-xs text-gray-600 mb-1">
              For <span className="font-semibold">SELinux configurations</span>, see{' '}
              <a href="#" className="text-gray-600 hover:text-gray-700">Dipole SELinux Deployment Guide</a>.
            </p>
            <p className="text-xs text-gray-600">
              This guide always uses the latest version of Agent Controller and Proxy. For other versions, see{' '}
              <a href="#" className="text-gray-600 hover:text-gray-700">Dipole Releases</a>.
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
        <h3 className="text-sm font-semibold text-gray-700 mb-2">Prerequisites</h3>
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
          <a href="#" className="text-gray-600 hover:text-gray-700">About Colocating Edge Components</a>.
        </p>
        <p className="text-xs text-gray-600 mb-2">
          Make sure the VM is running and accessible via RDP or SSH.
        </p>
        <CodeBlock text="ssh -i <PATH TO YOUR SSH KEY> <YOUR USERNAME ON REMOTE HOST>@<HOSTNAME>" onCopy={onCopy} />
      </div>

      {/* Step 1: Prepare Edge Components */}
      <div className="border border-border rounded-lg p-5">
        <div className="flex items-center gap-2 mb-3">
          <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">1</span>
          <h3 className="text-sm font-semibold text-gray-900">Prepare Edge Components</h3>
        </div>

        <p className="text-xs text-gray-600 mb-1">
          1. Click <span className="font-semibold">New Agent Controller</span> or select an existing Agent Controller.
        </p>
        <p className="text-xs text-gray-500 mb-4">
          See <Link to="/edge-components/agent_controllers/new" className="text-gray-600 hover:text-gray-700">Create an Agent Controller</Link>.
        </p>

        <div className="flex items-center gap-3 mb-5">
          <Link
            to="/edge-components/agent_controllers/new"
            className="flex items-center gap-1.5 px-4 py-2 border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 rounded-md transition-colors"
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
                className="px-4 py-2 border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 rounded-md transition-colors disabled:opacity-60"
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
        <p className="text-xs text-gray-600 text-center py-4">
          To continue, complete the preceding step: Add or create an Agent Controller.
        </p>
      )}

      {/* Steps 2 & 3 only visible after controller is selected */}
      {controllerSelected && (
        <>
          {/* Step 2: Install the Agent Controller */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">2</span>
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
              <span className="flex items-center justify-center w-6 h-6 rounded-full border-2 border-gray-400 text-gray-600 text-xs font-bold">3</span>
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
          <div className="bg-gray-50 rounded-md p-4">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Additional Configurations</h3>
            <p className="text-xs text-gray-600 mb-1">
              For further customization and advanced configurations, see{' '}
              <a href="#" className="text-gray-600 hover:text-gray-700">Dipole Windows Deployment Guide</a>.
            </p>
            <p className="text-xs text-gray-600">
              This guide always uses the latest version of Agent Controller and Proxy. For other versions, see{' '}
              <a href="#" className="text-gray-600 hover:text-gray-700">Dipole Releases</a>.
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
