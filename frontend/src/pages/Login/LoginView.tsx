import { useState, type FormEvent } from 'react'
import { BoltIcon } from '../../components/atoms/Icon'
import { Button } from '../../components/atoms/Button'
import { Card } from '../../components/atoms/Card'
import { Input } from '../../components/atoms/Input'

export interface LoginViewProps {
  onSubmit: (username: string, password: string) => void
  submitting: boolean
  error: string | null
}

export function LoginView({ onSubmit, submitting, error }: LoginViewProps) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [touched, setTouched] = useState(false)

  const usernameError = touched && username.trim() === '' ? 'Enter your username' : undefined
  const passwordError = touched && password === '' ? 'Enter your password' : undefined

  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setTouched(true)
    if (username.trim() === '' || password === '') return
    onSubmit(username.trim(), password)
  }

  return (
    <div className="grid min-h-screen lg:grid-cols-[minmax(0,5fr)_minmax(0,4fr)]">
      <section className="relative hidden flex-col justify-between overflow-hidden bg-sidebar p-12 text-white lg:flex">
        <div className="flex items-center gap-2.5">
          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent text-sidebar">
            <BoltIcon />
          </span>
          <span className="text-lg font-semibold tracking-tight">Energy Console</span>
        </div>
        <div className="max-w-md">
          <h2 className="text-3xl font-semibold leading-tight tracking-tight">Find the meters that need attention, and why.</h2>
          <p className="mt-4 text-base leading-relaxed text-sidebar-text">
            Spot consumption anomalies, separate real faults from scheduled events, and get a clear next action for every meter.
          </p>
        </div>
        <p className="text-xs text-sidebar-text/60">AI energy management platform</p>
        <div aria-hidden="true" className="pointer-events-none absolute -right-24 -top-24 h-72 w-72 rounded-full bg-primary/20 blur-3xl" />
      </section>

      <main className="flex items-center justify-center px-4 py-12">
        <div className="w-full max-w-sm">
          <div className="mb-8 flex items-center gap-2.5 lg:hidden">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-sidebar text-accent">
              <BoltIcon />
            </span>
            <span className="text-lg font-semibold tracking-tight">Energy Console</span>
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Sign in</h1>
          <p className="mt-1.5 text-sm text-muted">Use your console account to continue.</p>

          <Card className="mt-6">
            <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
              {error && (
                <p role="alert" className="rounded-lg bg-sev-high/10 px-3 py-2.5 text-sm font-medium text-sev-high-ink">
                  {error}
                </p>
              )}
              <Input
                id="username"
                label="Username"
                autoComplete="username"
                autoFocus
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                error={usernameError}
              />
              <Input
                id="password"
                label="Password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                error={passwordError}
              />
              <Button type="submit" loading={submitting} className="mt-1 w-full">
                Sign in
              </Button>
            </form>
          </Card>

          <p className="mt-5 rounded-lg border border-dashed border-border px-3 py-2.5 text-center text-xs text-muted">
            Demo access: <span className="tabular font-semibold text-text">admin / admin123</span>
          </p>
        </div>
      </main>
    </div>
  )
}
