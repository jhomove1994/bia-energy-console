import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactElement } from 'react'
import { cloneElement } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { setToken } from '../../api/token'
import type { Analysis, MeterDetail, MeterEvent, ReadingPoint } from '../../api/types'
import { AppProviders } from '../../app/providers'
import { routerFuture } from '../../app/router'
import { makeAnomaly } from '../../test/fixtures'
import { createTestQueryClient, jsonResponse } from '../../test/utils'
import { MeterDetailPage } from './MeterDetailPage'

vi.mock('recharts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('recharts')>()
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: ReactElement<{ width?: number; height?: number }> }) =>
      cloneElement(children, { width: 600, height: 200 }),
  }
})

const fetchMock = vi.fn<typeof fetch>()

const meter: MeterDetail = { meter_id: 'M-109', status: 'Critical', last_reading_at: null, total_kwh: 2180, variation_pct: 103.7, anomaly_count: 1 }
const readings: ReadingPoint[] = Array.from({ length: 48 }, (_, i) => ({
  timestamp: new Date(Date.UTC(2026, 8, 13, i)).toISOString(),
  consumption_kwh: 45,
  voltage_v: 230,
  current_a: 5,
  power_factor: 0.95,
  status: 'OK',
  baseline_kwh: 22,
  baseline_voltage_v: 230,
  baseline_current_a: 5,
  baseline_power_factor: 0.95,
}))
const events: MeterEvent[] = [{ id: 1, timestamp: '2026-09-12T14:00:00Z', type: 'UNKNOWN', description: 'Unexplained rise' }]

const runningAnalysis: Analysis = {
  id: 'a-1',
  status: 'running',
  stage: 'baseline',
  stage_index: 2,
  stage_total: 7,
  progress: 14,
  created_at: '2026-09-30T10:00:00Z',
  started_at: '2026-09-30T10:00:01Z',
  finished_at: null,
  error: '',
  summary: null,
}

/** Answers by "METHOD path"; lists answer successive calls in order (the last repeats). */
function mockApi(routes: Record<string, Array<Response | Error>>) {
  const counters: Record<string, number> = {}
  fetchMock.mockImplementation(async (input, init) => {
    const key = `${init?.method ?? 'GET'} ${String(input)}`
    // No analysis in flight unless a test says otherwise.
    const route = routes[key] ?? (key === 'GET /api/ai/analysis/latest' ? [jsonResponse(204)] : undefined)
    if (!route) throw new Error(`unexpected request ${key}`)
    const i = counters[key] ?? 0
    counters[key] = i + 1
    const answer = route[Math.min(i, route.length - 1)]!
    if (answer instanceof Error) throw answer
    return answer.clone()
  })
}

const happy = () => ({
  'GET /api/meters/M-109': [jsonResponse(200, meter)],
  'GET /api/meters/M-109/readings': [jsonResponse(200, { meter_id: 'M-109', points: readings })],
  'GET /api/meters/M-109/events': [jsonResponse(200, { items: events })],
  'GET /api/anomalies?meter_id=M-109': [jsonResponse(200, { analysis_id: 'a-1', summary_text: 'x', items: [makeAnomaly()] })],
})

const calls = (key: string) => fetchMock.mock.calls.filter(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key).length

function renderAt(path: string) {
  render(
    <AppProviders queryClient={createTestQueryClient()}>
      <MemoryRouter initialEntries={[path]} future={routerFuture}>
        <Routes>
          <Route path="/meters/:meterId" element={<MeterDetailPage />} />
        </Routes>
      </MemoryRouter>
    </AppProviders>,
  )
}

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  setToken('tok')
})

afterEach(() => vi.unstubAllGlobals())

describe('MeterDetailPage', () => {
  it('shows the meter, its four charts and its events', async () => {
    mockApi(happy())
    renderAt('/meters/M-109')

    expect(await screen.findByRole('group', { name: 'Total consumption' })).toHaveTextContent('2,180.0 kWh')
    expect(screen.getByRole('heading', { level: 1, name: /M-109/ })).toBeInTheDocument()
    expect(await screen.findByText('Unexplained rise')).toBeInTheDocument()
    expect(await screen.findAllByRole('img')).toHaveLength(4)
    expect(screen.getByRole('img', { name: /Consumption \(kWh\).*baseline 22\.0 kWh \(\+104\.5%\)/ })).toBeInTheDocument()
  })

  it('shows the meter anomalies and highlights their window on the charts', async () => {
    mockApi(happy())
    renderAt('/meters/M-109')

    const section = await screen.findByRole('region', { name: 'Anomalies for this meter' })
    expect(await within(section).findByRole('link', { name: 'Investigate M-109' })).toHaveAttribute('href', '/anomalies/an-1')
    expect(screen.getByRole('link', { name: 'View findings' })).toHaveAttribute('href', '/anomalies?meter_id=M-109')
    for (const chart of await screen.findAllByRole('img')) expect(chart).toHaveAccessibleName(/anomaly window highlighted: Level shift up, High severity/)
  })

  it('keeps the rest of the page when only the anomalies fail', async () => {
    mockApi({ ...happy(), 'GET /api/anomalies?meter_id=M-109': [jsonResponse(500, { error: { code: 'internal', message: 'anomalies down' } })] })
    renderAt('/meters/M-109')

    expect(await screen.findByRole('alert')).toHaveTextContent('anomalies down')
    expect(screen.getByRole('group', { name: 'Total consumption' })).toBeInTheDocument()
    expect(await screen.findAllByRole('img')).toHaveLength(4)
  })

  it('fetches the meter, readings and events in parallel', async () => {
    mockApi({
      'GET /api/meters/M-109': [jsonResponse(200, meter)],
      'GET /api/meters/M-109/readings': [jsonResponse(200, { meter_id: 'M-109', points: readings })],
      'GET /api/meters/M-109/events': [jsonResponse(200, { items: events })],
    })
    renderAt('/meters/M-109')
    // All three requests go out before any answer is awaited by the UI.
    await waitFor(() => expect(fetchMock.mock.calls.filter(([url]) => String(url).startsWith('/api/meters') || String(url).startsWith('/api/anomalies'))).toHaveLength(4))
  })

  it('shows not-found for an unknown meter', async () => {
    mockApi({
      'GET /api/meters/M-999': [jsonResponse(404, { error: { code: 'not_found', message: 'meter not found' } })],
      'GET /api/meters/M-999/readings': [jsonResponse(404, { error: { code: 'not_found', message: 'meter not found' } })],
      'GET /api/meters/M-999/events': [jsonResponse(404, { error: { code: 'not_found', message: 'meter not found' } })],
    })
    renderAt('/meters/M-999')
    expect(await screen.findByText('Meter not found')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to meters' })).toBeInTheDocument()
  })

  it('shows the error with Retry and recovers', async () => {
    mockApi({ ...happy(), 'GET /api/meters/M-109': [new TypeError('Failed to fetch'), jsonResponse(200, meter)] })
    renderAt('/meters/M-109')
    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot reach the server')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByRole('group', { name: 'Total consumption' })).toBeInTheDocument()
  })

  it('keeps the meter visible when only the readings fail', async () => {
    mockApi({ ...happy(), 'GET /api/meters/M-109/readings': [jsonResponse(500, { error: { code: 'internal', message: 'readings down' } })] })
    renderAt('/meters/M-109')
    expect(await screen.findByRole('group', { name: 'Total consumption' })).toBeInTheDocument()
    expect(await screen.findByRole('alert')).toHaveTextContent('readings down')
  })

  it('runs the analysis, shows its progress and refreshes the meter when it finishes', async () => {
    mockApi({
      ...happy(),
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' })],
      'GET /api/ai/analysis/a-1': [
        jsonResponse(200, runningAnalysis),
        jsonResponse(200, { ...runningAnalysis, status: 'done', stage: 'recommendation', stage_index: 7, progress: 100, summary: { total: 1, priority_attention: 1, meters_analyzed: 12, by_type: {}, by_severity: {}, avg_confidence: 0.9, text: '1 anomaly detected · 1 requires priority attention' } }),
      ],
    })
    renderAt('/meters/M-109')
    await screen.findByRole('group', { name: 'Total consumption' })
    expect(calls('GET /api/meters/M-109')).toBe(1)

    await userEvent.click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    expect(await screen.findByRole('progressbar', { name: 'Analysis progress' })).toHaveAttribute('aria-valuenow', '14')
    expect(await screen.findByText('1 anomaly detected · 1 requires priority attention')).toBeInTheDocument()
    expect(calls('POST /api/ai/analyze')).toBe(1)
    await waitFor(() => expect(calls('GET /api/meters/M-109')).toBe(2))
  })

  it('resumes a run in flight when the page opens', async () => {
    mockApi({
      ...happy(),
      'GET /api/ai/analysis/latest': [jsonResponse(200, runningAnalysis)],
      'GET /api/ai/analysis/a-1': [jsonResponse(200, runningAnalysis)],
    })
    renderAt('/meters/M-109')

    expect(await screen.findByRole('progressbar', { name: 'Analysis progress' })).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: /Analyzing/ })).toBeDisabled()
    expect(calls('POST /api/ai/analyze')).toBe(0)
  })
})
