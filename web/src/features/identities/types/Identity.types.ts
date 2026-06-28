export interface Identity {
  id: string
  name: string
  description: string
  identifiers: IdentityIdentifier[]
  createdAt: string
  updatedAt: string
}

export interface IdentityIdentifier {
  type: string
  value: string
}

export interface IdentityListResponse {
  items: Identity[]
  total: number
  page: number
  perPage: number
}
