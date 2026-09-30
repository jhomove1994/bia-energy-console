import type { MeterEvent, ReadingPoint } from '../api/types'
import {
  METRICS,
  clipSeries,
  deltaPct,
  eventLabel,
  latestDayTotals,
  summarizeSeries,
  toChartSeries,
  toReferenceAreas,
  toReferenceLines,
  zoomRange,
  type MetricKey,
} from './chartData'

function point(hour: number, overrides: Partial<ReadingPoint> = {}): ReadingPoint {
  return {
    timestamp: `2026-09-12T${String(hour).padStart(2, '0')}:00:00Z`,
    consumption_kwh: 10 + hour,
    voltage_v: 230,
    current_a: 5,
    power_factor: 0.95,
    status: 'OK',
    baseline_kwh: 9,
    baseline_voltage_v: 231,
    baseline_current_a: 4.8,
    baseline_power_factor: 0.96,
    ...overrides,
  }
}

const metric = (key: MetricKey) => METRICS.find((m) => m.key === key)!

describe('METRICS', () => {
  it('describes the four charts in order with units', () => {
    expect(METRICS.map((m) => [m.key, m.unit])).toEqual([
      ['kwh', 'kWh'],
      ['voltage', 'V'],
      ['current', 'A'],
      ['pf', 'PF'],
    ])
  })
})

describe('toChartSeries', () => {
  it('pairs each reading with its baseline per metric', () => {
    const points = [point(0), point(1, { consumption_kwh: 30, baseline_kwh: 12 })]
    expect(toChartSeries(points, metric('kwh'))).toEqual([
      { t: Date.parse('2026-09-12T00:00:00Z'), observed: 10, baseline: 9 },
      { t: Date.parse('2026-09-12T01:00:00Z'), observed: 30, baseline: 12 },
    ])
    expect(toChartSeries(points, metric('voltage'))[0]).toMatchObject({ observed: 230, baseline: 231 })
    expect(toChartSeries(points, metric('current'))[0]).toMatchObject({ observed: 5, baseline: 4.8 })
    expect(toChartSeries(points, metric('pf'))[0]).toMatchObject({ observed: 0.95, baseline: 0.96 })
  })

  it('leaves the baseline empty when the meter has too little history (0) or the API omits it', () => {
    const noHistory = point(0, { baseline_kwh: 0 })
    expect(toChartSeries([noHistory], metric('kwh'))[0]!.baseline).toBeNull()
    const { baseline_voltage_v: _omitted, ...withoutElectrical } = point(0)
    expect(toChartSeries([withoutElectrical], metric('voltage'))[0]!.baseline).toBeNull()
  })

  it('returns an empty series for no readings', () => {
    expect(toChartSeries([], metric('kwh'))).toEqual([])
  })
})

describe('deltaPct', () => {
  it('gives the deviation from the baseline in percent points', () => {
    expect(deltaPct(20, 10)).toBe(100)
    expect(deltaPct(5, 10)).toBe(-50)
  })
  it('is null without a usable baseline', () => {
    expect(deltaPct(5, null)).toBeNull()
    expect(deltaPct(5, 0)).toBeNull()
  })
})

describe('toReferenceAreas', () => {
  it('maps anomaly windows to x ranges tinted by severity', () => {
    const areas = toReferenceAreas([
      { start: '2026-09-12T14:00:00Z', end: '2026-09-13T02:00:00Z', severity: 'HIGH' },
      { start: '2026-09-10T00:00:00Z', end: '2026-09-10T05:00:00Z', severity: 'LOW' },
    ])
    expect(areas).toEqual([
      { x1: Date.parse('2026-09-12T14:00:00Z'), x2: Date.parse('2026-09-13T02:00:00Z'), tone: 'sev-high' },
      { x1: Date.parse('2026-09-10T00:00:00Z'), x2: Date.parse('2026-09-10T05:00:00Z'), tone: 'sev-low' },
    ])
  })
  it('is empty when there are no windows', () => {
    expect(toReferenceAreas([])).toEqual([])
  })
  it('keeps the label of a window so the chart can name it in text', () => {
    const [area] = toReferenceAreas([{ start: '2026-09-12T14:00:00Z', end: '2026-09-13T02:00:00Z', severity: 'MEDIUM', label: 'Level shift up' }])
    expect(area).toMatchObject({ tone: 'sev-medium', label: 'Level shift up' })
  })
})

describe('toReferenceLines', () => {
  const events: MeterEvent[] = [
    { id: 1, timestamp: '2026-09-12T14:00:00Z', type: 'UNKNOWN', description: 'x' },
    { id: 2, timestamp: '2026-09-11T08:00:00Z', type: 'OPERATIONAL_CHANGE', description: 'y' },
  ]
  it('maps events to labelled markers in time order', () => {
    expect(toReferenceLines(events)).toEqual([
      { x: Date.parse('2026-09-11T08:00:00Z'), label: 'Operational change' },
      { x: Date.parse('2026-09-12T14:00:00Z'), label: 'Unknown' },
    ])
  })
})

describe('eventLabel', () => {
  it('humanizes raw event types', () => {
    expect(eventLabel('SCHEDULED_OUTAGE')).toBe('Scheduled outage')
    expect(eventLabel('data_quality')).toBe('Data quality')
    expect(eventLabel('UNKNOWN')).toBe('Unknown')
  })
})

describe('zoomRange / clipSeries', () => {
  it('extends a window by 24 h on each side', () => {
    expect(zoomRange('2026-09-12T14:00:00Z', '2026-09-13T02:00:00Z')).toEqual({
      from: Date.parse('2026-09-11T14:00:00Z'),
      to: Date.parse('2026-09-14T02:00:00Z'),
    })
  })
  it('accepts another margin', () => {
    expect(zoomRange('2026-09-12T14:00:00Z', '2026-09-12T15:00:00Z', 2)).toEqual({
      from: Date.parse('2026-09-12T12:00:00Z'),
      to: Date.parse('2026-09-12T17:00:00Z'),
    })
  })
  it('keeps only the points inside the range, borders included', () => {
    const series = toChartSeries([point(0), point(1), point(2), point(3)], metric('kwh'))
    const clipped = clipSeries(series, { from: Date.parse('2026-09-12T01:00:00Z'), to: Date.parse('2026-09-12T02:00:00Z') })
    expect(clipped.map((p) => p.observed)).toEqual([11, 12])
  })
})

describe('summarizeSeries', () => {
  it('reports the latest value, the range and the deviation of the latest point', () => {
    const series = toChartSeries([point(0, { consumption_kwh: 8 }), point(1, { consumption_kwh: 30, baseline_kwh: 10 }), point(2, { consumption_kwh: 20, baseline_kwh: 10 })], metric('kwh'))
    expect(summarizeSeries(series)).toEqual({ count: 3, min: 8, max: 30, latest: 20, latestBaseline: 10, latestDeltaPct: 100 })
  })
  it('handles an empty series', () => {
    expect(summarizeSeries([])).toBeNull()
  })
})

describe('latestDayTotals', () => {
  const day = (baseline: number) =>
    Array.from({ length: 30 }, (_, i) => ({
      ...point(0),
      timestamp: new Date(Date.UTC(2026, 8, 12, i)).toISOString(),
      consumption_kwh: i < 6 ? 1000 : 2,
      baseline_kwh: i < 6 ? 1000 : baseline,
    }))

  it('sums the last 24 readings and their baseline', () => {
    expect(latestDayTotals(day(1))).toEqual({ observedKwh: 48, baselineKwh: 24 })
  })

  it('has no baseline total when any reading lacks one', () => {
    expect(latestDayTotals(day(0))).toEqual({ observedKwh: 48, baselineKwh: null })
  })

  it('is null for fewer than 24 readings', () => {
    expect(latestDayTotals(day(1).slice(0, 23))).toBeNull()
  })
})
