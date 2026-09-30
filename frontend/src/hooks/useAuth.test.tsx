import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { apiFetch } from '../api/client'
import { getToken, setToken } from '../api/token'
import { AppProviders } from '../app/providers'
import { createTestQueryClient, jsonResponse } from '../test/utils'
import { useAuth } from './useAuth'

const fetchMock = vi.fn<typeof fetch>()

function Probe() {
  const { isAuthenticated, username, login, logout } = useAuth()
  return (
    <div>
      <p>{isAuthenticated ? 'signed in' : 'signed out'}</p>
      <p data-testid="who">{username ?? 'nobody'}</p>
      <button onClick={() => login('admin', 'admin123').catch(() => undefined)}>login</button>
      <button onClick={() => login('admin', 'nope').catch((e: Error) => document.title = e.message)}>bad login</button>
      <button onClick={logout}>logout</button>
    </div>
  )
}

function renderProbe() {
  return render(
    <AppProviders queryClient={createTestQueryClient()}>
      <Probe />
    </AppProviders>,
  )
}

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  document.title = ''
})

afterEach(() => vi.unstubAllGlobals())

describe('useAuth', () => {
  it('starts signed out without a stored token', () => {
    renderProbe()
    expect(screen.getByText('signed out')).toBeInTheDocument()
  })

  it('starts signed in when a token is already stored', () => {
    setToken('stored')
    renderProbe()
    expect(screen.getByText('signed in')).toBeInTheDocument()
  })

  it('login posts the credentials, stores the token and signs in', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { token: 'tok-1', expires_at: '2026-09-30T20:00:00Z', user: { username: 'admin' } }))
    renderProbe()

    await userEvent.click(screen.getByRole('button', { name: 'login' }))

    expect(await screen.findByText('signed in')).toBeInTheDocument()
    expect(screen.getByTestId('who')).toHaveTextContent('admin')
    expect(getToken()).toBe('tok-1')
    const [url, init] = fetchMock.mock.calls[0]!
    expect(url).toBe('/api/auth/login')
    expect(init?.body).toBe('{"username":"admin","password":"admin123"}')
  })

  it('wrong credentials reject with the Spanish message and keep the user signed out', async () => {
    fetchMock.mockResolvedValue(jsonResponse(401, { error: { code: 'invalid_credentials', message: 'invalid username or password' } }))
    renderProbe()

    await userEvent.click(screen.getByRole('button', { name: 'bad login' }))

    await vi.waitFor(() => expect(document.title).toBe('Usuario o contraseña incorrectos.'))
    expect(screen.getByText('signed out')).toBeInTheDocument()
    expect(getToken()).toBeNull()
  })

  it('logout clears the token and signs out', async () => {
    setToken('stored')
    renderProbe()

    await userEvent.click(screen.getByRole('button', { name: 'logout' }))

    expect(screen.getByText('signed out')).toBeInTheDocument()
    expect(screen.getByTestId('who')).toHaveTextContent('nobody')
    expect(getToken()).toBeNull()
  })

  it('remembers the username across a reload', () => {
    setToken('stored')
    window.localStorage.setItem('energy.username', 'admin')
    renderProbe()
    expect(screen.getByTestId('who')).toHaveTextContent('admin')
  })

  it('signs out when any API call answers 401', async () => {
    setToken('expired')
    fetchMock.mockResolvedValue(jsonResponse(401, { error: { code: 'unauthorized', message: 'token expired' } }))
    renderProbe()

    await act(async () => {
      await apiFetch('/api/meters').catch(() => undefined)
    })

    expect(screen.getByText('signed out')).toBeInTheDocument()
    expect(getToken()).toBeNull()
  })
})
