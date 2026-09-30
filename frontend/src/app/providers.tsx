import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { login as loginRequest } from '../api/auth'
import { setUnauthorizedHandler } from '../api/client'
import { clearToken, getToken, getUsername, setToken, setUsername } from '../api/token'
import { AuthContext, type AuthState } from '../hooks/useAuth'

function createQueryClient(): QueryClient {
  return new QueryClient({ defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } } })
}

function AuthProvider({ queryClient, children }: { queryClient: QueryClient; children: ReactNode }) {
  const [authenticated, setAuthenticated] = useState(() => getToken() !== null)
  const [displayName, setDisplayName] = useState<string | null>(() => getUsername())

  const signOut = useCallback(() => {
    clearToken()
    queryClient.clear()
    setAuthenticated(false)
    setDisplayName(null)
  }, [queryClient])

  // The client clears the token on a 401; the guard reacts to this state change and redirects.
  useEffect(() => {
    setUnauthorizedHandler(signOut)
    return () => setUnauthorizedHandler(null)
  }, [signOut])

  const login = useCallback(async (username: string, password: string) => {
    const res = await loginRequest({ username, password })
    setToken(res.token)
    setUsername(res.user.username)
    setDisplayName(res.user.username)
    setAuthenticated(true)
  }, [])

  const value = useMemo<AuthState>(
    () => ({ isAuthenticated: authenticated, username: authenticated ? displayName : null, login, logout: signOut }),
    [authenticated, displayName, login, signOut],
  )
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function AppProviders({ queryClient, children }: { queryClient?: QueryClient; children: ReactNode }) {
  const [client] = useState(() => queryClient ?? createQueryClient())
  return (
    <QueryClientProvider client={client}>
      <AuthProvider queryClient={client}>{children}</AuthProvider>
    </QueryClientProvider>
  )
}
