import { Outlet } from 'react-router-dom'
import { AppShell } from '../components/organisms/AppShell'
import { useAuth } from '../hooks/useAuth'
import { navItems } from './navigation'

/** Container for the shell: wires the session into the presentational AppShell. */
export function AppLayout() {
  const { username, logout } = useAuth()
  return (
    <AppShell navItems={navItems} username={username ?? 'Usuario'} onLogout={logout}>
      <Outlet />
    </AppShell>
  )
}
