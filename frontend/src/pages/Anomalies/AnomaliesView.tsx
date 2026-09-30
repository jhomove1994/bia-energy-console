import type { Analysis, AnomalyList } from '../../api/types'
import { Button } from '../../components/atoms/Button'
import { EmptyState } from '../../components/atoms/EmptyState'
import { ErrorState } from '../../components/atoms/ErrorState'
import { AlertIcon, SparklesIcon } from '../../components/atoms/Icon'
import { Select, type SelectOption } from '../../components/atoms/Select'
import { Skeleton } from '../../components/atoms/Skeleton'
import { RunAnalysisButton } from '../../components/molecules/RunAnalysisButton'
import { AnomaliesTable } from '../../components/organisms/AnomaliesTable'
import { RunAnalysisFeedback } from '../../components/organisms/RunAnalysisFeedback'

export interface AnomalyViewFilters {
  /** Empty string means every type / severity / meter. */
  type: string
  severity: string
  meterId: string
}

export interface AnomaliesViewProps {
  state: 'loading' | 'error' | 'ready'
  list?: AnomalyList
  filters: AnomalyViewFilters
  errorMessage?: string
  onTypeChange: (type: string) => void
  onSeverityChange: (severity: string) => void
  onClearFilters: () => void
  onRetry: () => void
  onRun: () => void
  starting: boolean
  busy: boolean
  analysis?: Analysis
  runError: string | null
}

export const typeOptions: SelectOption[] = [
  { value: '', label: 'All types' },
  { value: 'REAL_ANOMALY', label: 'Real anomaly' },
  { value: 'DATA_QUALITY', label: 'Data quality' },
  { value: 'EXPLAINABLE_ANOMALY', label: 'Explainable' },
  { value: 'FALSE_POSITIVE', label: 'False positive' },
]

export const severityOptions: SelectOption[] = [
  { value: '', label: 'All severities' },
  { value: 'HIGH', label: 'High' },
  { value: 'MEDIUM', label: 'Medium' },
  { value: 'LOW', label: 'Low' },
]

function LoadingSkeleton() {
  return (
    <div role="status" aria-label="Loading anomalies" className="flex flex-col gap-2 rounded-card border border-border bg-surface p-4 shadow-card">
      {Array.from({ length: 4 }, (_, i) => (
        <Skeleton key={i} className="h-11 w-full" />
      ))}
    </div>
  )
}

export function AnomaliesView({ state, list, filters, errorMessage, onTypeChange, onSeverityChange, onClearFilters, onRetry, onRun, starting, busy, analysis, runError }: AnomaliesViewProps) {
  const filtered = filters.type !== '' || filters.severity !== '' || filters.meterId !== ''
  const noAnalysisYet = state === 'ready' && list?.analysis_id === null
  const items = list?.items ?? []
  const run = <RunAnalysisButton onRun={onRun} starting={starting} busy={busy} progress={analysis?.progress} />

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">AI Anomalies</h1>
          <p className="mt-1 text-sm text-muted">Findings of the latest analysis, ranked by priority. Open one to see the evidence and the recommended action.</p>
        </div>
        {state === 'ready' && !noAnalysisYet && run}
      </div>

      <RunAnalysisFeedback analysis={analysis} runError={runError} onRetry={onRun} />

      {state === 'ready' && list?.summary_text && analysis?.status !== 'done' && (
        <p className="flex items-center gap-3 rounded-card border border-primary/25 bg-primary/5 px-5 py-4 text-base font-semibold text-text">
          <AlertIcon aria-hidden="true" />
          {list.summary_text}
        </p>
      )}

      {state === 'ready' && !noAnalysisYet && (
        <div role="group" aria-label="Filters" className="flex flex-wrap items-end gap-3">
          <Select id="anomaly-type" label="Type" className="w-44" options={typeOptions} value={filters.type} onChange={(e) => onTypeChange(e.target.value)} />
          <Select id="anomaly-severity" label="Severity" className="w-44" options={severityOptions} value={filters.severity} onChange={(e) => onSeverityChange(e.target.value)} />
          {filters.meterId && (
            <span className="inline-flex h-10 items-center rounded-lg bg-primary/10 px-3 text-sm font-semibold text-primary-strong">Meter {filters.meterId}</span>
          )}
          {filtered && (
            <Button variant="ghost" onClick={onClearFilters}>
              Clear filters
            </Button>
          )}
        </div>
      )}

      {state === 'loading' && <LoadingSkeleton />}
      {state === 'error' && <ErrorState title="Could not load the anomalies" message={errorMessage ?? 'Unknown error'} onRetry={onRetry} />}
      {noAnalysisYet && (
        <EmptyState
          icon={<SparklesIcon width={24} height={24} />}
          title="No analysis yet"
          description="Run the AI analysis to detect consumption anomalies across the fleet and get a recommended action for each one."
          action={run}
        />
      )}
      {state === 'ready' && !noAnalysisYet && items.length === 0 && filtered && (
        <EmptyState
          icon={<AlertIcon width={24} height={24} />}
          title="No anomalies match these filters"
          description="Loosen the type, severity or meter filter to see more findings."
          action={
            <Button variant="secondary" onClick={onClearFilters}>
              Clear filters
            </Button>
          }
        />
      )}
      {state === 'ready' && !noAnalysisYet && items.length === 0 && !filtered && (
        <EmptyState icon={<AlertIcon width={24} height={24} />} title="No anomalies were detected" description="The latest analysis found nothing that needs attention. Run it again after new readings arrive." />
      )}
      {state === 'ready' && items.length > 0 && <AnomaliesTable anomalies={items} showMeter />}
    </div>
  )
}
