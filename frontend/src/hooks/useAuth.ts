import { createContext, useContext } from 'react'

export interface AuthState {
  isAuthenticated: boolean
  /** Rejects with an ApiError (e.g. code `invalid_credentials`) when the API refuses. */
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

export const AuthContext = createContext<AuthState | null>(null)

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside <AppProviders>')
  return ctx
}
