/** Standard paginated list response from the API */
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  perPage: number
}

/** Standard API error shape */
export interface ApiErrorResponse {
  error: string
  message?: string
  statusCode: number
}
