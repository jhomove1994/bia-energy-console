import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactElement } from 'react'
import { cloneElement } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { setToken } from '../../api/token'
import type { ReadingPoint } from '../../api/types'
import { AppProviders } from '../../app/providers'
import { routerFuture } from '../../app/router'
import { makeAnomaly } from '../../test/fixtures'
import { createTestQueryClient, jsonResponse } from '../../test/utils'
import { InvestigationPage } from './InvestigationPage'

vi.mock('recharts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('recharts')>()
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: ReactElement<{ width?: number; height?: number }> }) =>
      cloneElement(children, { width: 600, height: 200 }),
  }
})

const fetchMock = vi.fn<typeof fetch>()

const readings: ReadingPoint[] = Array.from({ length: 240 }, (_, i) => ({
  timestamp: new Date(Date.UTC(2026, 8, 8, i)).toISOString(),
  consumption_kwh: 44,
  voltage_v: 220,
  current_a: 200,
  power_factor: 0.94,
  status: 'OK',
  baseline_kwh: 44,
  baseline_voltage_v: 220,
  baseline_current_a: 200,
  baseline_power_factor: 0.94,
}))

function mockApi(routes: Record<string, Array<Response | Error>>) {
  const counters: Record<string, number> = {}
  fetchMock.mockImplementation(async (input, init) => {
    const key = `${init?.method ?? 'GET'} ${String(input)}`
    const route = routes[key]
    if (!route) throw new Error(`unexpected request ${key}`)
    const i = counters[key] ?? 0
    counters[key] = i + 1
    const answer = route[Math.min(i, route.length - 1)]!
    if (answer instanceof Error) throw answer
    return answer.clone()
  })
}

const calls = (key: string) => fetchMock.mock.calls.filter(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key).length

function renderAt(path: string) {
  render(
    <AppProviders queryClient={createTestQueryClient()}>
      <MemoryRouter initialEntries={[path]} future={routerFuture}>
        <Routes>
          <Route path="/anomalies/:id" element={<InvestigationPage />} />
        </Routes>
      </MemoryRouter>
    </AppProviders>,
  )
}

const readingsOk = { 'GET /api/meters/M-109/readings': [jsonResponse(200, { meter_id: 'M-109', points: readings })] }

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  setToken('tok')
})

afterEach(() => vi.unstubAllGlobals())

describe('InvestigationPage', () => {
  it('loads the anomaly, then the readings of its meter, and shows the finding with its charts', async () => {
    mockApi({ 'GET /api/anomalies/an-1': [jsonResponse(200, makeAnomaly())], ...readingsOk })
    renderAt('/anomalies/an-1')

    expect(await screen.findByRole('heading', { level: 1, name: /M-109/ })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Acción recomendada' })).toHaveTextContent('INSPECT_LOAD_INCREASE')
    expect(await screen.findAllByRole('img')).toHaveLength(4)
    expect(calls('GET /api/meters/M-109/readings')).toBe(1)
  })

  it('does not ask for readings before it knows the meter', async () => {
    mockApi({ 'GET /api/anomalies/an-1': [jsonResponse(200, makeAnomaly())], ...readingsOk })
    renderAt('/anomalies/an-1')

    expect(calls('GET /api/meters//readings')).toBe(0)
    await screen.findByRole('heading', { level: 1, name: /M-109/ })
  })

  it('shows not-found for an unknown anomaly', async () => {
    mockApi({ 'GET /api/anomalies/zzz': [jsonResponse(404, { error: { code: 'not_found', message: 'anomaly not found' } })] })
    renderAt('/anomalies/zzz')

    expect(await screen.findByText('Anomalía no encontrada')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver a anomalías' })).toBeInTheDocument()
  })

  it('shows the error with Retry and recovers', async () => {
    mockApi({ 'GET /api/anomalies/an-1': [new TypeError('Failed to fetch'), jsonResponse(200, makeAnomaly())], ...readingsOk })
    renderAt('/anomalies/an-1')

    expect(await screen.findByRole('alert')).toHaveTextContent('No se pudo conectar con el servidor')
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(await screen.findByRole('heading', { level: 1, name: /M-109/ })).toBeInTheDocument()
  })

  it('keeps the finding when only the readings fail, and retries them', async () => {
    mockApi({
      'GET /api/anomalies/an-1': [jsonResponse(200, makeAnomaly())],
      'GET /api/meters/M-109/readings': [jsonResponse(500, { error: { code: 'internal', message: 'readings down' } }), jsonResponse(200, { meter_id: 'M-109', points: readings })],
    })
    renderAt('/anomalies/an-1')

    expect(await screen.findByRole('alert')).toHaveTextContent('Ocurrió un error interno en el servidor')
    expect(screen.getByRole('region', { name: 'Acción recomendada' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))
    await waitFor(() => expect(screen.getAllByRole('img')).toHaveLength(4))
  })
})
