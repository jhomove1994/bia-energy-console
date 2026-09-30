import { Link } from 'react-router-dom'
import type { MeterDetail, MeterEvent, ReadingPoint } from '../../api/types'
import { Button } from '../../components/atoms/Button'
import { EmptyState } from '../../components/atoms/EmptyState'
import { ErrorState } from '../../components/atoms/ErrorState'
import { AlertIcon, ActivityIcon, BoltIcon, GaugeIcon, SparklesIcon } from '../../components/atoms/Icon'
import { Skeleton } from '../../components/atoms/Skeleton'
import { Spinner } from '../../components/atoms/Spinner'
import { KpiCard } from '../../components/molecules/KpiCard'
import { StatusPill } from '../../components/molecules/StatusPill'
import { ChartGrid } from '../../components/organisms/ChartGrid'
import { EventTimeline } from '../../components/organisms/EventTimeline'
import { latestDayTotals } from '../../lib/chartData'
import { formatDateTime, formatKwh, formatSignedPercent } from '../../lib/format'
import { variationTone } from '../../lib/status'

export type SectionState<T> = { status: 'loading' } | { status: 'error'; message: string } | { status: 'ready'; data: T }

export interface MeterDetailViewProps {
  meterId: string
  state: 'loading' | 'error' | 'not-found' | 'ready'
  meter?: MeterDetail
  errorMessage?: string
  readings: SectionState<ReadingPoint[]>
  events: SectionState<MeterEvent[]>
  /** Refetches whatever failed. */
  onRetry: () => void
  onRun: () => void
  starting: boolean
  waiting: boolean
  runError: string | null
}

const NO_DATA = <span className="text-muted">—</span>

const variationText = { high: 'text-sev-high-ink', medium: 'text-sev-medium-ink', none: 'text-text' } as const

function Loading({ label, className }: { label: string; className: string }) {
  return (
    <div role="status" aria-label={label} className={`rounded-card border border-border bg-surface p-5 shadow-card ${className}`}>
      <Skeleton className="h-full min-h-32 w-full" />
    </div>
  )
}

function Summary({ meter, readings }: { meter: MeterDetail; readings: SectionState<ReadingPoint[]> }) {
  const day = readings.status === 'ready' ? latestDayTotals(readings.data) : null
  const tone = variationTone(meter.variation_pct)
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <KpiCard label="Total consumption" value={formatKwh(meter.total_kwh)} caption={`Last reading ${formatDateTime(meter.last_reading_at)}`} icon={<ActivityIcon />} />
      <KpiCard
        label="Daily variation"
        value={meter.variation_pct === null ? NO_DATA : <span className={variationText[tone]}>{formatSignedPercent(meter.variation_pct)}</span>}
        caption={meter.variation_pct === null ? 'Not enough history for a baseline' : 'Latest day against the baseline'}
        icon={<BoltIcon />}
      />
      <KpiCard
        label="Latest 24 h"
        value={day ? formatKwh(day.observedKwh) : NO_DATA}
        caption={day?.baselineKwh != null ? `Baseline ~${formatKwh(day.baselineKwh)}` : 'No baseline for this window'}
        icon={<GaugeIcon />}
      />
      <KpiCard
        label="Open findings"
        value={meter.anomaly_count}
        caption={meter.anomaly_count === 0 ? 'None in the latest analysis' : 'From the latest analysis'}
        icon={<AlertIcon />}
      />
    </div>
  )
}

export function MeterDetailView({ meterId, state, meter, errorMessage, readings, events, onRetry, onRun, starting, waiting, runError }: MeterDetailViewProps) {
  if (state === 'not-found') {
    return (
      <EmptyState
        icon={<GaugeIcon width={24} height={24} />}
        title="Meter not found"
        description={`There is no meter with the id ${meterId}. Check the address or pick one from the list.`}
        action={
          <Link to="/meters" className="inline-flex h-10 items-center rounded-lg bg-primary-strong px-4 text-sm font-semibold text-white hover:bg-primary-strong/90">
            Back to meters
          </Link>
        }
      />
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <Link to="/meters" className="text-sm font-medium text-primary-strong hover:underline">
            Meters
          </Link>
          <div className="mt-1 flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-semibold tracking-tight">Meter {meterId}</h1>
            {meter && <StatusPill status={meter.status} />}
          </div>
        </div>
        {state === 'ready' && (
          <Button variant="accent" onClick={onRun} loading={starting}>
            <SparklesIcon width={16} height={16} />
            Run AI Analysis
          </Button>
        )}
      </div>

      {runError && (
        <p role="alert" className="rounded-lg bg-sev-high/10 px-4 py-3 text-sm font-medium text-sev-high-ink">
          Could not start the analysis: {runError}
        </p>
      )}
      {waiting && (
        <p className="flex items-center gap-2.5 rounded-lg bg-primary/10 px-4 py-3 text-sm font-medium text-primary-strong">
          <Spinner size="sm" label="Analysis running" />
          Analysis started. This meter updates as soon as it finishes.
        </p>
      )}

      {state === 'loading' && <Loading label="Loading meter" className="h-32" />}
      {state === 'error' && <ErrorState title="Could not load the meter" message={errorMessage ?? 'Unknown error'} onRetry={onRetry} />}
      {state === 'ready' && meter && (
        <>
          <Summary meter={meter} readings={readings} />

          <section aria-labelledby="charts-heading" className="flex flex-col gap-3">
            <h2 id="charts-heading" className="text-base font-semibold text-text">
              Readings against the baseline
            </h2>
            {readings.status === 'loading' && <Loading label="Loading charts" className="h-72" />}
            {readings.status === 'error' && <ErrorState title="Could not load the readings" message={readings.message} onRetry={onRetry} />}
            {readings.status === 'ready' && <ChartGrid readings={readings.data} events={events.status === 'ready' ? events.data : []} />}
          </section>

          {events.status === 'loading' && <Loading label="Loading events" className="h-24" />}
          {events.status === 'error' && <ErrorState title="Could not load the events" message={events.message} onRetry={onRetry} />}
          {events.status === 'ready' && <EventTimeline events={events.data} />}
        </>
      )}
    </div>
  )
}
