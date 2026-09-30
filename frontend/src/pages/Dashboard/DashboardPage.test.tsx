import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { setToken } from '../../api/token'
import type { DashboardSummary } from '../../api/types'
import { AppProviders } from '../../app/providers'
import { createTestQueryClient, jsonResponse } from '../../test/utils'
import { DashboardPage } from './DashboardPage'

const fetchMock = vi.fn<typeof fetch>()

const analyzed: DashboardSummary = {
  meters_count: 12,
  total_kwh: 12345.678,
  anomalies_count: 4,
  high_priority_count: 2,
  aggregate_confidence: 0.914,
  last_analysis: { id: 'a-1', status: 'done', finished_at: '2026-09-30T10:15:00Z' },
  summary_text: '4 anomalies detected · 2 require priority attention',
}

const pristine: DashboardSummary = {
  meters_count: 12,
  total_kwh: 9876.5,
  anomalies_count: 0,
  high_priority_count: 0,
  aggregate_confidence: null,
  last_analysis: null,
  summary_text: null,
}

function kpi(name: string) {
  return screen.getByRole('group', { name })
}

function renderPage() {
  render(
    <AppProviders queryClient={createTestQueryClient()}>
      <DashboardPage />
    </AppProviders>,
  )
}

/** Routes the mocked fetch by "METHOD path"; a list answers successive calls in order (the last one repeats; an Error rejects like a network failure). */
function mockApi(routes: Record<string, Array<Response | Error> | (() => Promise<Response>)>) {
  const counters: Record<string, number> = {}
  fetchMock.mockImplementation(async (input, init) => {
    const key = `${init?.method ?? 'GET'} ${String(input)}`
    const route = routes[key]
    if (!route) throw new Error(`unexpected request ${key}`)
    if (typeof route === 'function') return route()
    const i = counters[key] ?? 0
    counters[key] = i + 1
    const answer = route[Math.min(i, route.length - 1)]!
    if (answer instanceof Error) throw answer
    return answer.clone()
  })
}

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  setToken('tok')
})

afterEach(() => vi.unstubAllGlobals())

describe('DashboardPage after an analysis', () => {
  it('shows every KPI from the summary', async () => {
    mockApi({ 'GET /api/dashboard/summary': [jsonResponse(200, analyzed)] })
    renderPage()

    expect(await screen.findByText('12,345.7 kWh')).toBeInTheDocument()
    expect(kpi('Meters')).toHaveTextContent('12')
    expect(kpi('Anomalies')).toHaveTextContent('4')
    expect(kpi('High priority')).toHaveTextContent('2')
    expect(kpi('Aggregate confidence')).toHaveTextContent('91%')
    expect(kpi('Last analysis')).toHaveTextContent('30 Sep 2026, 10:15 UTC')
  })

  it('shows the summary text', async () => {
    mockApi({ 'GET /api/dashboard/summary': [jsonResponse(200, analyzed)] })
    renderPage()

    expect(await screen.findByText('4 anomalies detected · 2 require priority attention')).toBeInTheDocument()
    expect(screen.queryByText(/No analysis yet/)).not.toBeInTheDocument()
  })
})

describe('DashboardPage before any analysis', () => {
  it('keeps the fleet KPIs and invites running the analysis', async () => {
    mockApi({ 'GET /api/dashboard/summary': [jsonResponse(200, pristine)] })
    renderPage()

    expect(await screen.findByText('9,876.5 kWh')).toBeInTheDocument()
    expect(kpi('Meters')).toHaveTextContent('12')
    expect(kpi('Anomalies')).toHaveTextContent('—')
    expect(kpi('High priority')).toHaveTextContent('—')
    expect(kpi('Aggregate confidence')).toHaveTextContent('—')
    expect(kpi('Last analysis')).toHaveTextContent('Never')
    expect(screen.getByText('No analysis yet')).toBeInTheDocument()
    expect(screen.getByText(/across 12 meters/)).toBeInTheDocument()
  })
})

describe('DashboardPage loading and error states', () => {
  it('shows a loading indicator before the data arrives', async () => {
    mockApi({ 'GET /api/dashboard/summary': () => new Promise<Response>(() => undefined) })
    renderPage()

    expect(screen.getByRole('status', { name: 'Loading dashboard' })).toBeInTheDocument()
    expect(screen.queryByRole('group', { name: 'Meters' })).not.toBeInTheDocument()
  })

  it('shows the error with Retry and recovers when the retry succeeds', async () => {
    mockApi({ 'GET /api/dashboard/summary': [new TypeError('Failed to fetch'), jsonResponse(200, analyzed)] })
    renderPage()

    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot reach the server')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))

    expect(await screen.findByText('12,345.7 kWh')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})

describe('Run AI Analysis', () => {
  it('starts the analysis and refreshes the dashboard', async () => {
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, pristine), jsonResponse(200, analyzed)],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' })],
    })
    renderPage()
    await screen.findByText('No analysis yet')

    await userEvent.click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    expect(await screen.findByText('4 anomalies detected · 2 require priority attention')).toBeInTheDocument()
    expect(fetchMock.mock.calls.filter(([url, init]) => url === '/api/ai/analyze' && init?.method === 'POST')).toHaveLength(1)
  })

  it('disables the trigger while the request is in flight', async () => {
    let release: (r: Response) => void = () => undefined
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, pristine)],
      'POST /api/ai/analyze': () => new Promise<Response>((resolve) => (release = resolve)),
    })
    renderPage()
    await screen.findByText('No analysis yet')

    await userEvent.click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    expect(screen.getByRole('button', { name: /Run AI Analysis/ })).toBeDisabled()
    await act(async () => release(jsonResponse(202, { id: 'a-1' })))
    await vi.waitFor(() => expect(screen.getByRole('button', { name: /Run AI Analysis/ })).toBeEnabled())
  })

  it('shows an error when the analysis cannot start', async () => {
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, pristine)],
      'POST /api/ai/analyze': [jsonResponse(500, { error: { code: 'internal', message: 'analysis unavailable' } })],
    })
    renderPage()
    await screen.findByText('No analysis yet')

    await userEvent.click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not start the analysis: analysis unavailable')
  })
})
