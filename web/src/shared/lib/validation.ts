import { z } from 'zod'

// Access Policy — only name + client + server are truly required.
// Trust Providers and Credential Providers are optional (right-side policies
// don't need Trust Providers; some policies use passthrough without credentials).
// Status defaults to 'active' if not explicitly set.
export const policyFormSchema = z.object({
  name: z.string().min(1, 'Policy name is required'),
  description: z.string().optional().default(''),
  status: z.string().default('active'),
  tags: z.array(z.object({ key: z.string(), value: z.string() })).default([]),

  clientWorkloadId: z.string().default(''),
  client: z.any(),

  serverWorkloadId: z.string().default(''),
  server: z.any(),

  trustProviderIds: z.array(z.string()).default([]),
  trustProviders: z.any().default([]),

  accessConditionIds: z.array(z.string()).default([]),
  accessConditions: z.any().default([]),

  credentialProviderId: z.string().optional().default(''),
  credential: z.any(),
})

export type PolicyFormSchema = z.infer<typeof policyFormSchema>

// Client Workload — name + at least one identifier required.
export const clientWorkloadSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional().default(''),
  identifiers: z.array(z.object({
    type: z.string().min(1, 'Identifier type is required'),
    values: z.array(z.string().min(1, 'Value is required')).min(1),
  })).min(1, 'At least one identifier is required'),
  tags: z.array(z.object({ key: z.string(), value: z.string() })).default([]),
})

// Server Workload — name + host + port + protocol + auth method required.
export const serverWorkloadSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional().default(''),
  host: z.string().min(1, 'Host is required'),
  port: z.string().min(1, 'Port is required'),
  appProtocol: z.string().min(1, 'Application protocol is required'),
  transportProtocol: z.string().default('TCP'),
  tls: z.boolean().default(false),
  authMethod: z.string().min(1, 'Authentication method is required'),
  authScheme: z.string().optional().default(''),
  headerName: z.string().optional().default(''),
  urlPath: z.string().optional().default(''),
  forwardToPort: z.string().optional().default(''),
  forwardTls: z.boolean().default(false),
  forwardTlsVerification: z.string().default('full'),
  staticHeaders: z.array(z.object({ name: z.string(), value: z.string() })).default([]),
  tags: z.array(z.object({ key: z.string(), value: z.string() })).default([]),
})

// Trust Provider — name + type required. Match rules are optional.
export const trustProviderSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional().default(''),
  trustProviderType: z.string().min(1, 'Trust Provider type is required'),
  matchRules: z.array(z.object({ attribute: z.string(), value: z.string() })).default([]),
  tags: z.array(z.object({ key: z.string(), value: z.string() })).default([]),
})

// Access Condition — name + condition type required.
export const accessConditionSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional().default(''),
  integration: z.string().min(1, 'Condition type is required'),
  tags: z.array(z.object({ key: z.string(), value: z.string() })).default([]),
})

// Credential Provider — name + type required.
export const credentialProviderSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional().default(''),
  credentialType: z.string().min(1, 'Credential type is required'),
  tags: z.array(z.object({ key: z.string(), value: z.string() })).default([]),
})
