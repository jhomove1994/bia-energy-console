import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { setToken } from '../api/token'
import { createTestQueryClient } from '../test/utils'
import { AuthGuard } from './AuthGuard'
import { AppProviders } from './providers'

function renderAt(path: string) {
  return render(
    <AppProviders queryClient={createTestQueryClient()}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/login" element={<p>login screen</p>} />
          <Route element={<AuthGuard />}>
            <Route path="/" element={<p>dashboard screen</p>} />
            <Route path="/meters" element={<p>meters screen</p>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </AppProviders>,
  )
}

describe('AuthGuard', () => {
  it('redirects an unauthenticated visit to /login', () => {
    renderAt('/')
    expect(screen.getByText('login screen')).toBeInTheDocument()
    expect(screen.queryByText('dashboard screen')).not.toBeInTheDocument()
  })

  it('redirects deep links too', () => {
    renderAt('/meters')
    expect(screen.getByText('login screen')).toBeInTheDocument()
  })

  it('renders the protected route when a token is stored', () => {
    setToken('tok')
    renderAt('/meters')
    expect(screen.getByText('meters screen')).toBeInTheDocument()
    expect(screen.queryByText('login screen')).not.toBeInTheDocument()
  })
})
