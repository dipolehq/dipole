import { Monitor, CheckCircle, Clock, Lock, Server } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export interface StatConfig {
  key: string
  title: string
  icon: LucideIcon
  color: string
}

export const STAT_CARDS: StatConfig[] = [
  { key: 'clientWorkloads', title: 'Client Workloads', icon: Monitor, color: 'text-gray-600' },
  { key: 'trustProviders', title: 'Trust Providers', icon: CheckCircle, color: 'text-gray-600' },
  { key: 'accessConditions', title: 'Access Conditions', icon: Clock, color: 'text-gray-600' },
  { key: 'credentialProviders', title: 'Credential Providers', icon: Lock, color: 'text-gray-600' },
  { key: 'serverWorkloads', title: 'Server Workloads', icon: Server, color: 'text-gray-600' },
]
