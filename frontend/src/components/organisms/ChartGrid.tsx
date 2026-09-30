import type { MeterEvent, ReadingPoint } from '../../api/types'
import { METRICS, toChartSeries, toReferenceAreas, toReferenceLines, type AnomalyWindow } from '../../lib/chartData'
import { MetricChart } from './MetricChart'

export interface ChartGridProps {
  readings: ReadingPoint[]
  events: MeterEvent[]
  /** Anomaly windows to highlight; optional until anomaly data is wired in. */
  windows?: AnomalyWindow[]
}

/** kWh (with its baseline), voltage, current and power factor on one shared time range; stacked on narrow screens. */
export function ChartGrid({ readings, events, windows = [] }: ChartGridProps) {
  const domain: [number, number] | undefined =
    readings.length > 0 ? [Date.parse(readings[0]!.timestamp), Date.parse(readings[readings.length - 1]!.timestamp)] : undefined
  const areas = toReferenceAreas(windows)
  // Event markers annotate the consumption chart only, where their effect is read.
  const lines = toReferenceLines(events)

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {METRICS.map((metric) => (
        <MetricChart
          key={metric.key}
          metric={metric}
          series={toChartSeries(readings, metric)}
          areas={areas}
          lines={metric.key === 'kwh' ? lines : []}
          domain={domain}
        />
      ))}
    </div>
  )
}
