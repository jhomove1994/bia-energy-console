import { clearToken, getToken } from './token'

/** Error raised for every failed request; `code` is the API error code, `network`, or `http_<status>`. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export interface RequestOptions {
  method?: 'GET' | 'POST'
  body?: unknown
  /** Set to false for calls where a 401 is an expected answer (login). Default true. */
  handleUnauthorized?: boolean
}

let onUnauthorized: (() => void) | null = null

/** Registers the callback run after a 401 cleared the token (the app redirects to login). */
export function setUnauthorizedHandler(handler: (() => void) | null): void {
  onUnauthorized = handler
}

interface ErrorBody {
  error?: { code?: string; message?: string }
}

async function toApiError(res: Response): Promise<ApiError> {
  let body: ErrorBody | null = null
  try {
    body = (await res.json()) as ErrorBody
  } catch {
    body = null
  }
  const code = body?.error?.code ?? `http_${res.status}`
  const message = body?.error?.message ?? `Request failed with status ${res.status}`
  return new ApiError(res.status, code, message)
}

export async function apiFetch<T = unknown>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, handleUnauthorized = true } = options
  const headers = new Headers({ Accept: 'application/json' })
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (body !== undefined) headers.set('Content-Type', 'application/json')

  let res: Response
  try {
    res = await fetch(path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) })
  } catch {
    throw new ApiError(0, 'network', 'Cannot reach the server. Check your connection and try again.')
  }

  if (!res.ok) {
    if (res.status === 401 && handleUnauthorized) {
      clearToken()
      onUnauthorized?.()
    }
    throw await toApiError(res)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}
