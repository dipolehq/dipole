import { Monitor, CheckCircle, Clock, Lock, Server } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export interface StatConfig {
  key: string
  title: string
  icon: LucideIcon
  color: string
}

export const STAT_CARDS: StatConfig[] = [
  { key: 'clientWorkloads', title: 'Client Workloads', icon: Monitor, color: 'text-green-500' },
  { key: 'trustProviders', title: 'Trust Providers', icon: CheckCircle, color: 'text-purple-500' },
  { key: 'accessConditions', title: 'Access Conditions', icon: Clock, color: 'text-purple-500' },
  { key: 'credentialProviders', title: 'Credential Providers', icon: Lock, color: 'text-purple-500' },
  { key: 'serverWorkloads', title: 'Server Workloads', icon: Server, color: 'text-orange-500' },
]

export const CREDENTIAL_USAGE_CONFIG = {
  labels: [
    'Dipole Access Token',
    'Username & Password',
    'API Key',
    'OAuth 2.0 Client Credentials',
    'JSON Web Token (JWT)',
    'AWS Security Token Service',
    'Vault Client Token',
  ],
  colors: ['#ffd8a8', '#a5d8ff', '#ffec99', '#b2f2bb', '#d0bfff', '#fcc2d7', '#99e9f2'],
}

export const PROTOCOL_USAGE_CONFIG = {
  labels: ['TCP', 'HTTP', 'MySQL', 'PostgreSQL', 'Redis', 'Redshift', 'Snowflake'],
  colors: ['#ffd8a8', '#a5d8ff', '#ffec99', '#b2f2bb', '#d0bfff', '#fcc2d7', '#99e9f2'],
}

export const DENIAL_REASONS_CONFIG = {
  labels: ['No Matching Policy', 'Attestation Failed', 'Condition Failed', 'Credential Retrieval Failed'],
  colors: ['#ffd8a8', '#fcc2d7', '#d0bfff', '#a5d8ff'],
}

export const ATTESTATION_RESULTS_CONFIG = {
  labels: ['Attested', 'Match Rule Failed', 'Invalid Signature', 'No Data Found'],
  colors: ['#b2f2bb', '#ffd8a8', '#fcc2d7', '#e9ecef'],
}
