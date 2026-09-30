import type { DashboardSummary } from '../../api/types'
import { Button } from '../../components/atoms/Button'
import { EmptyState } from '../../components/atoms/EmptyState'
import { ErrorState } from '../../components/atoms/ErrorState'
import { ActivityIcon, AlertIcon, ClockIcon, GaugeIcon, SparklesIcon, TargetIcon } from '../../components/atoms/Icon'
import { Skeleton } from '../../components/atoms/Skeleton'
import { Spinner } from '../../components/atoms/Spinner'
import { ConfidenceMeter } from '../../components/molecules/ConfidenceMeter'
import { KpiCard } from '../../components/molecules/KpiCard'
import { formatDateTime, formatKwh } from '../../lib/format'

export interface DashboardViewProps {
  state: 'loading' | 'error' | 'ready'
  summary?: DashboardSummary
  errorMessage?: string
  onRetry: () => void
  onRun: () => void
  /** The start request is in flight. */
  starting: boolean
  /** The analysis was started and its results have not arrived yet. */
  waiting: boolean
  /** Why the last start attempt failed, if it did. */
  runError: string | null
}

const NOT_ANALYZED = '—'

function LoadingSkeleton() {
  return (
    <div role="status" aria-label="Loading dashboard" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="rounded-card border border-border bg-surface p-5 shadow-card">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="mt-4 h-8 w-32" />
          <Skeleton className="mt-3 h-3 w-40" />
        </div>
      ))}
    </div>
  )
}

function RunButton({ onRun, starting }: Pick<DashboardViewProps, 'onRun' | 'starting'>) {
  return (
    <Button variant="accent" onClick={onRun} loading={starting}>
      <SparklesIcon width={16} height={16} />
      Run AI Analysis
    </Button>
  )
}

function Kpis({ summary }: { summary: DashboardSummary }) {
  const analyzed = summary.last_analysis !== null
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <KpiCard label="Meters" value={summary.meters_count} caption="Monitored in the fleet" icon={<GaugeIcon />} />
      <KpiCard label="Total consumption" value={formatKwh(summary.total_kwh)} caption="Across all readings" icon={<ActivityIcon />} />
      <KpiCard
        label="Anomalies"
        value={analyzed ? summary.anomalies_count : NOT_ANALYZED}
        caption={analyzed ? 'Detected in the latest analysis' : 'Not analyzed yet'}
        icon={<AlertIcon />}
      />
      <KpiCard
        label="High priority"
        value={analyzed ? summary.high_priority_count : NOT_ANALYZED}
        caption={analyzed ? 'Require priority attention' : 'Not analyzed yet'}
        icon={<TargetIcon />}
      />
      <KpiCard label="Aggregate confidence" value={<ConfidenceMeter value={summary.aggregate_confidence} />} caption="Mean over detected anomalies" />
      <KpiCard
        label="Last analysis"
        size="md"
        value={summary.last_analysis ? formatDateTime(summary.last_analysis.finished_at) : 'Never'}
        caption={summary.last_analysis ? 'Completed' : 'Run the analysis to get started'}
        icon={<ClockIcon />}
      />
    </div>
  )
}

function SummaryBanner({ summary }: { summary: DashboardSummary }) {
  if (!summary.summary_text) return null
  const urgent = summary.high_priority_count > 0
  return (
    <p
      className={`flex items-center gap-3 rounded-card border px-5 py-4 text-base font-semibold ${
        urgent ? 'border-sev-high/25 bg-sev-high/5 text-sev-high-ink' : 'border-status-ok/25 bg-status-ok/5 text-status-ok-ink'
      }`}
    >
      <AlertIcon aria-hidden="true" />
      {summary.summary_text}
    </p>
  )
}

export function DashboardView({ state, summary, errorMessage, onRetry, onRun, starting, waiting, runError }: DashboardViewProps) {
  const needsFirstRun = state === 'ready' && summary?.last_analysis === null
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
          <p className="mt-1 text-sm text-muted">Fleet consumption and the latest AI anomaly analysis.</p>
        </div>
        {state === 'ready' && !needsFirstRun && <RunButton onRun={onRun} starting={starting} />}
      </div>

      {runError && (
        <p role="alert" className="rounded-lg bg-sev-high/10 px-4 py-3 text-sm font-medium text-sev-high-ink">
          Could not start the analysis: {runError}
        </p>
      )}
      {waiting && (
        <p className="flex items-center gap-2.5 rounded-lg bg-primary/10 px-4 py-3 text-sm font-medium text-primary-strong">
          <Spinner size="sm" label="Analysis running" />
          Analysis started. Results appear here as soon as it finishes.
        </p>
      )}

      {state === 'loading' && <LoadingSkeleton />}
      {state === 'error' && <ErrorState title="Could not load the dashboard" message={errorMessage ?? 'Unknown error'} onRetry={onRetry} />}
      {state === 'ready' && summary && (
        <>
          <SummaryBanner summary={summary} />
          <Kpis summary={summary} />
          {needsFirstRun && (
            <EmptyState
              icon={<SparklesIcon width={24} height={24} />}
              title="No analysis yet"
              description={`Run the AI analysis to detect consumption anomalies across ${summary.meters_count} meters and get a recommended action for each one.`}
              action={<RunButton onRun={onRun} starting={starting} />}
            />
          )}
        </>
      )}
    </div>
  )
}
