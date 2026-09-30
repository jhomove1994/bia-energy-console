import { CartesianGrid, ComposedChart, Line, ReferenceArea, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import {
  deltaPct,
  summarizeSeries,
  type ChartPoint,
  type MetricConfig,
  type ReferenceAreaRange,
  type ReferenceLineMarker,
  type SeriesSummary,
} from '../../lib/chartData'
import { chartTheme } from '../../lib/chartTheme'
import { formatAxisTick, formatDateTime, formatNumber, formatSignedPercent } from '../../lib/format'
import { Card } from '../atoms/Card'

export interface MetricChartProps {
  metric: MetricConfig
  series: ChartPoint[]
  /** Anomaly windows, tinted by severity. */
  areas?: ReferenceAreaRange[]
  /** Event markers. */
  lines?: ReferenceLineMarker[]
  /** Shared x range so stacked charts line up. Defaults to the series extent. */
  domain?: [number, number]
}

const withUnit = (metric: MetricConfig, value: number): string => `${formatNumber(value, metric.decimals)} ${metric.unit}`

function describeSeries(metric: MetricConfig, s: SeriesSummary): string {
  const baseline =
    s.latestBaseline !== null && s.latestDeltaPct !== null
      ? `, baseline ${withUnit(metric, s.latestBaseline)} (${formatSignedPercent(s.latestDeltaPct)})`
      : ''
  return `${metric.title} (${metric.unit}): latest ${withUnit(metric, s.latest)}${baseline}; range ${formatNumber(s.min, metric.decimals)} to ${withUnit(metric, s.max)} over ${s.count} readings.`
}

/** Hover card: UTC time, observed value, baseline and the deviation from it. */
export function ChartTooltipContent({ metric, point }: { metric: MetricConfig; point?: ChartPoint }) {
  if (!point) return null
  const delta = deltaPct(point.observed, point.baseline)
  return (
    <div className="rounded-lg border border-border bg-surface px-3 py-2 text-xs shadow-raised">
      <p className="mb-1 font-semibold text-text">{formatDateTime(new Date(point.t).toISOString())}</p>
      <dl className="tabular grid grid-cols-[auto_auto] gap-x-4 gap-y-0.5">
        <dt className="text-muted">Observed</dt>
        <dd className="text-right font-semibold text-text">{withUnit(metric, point.observed)}</dd>
        {point.baseline !== null && (
          <>
            <dt className="text-muted">Baseline</dt>
            <dd className="text-right text-text">{withUnit(metric, point.baseline)}</dd>
          </>
        )}
        {delta !== null && (
          <>
            <dt className="text-muted">Delta</dt>
            <dd className="text-right font-semibold text-text">{formatSignedPercent(delta)}</dd>
          </>
        )}
      </dl>
    </div>
  )
}

function Legend({ hasBaseline }: { hasBaseline: boolean }) {
  return (
    <ul className="flex items-center gap-4 text-xs text-muted" aria-label="Legend">
      <li className="flex items-center gap-1.5">
        <svg width="20" height="4" aria-hidden="true">
          <line x1="0" y1="2" x2="20" y2="2" stroke={chartTheme.observed} strokeWidth="2.5" />
        </svg>
        Observed
      </li>
      {hasBaseline && (
        <li className="flex items-center gap-1.5">
          <svg width="20" height="4" aria-hidden="true">
            <line x1="0" y1="2" x2="20" y2="2" stroke={chartTheme.baseline} strokeWidth="2" strokeDasharray="4 3" />
          </svg>
          Baseline
        </li>
      )}
    </ul>
  )
}

export function MetricChart({ metric, series, areas = [], lines = [], domain }: MetricChartProps) {
  const summary = summarizeSeries(series)
  const hasBaseline = series.some((p) => p.baseline !== null)
  const xDomain = domain ?? (summary ? [series[0]!.t, series[series.length - 1]!.t] : undefined)

  return (
    <Card className="flex flex-col gap-3">
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <h3 className="text-sm font-semibold text-text">{metric.title}</h3>
        <Legend hasBaseline={hasBaseline} />
      </div>

      {!summary && <p className="py-10 text-center text-sm text-muted">No readings to chart yet.</p>}
      {summary && (
        <div role="img" aria-label={describeSeries(metric, summary)} className="h-48 w-full sm:h-56">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={series} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} accessibilityLayer={false}>
              <CartesianGrid stroke={chartTheme.grid} strokeDasharray="3 3" vertical={false} />
              {areas.map((a) => (
                <ReferenceArea key={`${a.x1}-${a.x2}`} x1={a.x1} x2={a.x2} fill={chartTheme.area[a.tone]} fillOpacity={0.15} strokeOpacity={0} />
              ))}
              {lines.map((l) => (
                <ReferenceLine key={`${l.x}-${l.label}`} x={l.x} stroke={chartTheme.event} strokeDasharray="2 3" label={{ value: l.label, position: 'insideTopRight', fontSize: 11, fill: chartTheme.event }} />
              ))}
              <XAxis
                dataKey="t"
                type="number"
                scale="time"
                domain={xDomain}
                tickFormatter={formatAxisTick}
                tick={{ fontSize: 11, fill: chartTheme.axis }}
                tickLine={false}
                axisLine={{ stroke: chartTheme.grid }}
                minTickGap={32}
              />
              <YAxis
                domain={metric.key === 'kwh' ? [0, 'auto'] : ['auto', 'auto']}
                tickFormatter={(v: number) => formatNumber(v, metric.key === 'kwh' ? 0 : metric.decimals)}
                tick={{ fontSize: 11, fill: chartTheme.axis }}
                tickLine={false}
                axisLine={false}
                width={64}
                label={{ value: metric.unit, angle: -90, position: 'insideLeft', fontSize: 11, fill: chartTheme.axis, offset: 14 }}
              />
              <Tooltip
                isAnimationActive={false}
                content={({ active, payload }) => (active ? <ChartTooltipContent metric={metric} point={payload?.[0]?.payload as ChartPoint | undefined} /> : null)}
              />
              {hasBaseline && (
                <Line type="monotone" dataKey="baseline" stroke={chartTheme.baseline} strokeWidth={2} strokeDasharray="5 4" dot={false} activeDot={false} isAnimationActive={false} connectNulls={false} />
              )}
              <Line type="monotone" dataKey="observed" stroke={chartTheme.observed} strokeWidth={2.5} dot={false} activeDot={{ r: 4 }} isAnimationActive={false} />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      )}
    </Card>
  )
}
