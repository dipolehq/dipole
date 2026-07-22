export type IdpType = 'saml_2.0' | 'oidc_1.0'
export type AuthMethod = 'client_secret' | 'public_private_keypair'

export const IDP_TYPE_OPTIONS = [
  { value: 'saml_2.0', label: 'SAML 2.0' },
  { value: 'oidc_1.0', label: 'OIDC 1.0' },
] as const

export type LogStreamEventType = 'access_authorization' | 'audit_logs' | 'workload_events'
export type LogStreamDestinationType = 'aws_s3' | 'gcs_bucket' | 'crowdstrike_siem' | 'splunk_siem'

export const LOG_STREAM_EVENT_TYPE_OPTIONS = [
  { value: 'access_authorization', label: 'Access Authorization Events' },
  { value: 'audit_logs', label: 'Audit Logs' },
  { value: 'workload_events', label: 'Workload Events' },
] as const

export const LOG_STREAM_DESTINATION_OPTIONS = [
  { value: 'aws_s3', label: 'AWS S3 using Bucket Policy' },
  { value: 'gcs_bucket', label: 'GCS Bucket using Workload Identity Federation' },
  { value: 'crowdstrike_siem', label: 'CrowdStrike Next-Gen SIEM using Http Event Collector' },
  { value: 'splunk_siem', label: 'Splunk SIEM using Http Event Collector (HEC)' },
] as const
