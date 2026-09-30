import type { MeterEvent, ReadingPoint } from '../api/types'
import { severityToken, type Severity, type SeverityToken } from './status'

export type MetricKey = 'kwh' | 'voltage' | 'current' | 'pf'

export interface MetricConfig {
  key: MetricKey
  title: string
  unit: string
  observed: 'consumption_kwh' | 'voltage_v' | 'current_a' | 'power_factor'
  baseline: 'baseline_kwh' | 'baseline_voltage_v' | 'baseline_current_a' | 'baseline_power_factor'
  decimals: number
}

/** The four charts of the meter detail, in reading order. */
export const METRICS: readonly MetricConfig[] = [
  { key: 'kwh', title: 'Consumption', unit: 'kWh', observed: 'consumption_kwh', baseline: 'baseline_kwh', decimals: 1 },
  { key: 'voltage', title: 'Voltage', unit: 'V', observed: 'voltage_v', baseline: 'baseline_voltage_v', decimals: 1 },
  { key: 'current', title: 'Current', unit: 'A', observed: 'current_a', baseline: 'baseline_current_a', decimals: 2 },
  { key: 'pf', title: 'Power factor', unit: 'PF', observed: 'power_factor', baseline: 'baseline_power_factor', decimals: 2 },
]

export interface ChartPoint {
  /** Epoch milliseconds (UTC). */
  t: number
  observed: number
  /** Null when there is no usable baseline (too little history, or the API left it out). */
  baseline: number | null
}

/** Pairs each reading with its baseline for one metric. A baseline of 0 means "not built yet". */
export function toChartSeries(points: ReadingPoint[], metric: MetricConfig): ChartPoint[] {
  return points.map((p) => {
    const baseline = p[metric.baseline]
    return { t: Date.parse(p.timestamp), observed: p[metric.observed], baseline: baseline ? baseline : null }
  })
}

/** Deviation from the baseline in percent points, or null without a usable baseline. */
export function deltaPct(observed: number, baseline: number | null): number | null {
  if (!baseline) return null
  return ((observed - baseline) / baseline) * 100
}

export interface AnomalyWindow {
  start: string
  end: string
  severity: Severity
  /** Short name of the anomaly, used in the chart's text alternative. */
  label?: string
}

export interface ReferenceAreaRange {
  x1: number
  x2: number
  tone: SeverityToken
  label?: string
}

/** Anomaly windows as x ranges tinted by severity. */
export function toReferenceAreas(windows: AnomalyWindow[]): ReferenceAreaRange[] {
  return windows.map((w) => ({ x1: Date.parse(w.start), x2: Date.parse(w.end), tone: severityToken(w.severity), label: w.label }))
}

export interface ReferenceLineMarker {
  x: number
  label: string
}

/** "OPERATIONAL_CHANGE" to "Operational change". */
export function eventLabel(type: string): string {
  const text = type.replace(/_/g, ' ').toLowerCase()
  return text.charAt(0).toUpperCase() + text.slice(1)
}

export function toReferenceLines(events: MeterEvent[]): ReferenceLineMarker[] {
  return events
    .map((e) => ({ x: Date.parse(e.timestamp), label: eventLabel(e.type) }))
    .sort((a, b) => a.x - b.x)
}

export interface TimeRange {
  from: number
  to: number
}

const HOUR_MS = 3_600_000

/** A window widened by `marginHours` (24 by default) on each side, for the investigation zoom. */
export function zoomRange(start: string, end: string, marginHours = 24): TimeRange {
  return { from: Date.parse(start) - marginHours * HOUR_MS, to: Date.parse(end) + marginHours * HOUR_MS }
}

export function clipSeries(series: ChartPoint[], range: TimeRange): ChartPoint[] {
  return series.filter((p) => p.t >= range.from && p.t <= range.to)
}

export interface SeriesSummary {
  count: number
  min: number
  max: number
  latest: number
  latestBaseline: number | null
  latestDeltaPct: number | null
}

/** Key figures used as the text alternative of a chart. */
export function summarizeSeries(series: ChartPoint[]): SeriesSummary | null {
  const last = series.at(-1)
  if (!last) return null
  const values = series.map((p) => p.observed)
  return {
    count: series.length,
    min: Math.min(...values),
    max: Math.max(...values),
    latest: last.observed,
    latestBaseline: last.baseline,
    latestDeltaPct: deltaPct(last.observed, last.baseline),
  }
}

export interface DayTotals {
  observedKwh: number
  /** Null when any of the readings has no baseline. */
  baselineKwh: number | null
}

/** Totals over the latest 24 readings, or null when the series is shorter than a day. */
export function latestDayTotals(points: ReadingPoint[]): DayTotals | null {
  if (points.length < 24) return null
  const day = points.slice(-24)
  const observedKwh = day.reduce((sum, p) => sum + p.consumption_kwh, 0)
  const complete = day.every((p) => p.baseline_kwh > 0)
  return { observedKwh, baselineKwh: complete ? day.reduce((sum, p) => sum + p.baseline_kwh, 0) : null }
}
