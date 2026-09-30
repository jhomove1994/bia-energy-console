import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { setToken } from '../../api/token'
import { MemoryRouter } from 'react-router-dom'
import type { Analysis, DashboardSummary } from '../../api/types'
import { AppProviders } from '../../app/providers'
import { routerFuture } from '../../app/router'
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
      <MemoryRouter future={routerFuture}>
        <DashboardPage />
      </MemoryRouter>
    </AppProviders>,
  )
}

/** Routes the mocked fetch by "METHOD path"; a list answers successive calls in order (the last one repeats; an Error rejects like a network failure). */
function mockApi(routes: Record<string, Array<Response | Error> | (() => Promise<Response>)>) {
  const counters: Record<string, number> = {}
  fetchMock.mockImplementation(async (input, init) => {
    const key = `${init?.method ?? 'GET'} ${String(input)}`
    // No analysis in flight unless a test says otherwise.
    const route = routes[key] ?? (key === 'GET /api/ai/analysis/latest' ? [jsonResponse(204)] : undefined)
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

const analysis = (over: Partial<Analysis> = {}): Analysis => ({
  id: 'a-1',
  status: 'running',
  stage: 'detection',
  stage_index: 3,
  stage_total: 7,
  progress: 28,
  created_at: '2026-09-30T10:00:00Z',
  started_at: '2026-09-30T10:00:01Z',
  finished_at: null,
  error: '',
  summary: null,
  ...over,
})

const finished = analysis({
  status: 'done',
  stage: 'recommendation',
  stage_index: 7,
  progress: 100,
  finished_at: '2026-09-30T10:15:00Z',
  summary: { total: 4, priority_attention: 2, meters_analyzed: 12, by_type: {}, by_severity: {}, avg_confidence: 0.914, text: '4 anomalies detected · 2 require priority attention' },
})

const posts = () => fetchMock.mock.calls.filter(([url, init]) => url === '/api/ai/analyze' && init?.method === 'POST').length

describe('Run AI Analysis', () => {
  beforeEach(() => vi.useFakeTimers({ shouldAdvanceTime: true }))
  afterEach(() => vi.useRealTimers())

  const user = () => userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

  it('shows live progress through the stages, then the headline and a link to the anomalies', async () => {
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, pristine), jsonResponse(200, analyzed)],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' })],
      'GET /api/ai/analysis/a-1': [jsonResponse(200, analysis()), jsonResponse(200, analysis({ stage: 'explanation', stage_index: 6, progress: 71 })), jsonResponse(200, finished)],
    })
    renderPage()
    await screen.findByText('No analysis yet')

    await user().click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    const panel = await screen.findByRole('region', { name: 'AI analysis' })
    expect(within(panel).getByRole('status')).toHaveTextContent('Stage 3 of 7: Detection')
    await act(() => vi.advanceTimersByTimeAsync(500))
    await waitFor(() => expect(within(panel).getByRole('status')).toHaveTextContent('Stage 6 of 7: Explanation'))
    await act(() => vi.advanceTimersByTimeAsync(500))

    await waitFor(() => expect(within(panel).getByRole('link', { name: 'View anomalies' })).toHaveAttribute('href', '/anomalies'))
    expect(within(panel).getByText('4 anomalies detected · 2 require priority attention')).toBeInTheDocument()
    // The dashboard refreshed itself with the new analysis.
    await waitFor(() => expect(kpi('Anomalies')).toHaveTextContent('4'))
    expect(posts()).toBe(1)
  })

  it('disables the trigger and shows the current progress while a run is active', async () => {
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, analyzed)],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-2' })],
      'GET /api/ai/analysis/a-2': [jsonResponse(200, analysis({ id: 'a-2' }))],
    })
    renderPage()
    await screen.findByText('12,345.7 kWh')

    await user().click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    const trigger = await screen.findByRole('button', { name: /Analyzing/ })
    expect(trigger).toBeDisabled()
    expect(trigger).toHaveTextContent('28%')
    await user().click(trigger)
    expect(posts()).toBe(1)
  })

  it('disables the trigger while the request is in flight', async () => {
    let release: (r: Response) => void = () => undefined
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, pristine)],
      'POST /api/ai/analyze': () => new Promise<Response>((resolve) => (release = resolve)),
      'GET /api/ai/analysis/a-1': [jsonResponse(200, analysis())],
    })
    renderPage()
    await screen.findByText('No analysis yet')

    await user().click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    expect(screen.getByRole('button', { name: /Run AI Analysis/ })).toBeDisabled()
    await act(async () => release(jsonResponse(202, { id: 'a-1' })))
    expect(await screen.findByRole('button', { name: /Analyzing/ })).toBeDisabled()
  })

  it('shows an error when the analysis cannot start', async () => {
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, pristine)],
      'POST /api/ai/analyze': [jsonResponse(500, { error: { code: 'internal', message: 'analysis unavailable' } })],
    })
    renderPage()
    await screen.findByText('No analysis yet')

    await user().click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not start the analysis: analysis unavailable')
    expect(screen.getByRole('button', { name: /Run AI Analysis/ })).toBeEnabled()
  })

  it('shows the failure and starts a new run on Retry', async () => {
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, analyzed)],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' }), jsonResponse(202, { id: 'a-2' })],
      'GET /api/ai/analysis/a-1': [jsonResponse(200, analysis({ status: 'failed', error: 'engine crashed' }))],
      'GET /api/ai/analysis/a-2': [jsonResponse(200, analysis({ id: 'a-2' }))],
    })
    renderPage()
    await screen.findByText('12,345.7 kWh')

    await user().click(screen.getByRole('button', { name: 'Run AI Analysis' }))
    const panel = await screen.findByRole('region', { name: 'AI analysis' })
    await waitFor(() => expect(within(panel).getByRole('alert')).toHaveTextContent('engine crashed'))

    await user().click(within(panel).getByRole('button', { name: 'Retry' }))

    // The panel is replaced by the new run's, so it is looked up again.
    await waitFor(() => expect(within(screen.getByRole('region', { name: 'AI analysis' })).getByRole('status')).toHaveTextContent('Stage 3 of 7: Detection'))
    expect(posts()).toBe(2)
  })

  it('resumes a run that was already in flight when the page opened', async () => {
    mockApi({
      'GET /api/dashboard/summary': [jsonResponse(200, analyzed)],
      'GET /api/ai/analysis/latest': [jsonResponse(200, analysis({ id: 'a-9' }))],
      'GET /api/ai/analysis/a-9': [jsonResponse(200, analysis({ id: 'a-9', stage: 'events', stage_index: 5, progress: 57 }))],
    })
    renderPage()

    const panel = await screen.findByRole('region', { name: 'AI analysis' })
    await waitFor(() => expect(within(panel).getByRole('status')).toHaveTextContent('Stage 5 of 7: Events'))
    expect(screen.getByRole('button', { name: /Analyzing/ })).toBeDisabled()
    expect(posts()).toBe(0)
  })
})
