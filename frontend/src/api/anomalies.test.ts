import { setToken } from './token'
import { jsonResponse } from '../test/utils'
import { demoAnomalyList, makeAnomaly } from '../test/fixtures'
import { getAnomaly, listAnomalies } from './anomalies'

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  setToken('tok')
})

afterEach(() => vi.unstubAllGlobals())

const requested = () => String(fetchMock.mock.calls[0]![0])

describe('listAnomalies', () => {
  it('asks for the latest analysis when no filter is set', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, demoAnomalyList()))
    const list = await listAnomalies({})

    expect(requested()).toBe('/api/anomalies')
    expect(list.items).toHaveLength(4)
    expect(list.summary_text).toBe('4 anomalías detectadas · 2 requieren atención prioritaria')
  })

  it('sends every filter that is set and skips empty ones', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, demoAnomalyList([])))
    await listAnomalies({ type: 'REAL_ANOMALY', severity: '', meter_id: 'M-109' })

    expect(requested()).toBe('/api/anomalies?type=REAL_ANOMALY&meter_id=M-109')
  })
})

describe('getAnomaly', () => {
  it('fetches one anomaly by its encoded id', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, makeAnomaly({ id: 'a/b' })))
    const anomaly = await getAnomaly('a/b')

    expect(requested()).toBe('/api/anomalies/a%2Fb')
    expect(anomaly.id).toBe('a/b')
  })
})
