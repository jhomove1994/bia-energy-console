import { useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useAuth } from '../../hooks/useAuth'
import { LoginView } from './LoginView'

function messageFor(error: unknown): string {
  if (error instanceof ApiError) {
    return error.code === 'invalid_credentials' ? 'Incorrect username or password.' : error.message
  }
  return 'Something went wrong. Please try again.'
}

export function LoginPage() {
  const { isAuthenticated, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const from = (location.state as { from?: string } | null)?.from ?? '/'
  if (isAuthenticated) return <Navigate to={from} replace />

  async function handleSubmit(username: string, password: string) {
    setSubmitting(true)
    setError(null)
    try {
      await login(username, password)
      navigate(from, { replace: true })
    } catch (e) {
      setError(messageFor(e))
    } finally {
      setSubmitting(false)
    }
  }

  return <LoginView onSubmit={handleSubmit} submitting={submitting} error={error} />
}
