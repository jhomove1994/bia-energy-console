import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { getToken } from '../../api/token'
import { AuthGuard } from '../../app/AuthGuard'
import { AppProviders } from '../../app/providers'
import { createTestQueryClient, jsonResponse } from '../../test/utils'
import { LoginPage } from './LoginPage'
import { routerFuture } from '../../app/router'

const fetchMock = vi.fn<typeof fetch>()

function renderApp(path = '/login') {
  render(
    <AppProviders queryClient={createTestQueryClient()}>
      <MemoryRouter future={routerFuture} initialEntries={[path]}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<AuthGuard />}>
            <Route path="/" element={<p>dashboard screen</p>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </AppProviders>,
  )
}

async function signIn(username: string, password: string) {
  await userEvent.type(screen.getByLabelText('Usuario'), username)
  await userEvent.type(screen.getByLabelText('Contraseña'), password)
  await userEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))
}

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => vi.unstubAllGlobals())

describe('LoginPage', () => {
  it('opens the dashboard after a successful login', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { token: 't', expires_at: '2026-09-30T20:00:00Z', user: { username: 'admin' } }))
    renderApp()

    await signIn('admin', 'admin123')

    expect(await screen.findByText('dashboard screen')).toBeInTheDocument()
    expect(getToken()).toBe('t')
  })

  it('shows an error and stays on login with wrong credentials', async () => {
    fetchMock.mockResolvedValue(jsonResponse(401, { error: { code: 'invalid_credentials', message: 'invalid credentials' } }))
    renderApp()

    await signIn('admin', 'wrong')

    expect(await screen.findByRole('alert')).toHaveTextContent('Usuario o contraseña incorrectos.')
    expect(screen.queryByText('dashboard screen')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Usuario')).toBeInTheDocument()
    expect(getToken()).toBeNull()
  })

  it('reports an unreachable API instead of a credentials error', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))
    renderApp()

    await signIn('admin', 'admin123')

    expect(await screen.findByRole('alert')).toHaveTextContent('No se pudo conectar con el servidor')
  })

  it('goes straight to the dashboard when already signed in', () => {
    window.localStorage.setItem('energy.token', 'existing')
    renderApp('/login')
    expect(screen.getByText('dashboard screen')).toBeInTheDocument()
  })
})
