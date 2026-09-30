import type { MeterEvent, ReadingPoint } from '../../api/types'
import { METRICS, clipSeries, toChartSeries, toReferenceAreas, toReferenceLines, type AnomalyWindow, type TimeRange } from '../../lib/chartData'
import { MetricChart } from './MetricChart'

export interface ChartGridProps {
  readings: ReadingPoint[]
  events: MeterEvent[]
  /** Anomaly windows to highlight. */
  windows?: AnomalyWindow[]
  /** Zoom every chart to this range (epoch ms); defaults to the whole series. */
  range?: TimeRange
}

/** kWh (with its baseline), voltage, current and power factor on one shared time range; stacked on narrow screens. */
export function ChartGrid({ readings, events, windows = [], range }: ChartGridProps) {
  const domain: [number, number] | undefined = range
    ? [range.from, range.to]
    : readings.length > 0
      ? [Date.parse(readings[0]!.timestamp), Date.parse(readings[readings.length - 1]!.timestamp)]
      : undefined
  const areas = toReferenceAreas(windows)
  // Event markers annotate the consumption chart only, where their effect is read.
  const lines = toReferenceLines(events)

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {METRICS.map((metric) => (
        <MetricChart
          key={metric.key}
          metric={metric}
          series={range ? clipSeries(toChartSeries(readings, metric), range) : toChartSeries(readings, metric)}
          areas={areas}
          lines={metric.key === 'kwh' ? lines : []}
          domain={domain}
        />
      ))}
    </div>
  )
}
