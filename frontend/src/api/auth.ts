import { apiFetch } from './client'
import type { LoginRequest, LoginResponse } from './types'

/** A 401 here means wrong credentials, not an expired session, so it must not trigger the global logout. */
export function login(credentials: LoginRequest): Promise<LoginResponse> {
  return apiFetch<LoginResponse>('/api/auth/login', { method: 'POST', body: credentials, handleUnauthorized: false })
}
