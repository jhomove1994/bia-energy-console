import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { setToken } from '../api/token'
import type { Analysis } from '../api/types'
import { AppProviders } from '../app/providers'
import { createTestQueryClient, jsonResponse } from '../test/utils'
import { analysisRefetchInterval, useAnalysisJob } from './useAnalysisJob'

const fetchMock = vi.fn<typeof fetch>()

const analysis = (over: Partial<Analysis> = {}): Analysis => ({
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
  ...over,
})

/** Answers by "METHOD path"; a list answers successive calls in order and the last entry repeats. */
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

function setup(queryClient = createTestQueryClient()) {
  const wrapper = ({ children }: { children: ReactNode }) => <AppProviders queryClient={queryClient}>{children}</AppProviders>
  return { queryClient, ...renderHook(() => useAnalysisJob(), { wrapper }) }
}

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  setToken('tok')
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('analysisRefetchInterval', () => {
  it('polls every 500 ms while the run is pending or running', () => {
    expect(analysisRefetchInterval('pending')).toBe(500)
    expect(analysisRefetchInterval('running')).toBe(500)
  })

  it('stops on a terminal status and while there is no data', () => {
    expect(analysisRefetchInterval('done')).toBe(false)
    expect(analysisRefetchInterval('failed')).toBe(false)
    expect(analysisRefetchInterval(undefined)).toBe(false)
  })
})

describe('useAnalysisJob', () => {
  it('starts a run, then polls every 500 ms until it is done', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    mockApi({
      'GET /api/ai/analysis/latest': [jsonResponse(204)],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' })],
      'GET /api/ai/analysis/a-1': [
        jsonResponse(200, analysis({ status: 'pending', stage: '', stage_index: 0, progress: 0 })),
        jsonResponse(200, analysis()),
        jsonResponse(200, analysis({ status: 'done', stage: 'recommendation', stage_index: 7, progress: 100, finished_at: '2026-09-30T10:00:05Z' })),
      ],
    })
    const { result } = setup()
    await waitFor(() => expect(calls('GET /api/ai/analysis/latest')).toBe(1))
    expect(result.current.analysis).toBeUndefined()

    act(() => result.current.start())
    await waitFor(() => expect(result.current.analysis?.status).toBe('pending'))
    expect(result.current.active).toBe(true)

    await act(() => vi.advanceTimersByTimeAsync(500))
    await waitFor(() => expect(result.current.analysis?.status).toBe('running'))
    expect(calls('GET /api/ai/analysis/a-1')).toBe(2)

    await act(() => vi.advanceTimersByTimeAsync(500))
    await waitFor(() => expect(result.current.analysis?.status).toBe('done'))
    expect(result.current.active).toBe(false)

    // Terminal: no further polling.
    await act(() => vi.advanceTimersByTimeAsync(2000))
    expect(calls('GET /api/ai/analysis/a-1')).toBe(3)
  })

  it('refreshes dashboard, meters, meter and anomalies data once the run is done', async () => {
    mockApi({
      'GET /api/ai/analysis/latest': [jsonResponse(204)],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' })],
      'GET /api/ai/analysis/a-1': [jsonResponse(200, analysis({ status: 'done', progress: 100, stage_index: 7 }))],
    })
    const { result, queryClient } = setup()
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries')
    await waitFor(() => expect(calls('GET /api/ai/analysis/latest')).toBe(1))

    act(() => result.current.start())
    await waitFor(() => expect(result.current.analysis?.status).toBe('done'))

    const keys = invalidate.mock.calls.map(([filter]) => (filter as { queryKey: unknown[] }).queryKey[0])
    expect(keys).toEqual(expect.arrayContaining(['dashboard', 'meters', 'meter', 'anomalies', 'anomaly']))
    expect(invalidate.mock.calls.length).toBe(5)
  })

  it('does not start another run while one is active', async () => {
    mockApi({
      'GET /api/ai/analysis/latest': [jsonResponse(204)],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' })],
      'GET /api/ai/analysis/a-1': [jsonResponse(200, analysis())],
    })
    const { result } = setup()
    await waitFor(() => expect(calls('GET /api/ai/analysis/latest')).toBe(1))

    act(() => result.current.start())
    await waitFor(() => expect(result.current.active).toBe(true))
    act(() => result.current.start())

    expect(calls('POST /api/ai/analyze')).toBe(1)
  })

  it('resumes an in-flight analysis found on mount', async () => {
    mockApi({
      'GET /api/ai/analysis/latest': [jsonResponse(200, analysis({ id: 'a-7' }))],
      'GET /api/ai/analysis/a-7': [jsonResponse(200, analysis({ id: 'a-7', stage: 'detection', stage_index: 3, progress: 28 }))],
    })
    const { result } = setup()

    await waitFor(() => expect(result.current.analysis?.id).toBe('a-7'))
    expect(result.current.active).toBe(true)
    expect(calls('POST /api/ai/analyze')).toBe(0)
  })

  it('ignores a finished latest analysis', async () => {
    mockApi({ 'GET /api/ai/analysis/latest': [jsonResponse(200, analysis({ id: 'a-3', status: 'done', progress: 100, stage_index: 7 }))] })
    const { result } = setup()

    await waitFor(() => expect(calls('GET /api/ai/analysis/latest')).toBe(1))
    expect(result.current.analysis).toBeUndefined()
    expect(result.current.active).toBe(false)
  })

  it('reports a failed run and lets the user start again', async () => {
    mockApi({
      'GET /api/ai/analysis/latest': [jsonResponse(204)],
      'POST /api/ai/analyze': [jsonResponse(202, { id: 'a-1' }), jsonResponse(202, { id: 'a-2' })],
      'GET /api/ai/analysis/a-1': [jsonResponse(200, analysis({ status: 'failed', error: 'engine crashed' }))],
      'GET /api/ai/analysis/a-2': [jsonResponse(200, analysis({ id: 'a-2' }))],
    })
    const { result } = setup()
    await waitFor(() => expect(calls('GET /api/ai/analysis/latest')).toBe(1))

    act(() => result.current.start())
    await waitFor(() => expect(result.current.analysis?.status).toBe('failed'))
    expect(result.current.analysis?.error).toBe('engine crashed')
    expect(result.current.active).toBe(false)

    act(() => result.current.start())
    await waitFor(() => expect(result.current.analysis?.id).toBe('a-2'))
    expect(result.current.active).toBe(true)
  })

  it('exposes the start error when the request is rejected', async () => {
    mockApi({
      'GET /api/ai/analysis/latest': [jsonResponse(204)],
      'POST /api/ai/analyze': [jsonResponse(500, { error: { code: 'internal', message: 'analysis unavailable' } })],
    })
    const { result } = setup()
    await waitFor(() => expect(calls('GET /api/ai/analysis/latest')).toBe(1))

    act(() => result.current.start())

    await waitFor(() => expect(result.current.error?.message).toBe('analysis unavailable'))
    expect(result.current.active).toBe(false)
  })
})
