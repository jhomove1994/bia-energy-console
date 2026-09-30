import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { setToken } from '../../api/token'
import { AppProviders } from '../../app/providers'
import { routerFuture } from '../../app/router'
import { demoAnomalies, demoAnomalyList, doneAnalysis, makeAnalysis } from '../../test/fixtures'
import { createTestQueryClient, jsonResponse } from '../../test/utils'
import { AnomaliesPage } from './AnomaliesPage'

const fetchMock = vi.fn<typeof fetch>()

function LocationProbe() {
  const { pathname, search } = useLocation()
  return <output aria-label="location">{pathname + search}</output>
}

function renderAt(path: string) {
  render(
    <AppProviders queryClient={createTestQueryClient()}>
      <MemoryRouter initialEntries={[path]} future={routerFuture}>
        <AnomaliesPage />
        <LocationProbe />
      </MemoryRouter>
    </AppProviders>,
  )
}

/** Answers by "METHOD path"; lists answer successive calls in order (the last repeats). */
function mockApi(routes: Record<string, Array<Response | Error>>) {
  const counters: Record<string, number> = {}
  fetchMock.mockImplementation(async (input, init) => {
    const key = `${init?.method ?? 'GET'} ${String(input)}`
    const route = routes[key] ?? (key === 'GET /api/ai/analysis/latest' ? [jsonResponse(204)] : undefined)
    if (!route) throw new Error(`unexpected request ${key}`)
    const i = counters[key] ?? 0
    counters[key] = i + 1
    const answer = route[Math.min(i, route.length - 1)]!
    if (answer instanceof Error) throw answer
    return answer.clone()
  })
}

const calls = (key: string) => fetchMock.mock.calls.filter(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key).length
const bodyRows = () => within(screen.getByRole('table')).getAllByRole('row').slice(1)
const location = () => screen.getByLabelText('location')

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  setToken('tok')
})

afterEach(() => vi.unstubAllGlobals())

describe('AnomaliesPage', () => {
  it('lists the four demo anomalies ranked by priority with the summary', async () => {
    mockApi({ 'GET /api/anomalies': [jsonResponse(200, demoAnomalyList())] })
    renderAt('/anomalies')

    expect(await screen.findByText('4 anomalies detected · 2 require priority attention')).toBeInTheDocument()
    expect(bodyRows().map((r) => within(r).getByRole('rowheader').textContent)).toEqual(['M-109', 'M-112', 'M-104', 'M-106'])
  })

  it('applies the filters found in the URL', async () => {
    mockApi({
      'GET /api/anomalies?type=REAL_ANOMALY&severity=HIGH': [jsonResponse(200, demoAnomalyList(demoAnomalies().slice(0, 1)))],
    })
    renderAt('/anomalies?type=REAL_ANOMALY&severity=HIGH')

    expect(await screen.findByRole('table')).toBeInTheDocument()
    expect(bodyRows()).toHaveLength(1)
    expect(screen.getByLabelText('Type')).toHaveValue('REAL_ANOMALY')
    expect(screen.getByLabelText('Severity')).toHaveValue('HIGH')
  })

  it('filters through the URL and refetches', async () => {
    mockApi({
      'GET /api/anomalies': [jsonResponse(200, demoAnomalyList())],
      'GET /api/anomalies?type=DATA_QUALITY': [jsonResponse(200, demoAnomalyList(demoAnomalies().slice(1, 2)))],
      'GET /api/anomalies?type=DATA_QUALITY&severity=LOW': [jsonResponse(200, demoAnomalyList([]))],
    })
    renderAt('/anomalies')
    await screen.findByRole('table')

    await userEvent.selectOptions(screen.getByLabelText('Type'), 'DATA_QUALITY')
    await waitFor(() => expect(bodyRows()).toHaveLength(1))
    expect(location()).toHaveTextContent('/anomalies?type=DATA_QUALITY')
    expect(within(bodyRows()[0]!).getByRole('rowheader')).toHaveTextContent('M-112')

    await userEvent.selectOptions(screen.getByLabelText('Severity'), 'LOW')
    expect(await screen.findByText('No anomalies match these filters')).toBeInTheDocument()
    expect(location()).toHaveTextContent('/anomalies?type=DATA_QUALITY&severity=LOW')

    await userEvent.click(screen.getAllByRole('button', { name: 'Clear filters' })[0]!)
    await waitFor(() => expect(bodyRows()).toHaveLength(4))
    expect(location()).toHaveTextContent(/^\/anomalies$/)
  })

  it('filters by the meter named in the URL', async () => {
    mockApi({ 'GET /api/anomalies?meter_id=M-109': [jsonResponse(200, demoAnomalyList(demoAnomalies().slice(0, 1)))] })
    renderAt('/anomalies?meter_id=M-109')

    expect(await screen.findByText('Meter M-109')).toBeInTheDocument()
    expect(bodyRows()).toHaveLength(1)
  })

  it('offers to run the analysis when none has completed and shows the table once it is done', async () => {
    mockApi({
      'GET /api/anomalies': [jsonResponse(200, { analysis_id: null, summary_text: null, items: [] }), jsonResponse(200, demoAnomalyList())],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' })],
      'GET /api/ai/analysis/a-1': [jsonResponse(200, makeAnalysis()), jsonResponse(200, doneAnalysis())],
    })
    renderAt('/anomalies')
    expect(await screen.findByText('No analysis yet')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Run AI Analysis' }))

    expect(await screen.findByRole('region', { name: 'AI analysis' })).toBeInTheDocument()
    expect(await screen.findByRole('table', {}, { timeout: 3000 })).toBeInTheDocument()
    expect(bodyRows()).toHaveLength(4)
    expect(calls('POST /api/ai/analyze')).toBe(1)
  })

  it('shows the error with Retry and recovers', async () => {
    mockApi({ 'GET /api/anomalies': [new TypeError('Failed to fetch'), jsonResponse(200, demoAnomalyList())] })
    renderAt('/anomalies')

    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot reach the server')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))

    expect(await screen.findByRole('table')).toBeInTheDocument()
  })
})
