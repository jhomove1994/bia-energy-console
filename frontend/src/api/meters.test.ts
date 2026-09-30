import { getMeter, getMeterEvents, getMeterReadings, listMeters } from './meters'

const fetchMock = vi.fn<typeof fetch>()

function ok(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const requestedUrl = () => fetchMock.mock.calls[0]![0] as string

describe('meters API', () => {
  it('lists meters without a query string when no filter is set', async () => {
    fetchMock.mockResolvedValue(ok({ items: [] }))
    await listMeters({})
    expect(requestedUrl()).toBe('/api/meters')
  })

  it('sends only the filters that have a value', async () => {
    fetchMock.mockResolvedValue(ok({ items: [] }))
    await listMeters({ q: '109', status: 'Critical', sort: 'variation', order: 'desc' })
    const url = new URL(requestedUrl(), 'http://x')
    expect(url.pathname).toBe('/api/meters')
    expect(Object.fromEntries(url.searchParams)).toEqual({ q: '109', status: 'Critical', sort: 'variation', order: 'desc' })
  })

  it('skips empty filters', async () => {
    fetchMock.mockResolvedValue(ok({ items: [] }))
    await listMeters({ q: '', status: '', sort: 'kwh' })
    expect(requestedUrl()).toBe('/api/meters?sort=kwh')
  })

  it('returns the items of the list', async () => {
    const items = [{ meter_id: 'M-101', status: 'OK', last_reading_at: null, total_kwh: 1, variation_pct: null }]
    fetchMock.mockResolvedValue(ok({ items }))
    expect(await listMeters({})).toEqual(items)
  })

  it('reads a meter, its readings and its events by encoded id', async () => {
    fetchMock.mockResolvedValue(ok({ meter_id: 'M 1', points: [], items: [] }))
    await getMeter('M 1')
    expect(requestedUrl()).toBe('/api/meters/M%201')

    fetchMock.mockClear()
    fetchMock.mockResolvedValue(ok({ meter_id: 'M-1', points: [{ timestamp: 't' }] }))
    expect(await getMeterReadings('M-1')).toEqual([{ timestamp: 't' }])
    expect(requestedUrl()).toBe('/api/meters/M-1/readings')

    fetchMock.mockClear()
    fetchMock.mockResolvedValue(ok({ items: [{ id: 1 }] }))
    expect(await getMeterEvents('M-1')).toEqual([{ id: 1 }])
    expect(requestedUrl()).toBe('/api/meters/M-1/events')
  })
})
