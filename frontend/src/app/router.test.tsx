import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router-dom'
import type { DashboardSummary } from '../api/types'
import { demoAnomalies, demoAnomalyList } from '../test/fixtures'
import { createTestQueryClient, jsonResponse } from '../test/utils'
import { AppProviders } from './providers'
import { AppRoutes, routerFuture } from './router'

const fetchMock = vi.fn<typeof fetch>()

const summary: DashboardSummary = {
  meters_count: 12,
  total_kwh: 5000,
  anomalies_count: 4,
  high_priority_count: 2,
  aggregate_confidence: 0.9,
  last_analysis: { id: 'a-1', status: 'done', finished_at: '2026-09-30T10:15:00Z' },
  summary_text: '4 anomalies detected · 2 require priority attention',
}

function PathProbe() {
  return <output aria-label="current path">{useLocation().pathname}</output>
}

function renderAt(path: string) {
  render(
    <AppProviders queryClient={createTestQueryClient()}>
      <MemoryRouter future={routerFuture} initialEntries={[path]}>
        <AppRoutes />
        <PathProbe />
      </MemoryRouter>
    </AppProviders>,
  )
}

const currentPath = () => screen.getByLabelText('current path')

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  fetchMock.mockImplementation(async (input, init) => {
    const key = `${init?.method ?? 'GET'} ${String(input)}`
    if (key === 'POST /api/auth/login') {
      const body = JSON.parse(String(init?.body)) as { password: string }
      return body.password === 'admin123'
        ? jsonResponse(200, { token: 'tok', expires_at: '2026-09-30T20:00:00Z', user: { username: 'admin' } })
        : jsonResponse(401, { error: { code: 'invalid_credentials', message: 'invalid credentials' } })
    }
    if (key === 'GET /api/dashboard/summary') {
      return new Headers(init?.headers).get('Authorization') === 'Bearer tok'
        ? jsonResponse(200, summary)
        : jsonResponse(401, { error: { code: 'unauthorized', message: 'missing token' } })
    }
    if (key === 'GET /api/meters') {
      return jsonResponse(200, { items: [{ meter_id: 'M-109', status: 'Critical', last_reading_at: null, total_kwh: 10, variation_pct: 110 }] })
    }
    if (key === 'GET /api/ai/analysis/latest') return jsonResponse(204)
    if (key === 'GET /api/anomalies') return jsonResponse(200, demoAnomalyList())
    if (key === 'GET /api/anomalies/an-2') return jsonResponse(200, demoAnomalies()[1])
    throw new Error(`unexpected request ${key}`)
  })
})

afterEach(() => vi.unstubAllGlobals())

describe('app routes', () => {
  it('sends an anonymous visitor to the login page', () => {
    renderAt('/')
    expect(currentPath().textContent).toBe('/login')
    expect(screen.getByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
  })

  it('logs in and lands on the dashboard inside the app shell', async () => {
    renderAt('/login')

    await userEvent.type(screen.getByLabelText('Username'), 'admin')
    await userEvent.type(screen.getByLabelText('Password'), 'admin123')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(await screen.findByText('4 anomalies detected · 2 require priority attention')).toBeInTheDocument()
    expect(currentPath().textContent).toBe('/')
    expect(screen.getByRole('link', { name: 'Dashboard' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByText('admin')).toBeInTheDocument()
  })

  it('returns to the login page after signing out', async () => {
    renderAt('/login')
    await userEvent.type(screen.getByLabelText('Username'), 'admin')
    await userEvent.type(screen.getByLabelText('Password'), 'admin123')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    await screen.findByText('4 anomalies detected · 2 require priority attention')

    await userEvent.click(screen.getByRole('button', { name: 'Sign out' }))

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(currentPath().textContent).toBe('/login')
  })

  it('redirects to login when the session expires (401) while browsing', async () => {
    window.localStorage.setItem('energy.token', 'expired-token')
    renderAt('/')

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(currentPath().textContent).toBe('/login')
    expect(window.localStorage.getItem('energy.token')).toBeNull()
  })

  it('redirects unknown paths to the dashboard', async () => {
    window.localStorage.setItem('energy.token', 'tok')
    renderAt('/nowhere')
    expect(await screen.findByText('4 anomalies detected · 2 require priority attention')).toBeInTheDocument()
    expect(currentPath().textContent).toBe('/')
  })
})

describe('meters routes', () => {
  it('opens the meters list from the sidebar and marks it as current', async () => {
    window.localStorage.setItem('energy.token', 'tok')
    renderAt('/')
    await screen.findByText('4 anomalies detected · 2 require priority attention')

    await userEvent.click(screen.getByRole('link', { name: 'Meters' }))

    expect(await screen.findByRole('heading', { level: 1, name: 'Meters' })).toBeInTheDocument()
    expect(currentPath().textContent).toBe('/meters')
    expect(screen.getByRole('link', { name: 'Meters' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Dashboard' })).not.toHaveAttribute('aria-current')
  })

  it('opens a meter from its row', async () => {
    window.localStorage.setItem('energy.token', 'tok')
    fetchMock.mockImplementation(async (input) => {
      const url = String(input)
      if (url === '/api/meters') {
        return jsonResponse(200, { items: [{ meter_id: 'M-109', status: 'Critical', last_reading_at: null, total_kwh: 10, variation_pct: 110 }] })
      }
      return jsonResponse(404, { error: { code: 'not_found', message: 'meter not found' } })
    })
    renderAt('/meters')

    await userEvent.click(await screen.findByRole('link', { name: 'M-109' }))

    expect(currentPath().textContent).toBe('/meters/M-109')
    expect(await screen.findByText('Meter not found')).toBeInTheDocument()
    // The shell keeps the Meters entry current on a detail page.
    expect(screen.getByRole('link', { name: 'Meters' })).toHaveAttribute('aria-current', 'page')
  })
})

describe('anomalies routes', () => {
  it('opens the anomalies list from the sidebar and marks it as current', async () => {
    window.localStorage.setItem('energy.token', 'tok')
    renderAt('/')
    await screen.findByText('4 anomalies detected · 2 require priority attention')

    await userEvent.click(screen.getByRole('link', { name: 'AI Anomalies' }))

    expect(await screen.findByRole('heading', { level: 1, name: 'AI Anomalies' })).toBeInTheDocument()
    expect(currentPath().textContent).toBe('/anomalies')
    expect(screen.getByRole('link', { name: 'AI Anomalies' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Dashboard' })).not.toHaveAttribute('aria-current')
  })

  it('opens an investigation from its row and keeps AI Anomalies current', async () => {
    window.localStorage.setItem('energy.token', 'tok')
    renderAt('/anomalies')

    await userEvent.click(await screen.findByRole('link', { name: 'Investigate M-112' }))

    expect(currentPath().textContent).toBe('/anomalies/an-2')
    expect(await screen.findByRole('heading', { level: 1, name: /M-112/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'AI Anomalies', current: 'page' })).toBeInTheDocument()
  })
})
