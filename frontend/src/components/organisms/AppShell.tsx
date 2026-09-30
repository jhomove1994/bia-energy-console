import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { BoltIcon, LogoutIcon } from '../atoms/Icon'
import { Button } from '../atoms/Button'

export interface NavItem {
  to: string
  label: string
  end?: boolean
  icon?: ReactNode
}

export interface AppShellProps {
  navItems: NavItem[]
  username: string
  onLogout: () => void
  children: ReactNode
}

function Brand() {
  return (
    <div className="flex items-center gap-2.5">
      <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent text-sidebar">
        <BoltIcon width={18} height={18} />
      </span>
      <span className="text-base font-semibold tracking-tight">Energy Console</span>
    </div>
  )
}

function Sidebar({ navItems }: { navItems: NavItem[] }) {
  return (
    <aside className="hidden w-60 shrink-0 flex-col bg-sidebar text-sidebar-text md:flex">
      <div className="flex h-16 items-center px-5 text-white">
        <Brand />
      </div>
      <nav aria-label="Primary" className="flex flex-1 flex-col gap-1 px-3 py-4">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              `flex h-10 items-center gap-3 rounded-lg px-3 text-sm font-medium transition-colors duration-fast ease-console ${
                isActive ? 'bg-sidebar-hover text-white shadow-[inset_3px_0_0_rgb(var(--accent))]' : 'hover:bg-sidebar-hover/70 hover:text-white'
              }`
            }
          >
            {item.icon}
            {item.label}
          </NavLink>
        ))}
      </nav>
      <p className="px-5 pb-5 text-xs text-sidebar-text/60">AI energy analytics</p>
    </aside>
  )
}

function Topbar({ username, onLogout }: { username: string; onLogout: () => void }) {
  return (
    <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b border-border bg-surface/90 px-4 backdrop-blur md:px-8">
      <div className="text-sidebar md:hidden">
        <Brand />
      </div>
      <div className="ml-auto flex items-center gap-3">
        <span aria-hidden="true" className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/15 text-xs font-semibold uppercase text-primary-strong">
          {username.slice(0, 1)}
        </span>
        <span className="hidden text-sm font-medium text-text sm:inline">{username}</span>
        <Button variant="ghost" onClick={onLogout} className="h-9 px-3">
          <LogoutIcon width={16} height={16} />
          Sign out
        </Button>
      </div>
    </header>
  )
}

/** Application chrome: skip link, navy sidebar (md+), topbar with the session controls, and the page slot. */
export function AppShell({ navItems, username, onLogout, children }: AppShellProps) {
  return (
    <div className="flex min-h-screen">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:left-3 focus:top-3 focus:z-50 focus:rounded-lg focus:bg-surface focus:px-4 focus:py-2 focus:text-sm focus:font-semibold focus:shadow-raised"
      >
        Skip to content
      </a>
      <Sidebar navItems={navItems} />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar username={username} onLogout={onLogout} />
        <main id="main" tabIndex={-1} className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 md:px-8 md:py-8">
          {children}
        </main>
      </div>
    </div>
  )
}
