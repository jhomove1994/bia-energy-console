import { render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import { cloneElement } from 'react'
import { METRICS, toChartSeries, toReferenceAreas } from '../../lib/chartData'
import type { ReadingPoint } from '../../api/types'
import { ChartTooltipContent, MetricChart } from './MetricChart'

// jsdom has no layout, so the responsive wrapper would render nothing; give it a fixed size.
vi.mock('recharts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('recharts')>()
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: ReactElement<{ width?: number; height?: number }> }) =>
      cloneElement(children, { width: 600, height: 200 }),
  }
})

const kwh = METRICS[0]!
const pf = METRICS[3]!

function reading(hour: number, kwhValue: number, baseline: number): ReadingPoint {
  return {
    timestamp: `2026-09-12T${String(hour).padStart(2, '0')}:00:00Z`,
    consumption_kwh: kwhValue,
    voltage_v: 230,
    current_a: 5,
    power_factor: 0.95,
    status: 'OK',
    baseline_kwh: baseline,
    baseline_voltage_v: 0,
    baseline_current_a: 0,
    baseline_power_factor: 0,
  }
}

const series = toChartSeries([reading(0, 40, 40), reading(1, 60, 40), reading(2, 92.8, 44.1)], kwh)

describe('MetricChart', () => {
  it('titles the chart with its unit and describes it with the key figures', () => {
    render(<MetricChart metric={kwh} series={series} />)
    expect(screen.getByText('Consumption')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /Consumption \(kWh\).*latest 92\.8 kWh.*baseline 44\.1 kWh.*\+110\.4%.*3 readings/ })).toBeInTheDocument()
  })

  it('explains the two line styles when a baseline exists', () => {
    render(<MetricChart metric={kwh} series={series} />)
    expect(screen.getByText('Observed')).toBeInTheDocument()
    expect(screen.getByText('Baseline')).toBeInTheDocument()
  })

  it('leaves the baseline out of the legend and the summary when the meter has none', () => {
    render(<MetricChart metric={pf} series={toChartSeries([reading(0, 1, 1)], pf)} />)
    expect(screen.queryByText('Baseline')).not.toBeInTheDocument()
    expect(screen.getByRole('img', { name: /Power factor \(PF\).*latest 0\.95 PF/ })).not.toHaveAccessibleName(/baseline/)
  })

  it('names each highlighted anomaly window in the text alternative and the legend', () => {
    const areas = toReferenceAreas([{ start: '2026-09-12T14:00:00Z', end: '2026-09-13T02:00:00Z', severity: 'HIGH', label: 'Level shift up' }])
    render(<MetricChart metric={kwh} series={series} areas={areas} />)
    expect(screen.getByRole('img', { name: /1 anomaly window highlighted: Level shift up, High severity, 12 Sep 2026, 14:00 UTC – 13 Sep 2026, 02:00 UTC/ })).toBeInTheDocument()
    expect(screen.getByText('Anomaly window')).toBeInTheDocument()
  })

  it('counts several windows and falls back to the severity when a window has no label', () => {
    const areas = toReferenceAreas([
      { start: '2026-09-12T14:00:00Z', end: '2026-09-12T20:00:00Z', severity: 'LOW' },
      { start: '2026-09-13T00:00:00Z', end: '2026-09-13T05:00:00Z', severity: 'MEDIUM' },
    ])
    render(<MetricChart metric={kwh} series={series} areas={areas} />)
    expect(screen.getByRole('img', { name: /2 anomaly windows highlighted: Low severity, 12 Sep 2026, 14:00–20:00 UTC; Medium severity, 13 Sep 2026, 00:00–05:00 UTC/ })).toBeInTheDocument()
  })

  it('shows no window text or legend entry without windows', () => {
    render(<MetricChart metric={kwh} series={series} />)
    expect(screen.queryByText('Anomaly window')).not.toBeInTheDocument()
    expect(screen.getByRole('img')).not.toHaveAccessibleName(/anomaly window/)
  })

  it('says so when there are no readings', () => {
    render(<MetricChart metric={kwh} series={[]} />)
    expect(screen.getByText('No readings to chart yet.')).toBeInTheDocument()
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })
})

describe('ChartTooltipContent', () => {
  const point = series[2]!

  it('shows the UTC timestamp, the value, the baseline and the delta', () => {
    render(<ChartTooltipContent metric={kwh} point={point} />)
    expect(screen.getByText('12 Sep 2026, 02:00 UTC')).toBeInTheDocument()
    expect(screen.getByText('92.8 kWh')).toBeInTheDocument()
    expect(screen.getByText('44.1 kWh')).toBeInTheDocument()
    expect(screen.getByText('+110.4%')).toBeInTheDocument()
  })

  it('omits baseline and delta without a baseline', () => {
    render(<ChartTooltipContent metric={kwh} point={{ ...point, baseline: null }} />)
    expect(screen.getByText('92.8 kWh')).toBeInTheDocument()
    expect(screen.queryByText('Baseline')).not.toBeInTheDocument()
    expect(screen.queryByText('Delta')).not.toBeInTheDocument()
  })

  it('renders nothing without a hovered point', () => {
    const { container } = render(<ChartTooltipContent metric={kwh} point={undefined} />)
    expect(container).toBeEmptyDOMElement()
  })
})
