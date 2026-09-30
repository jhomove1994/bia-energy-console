import { Link } from 'react-router-dom'
import type { Anomaly, ReadingPoint } from '../../api/types'
import { EmptyState } from '../../components/atoms/EmptyState'
import { ErrorState } from '../../components/atoms/ErrorState'
import { AlertIcon } from '../../components/atoms/Icon'
import { Skeleton } from '../../components/atoms/Skeleton'
import { ConfidenceMeter } from '../../components/molecules/ConfidenceMeter'
import { SeverityBadge } from '../../components/molecules/SeverityBadge'
import { TypeBadge } from '../../components/molecules/TypeBadge'
import { ActionCard } from '../../components/organisms/ActionCard'
import { ChartGrid } from '../../components/organisms/ChartGrid'
import { EvidenceTable } from '../../components/organisms/EvidenceTable'
import { RelatedEventCard, explainsChange } from '../../components/organisms/RelatedEventCard'
import { zoomRange } from '../../lib/chartData'
import { formatWindow } from '../../lib/format'
import type { SectionState } from '../../lib/section'

export interface InvestigationViewProps {
  state: 'loading' | 'error' | 'not-found' | 'ready'
  anomaly?: Anomaly
  errorMessage?: string
  /** Readings of the anomaly's meter, for the window charts. */
  readings: SectionState<ReadingPoint[]>
  /** Refetches whatever failed. */
  onRetry: () => void
}

const linkClass = 'text-sm font-medium text-primary-strong hover:underline'

function Loading({ label, className }: { label: string; className: string }) {
  return (
    <div role="status" aria-label={label} className={`rounded-card border border-border bg-surface p-5 shadow-card ${className}`}>
      <Skeleton className="h-full min-h-32 w-full" />
    </div>
  )
}

function Charts({ anomaly, readings, onRetry }: { anomaly: Anomaly; readings: SectionState<ReadingPoint[]>; onRetry: () => void }) {
  if (readings.status === 'loading') return <Loading label="Loading charts" className="h-72" />
  if (readings.status === 'error') return <ErrorState title="Could not load the readings" message={readings.message} onRetry={onRetry} />
  const windows = [{ start: anomaly.window_start, end: anomaly.window_end, severity: anomaly.severity, label: anomaly.anomaly }]
  // The related event is a marker only when it explains the change; an UNKNOWN entry is not an explanation.
  const events = explainsChange(anomaly.related_event) ? [anomaly.related_event] : []
  return <ChartGrid readings={readings.data} events={events} windows={windows} range={zoomRange(anomaly.window_start, anomaly.window_end)} />
}

export function InvestigationView({ state, anomaly, errorMessage, readings, onRetry }: InvestigationViewProps) {
  if (state === 'not-found') {
    return (
      <EmptyState
        icon={<AlertIcon width={24} height={24} />}
        title="Anomaly not found"
        description="This finding does not exist, or it belongs to an earlier analysis that a newer run replaced. Pick one from the current list."
        action={
          <Link to="/anomalies" className="inline-flex h-10 items-center rounded-lg bg-primary-strong px-4 text-sm font-semibold text-white hover:bg-primary-strong/90">
            Back to anomalies
          </Link>
        }
      />
    )
  }
  if (state === 'loading') return <Loading label="Loading anomaly" className="h-64" />
  if (state === 'error' || !anomaly) return <ErrorState title="Could not load the anomaly" message={errorMessage ?? 'Unknown error'} onRetry={onRetry} />

  return (
    <div className="flex flex-col gap-6">
      <div>
        <Link to="/anomalies" className={linkClass}>
          AI Anomalies
        </Link>
        <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-2">
          <h1 className="text-2xl font-semibold tracking-tight">
            {anomaly.meter_id} · {anomaly.anomaly}
          </h1>
          <TypeBadge type={anomaly.type} />
          <SeverityBadge severity={anomaly.severity} />
        </div>
        <div className="mt-2 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted">
          <span className="flex items-center gap-2.5">
            <ConfidenceMeter value={anomaly.confidence} />
            {`${anomaly.confidence_label} confidence`}
          </span>
          <span className="tabular">Detected {formatWindow(anomaly.window_start, anomaly.window_end)}</span>
          <Link to={`/meters/${encodeURIComponent(anomaly.meter_id)}`} className={linkClass} aria-label={`View meter ${anomaly.meter_id}`}>
            View meter
          </Link>
        </div>
      </div>

      <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
        {/* The action leads on narrow screens and sits beside the evidence on wide ones. */}
        <aside className="flex flex-col gap-4 lg:order-2">
          <ActionCard action={anomaly.recommended_action} code={anomaly.action_code} rank={anomaly.priority_rank} type={anomaly.type} />
          <RelatedEventCard event={anomaly.related_event} />
        </aside>

        <div className="flex flex-col gap-6 lg:order-1">
          <section aria-labelledby="reason-heading" className="flex flex-col gap-2">
            <h2 id="reason-heading" className="text-base font-semibold text-text">
              Why it was flagged
            </h2>
            <p className="max-w-3xl text-sm leading-relaxed text-text">{anomaly.reason}</p>
          </section>

          <section aria-labelledby="evidence-heading" className="flex flex-col gap-3">
            <h2 id="evidence-heading" className="text-base font-semibold text-text">
              Evidence
            </h2>
            <EvidenceTable items={anomaly.evidence} />
          </section>

          <section aria-labelledby="window-heading" className="flex flex-col gap-3">
            <h2 id="window-heading" className="text-base font-semibold text-text">
              Affected window
            </h2>
            <Charts anomaly={anomaly} readings={readings} onRetry={onRetry} />
          </section>
        </div>
      </div>
    </div>
  )
}
