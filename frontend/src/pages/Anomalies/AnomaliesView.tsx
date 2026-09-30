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
import { severityLabel, typeLabel } from '../../lib/labels'

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
  { value: '', label: 'Todos los tipos' },
  { value: 'REAL_ANOMALY', label: typeLabel('REAL_ANOMALY') },
  { value: 'DATA_QUALITY', label: typeLabel('DATA_QUALITY') },
  { value: 'EXPLAINABLE_ANOMALY', label: typeLabel('EXPLAINABLE_ANOMALY') },
  { value: 'FALSE_POSITIVE', label: typeLabel('FALSE_POSITIVE') },
]

export const severityOptions: SelectOption[] = [
  { value: '', label: 'Todas las severidades' },
  { value: 'HIGH', label: severityLabel('HIGH') },
  { value: 'MEDIUM', label: severityLabel('MEDIUM') },
  { value: 'LOW', label: severityLabel('LOW') },
]

function LoadingSkeleton() {
  return (
    <div role="status" aria-label="Cargando anomalías" className="flex flex-col gap-2 rounded-card border border-border bg-surface p-4 shadow-card">
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
          <h1 className="text-2xl font-semibold tracking-tight">Anomalías IA</h1>
          <p className="mt-1 text-sm text-muted">Hallazgos del último análisis, ordenados por prioridad. Abra uno para ver la evidencia y la acción recomendada.</p>
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
        <div role="group" aria-label="Filtros" className="flex flex-wrap items-end gap-3">
          <Select id="anomaly-type" label="Tipo" className="w-56" options={typeOptions} value={filters.type} onChange={(e) => onTypeChange(e.target.value)} />
          <Select id="anomaly-severity" label="Severidad" className="w-56" options={severityOptions} value={filters.severity} onChange={(e) => onSeverityChange(e.target.value)} />
          {filters.meterId && (
            <span className="inline-flex h-10 items-center rounded-lg bg-primary/10 px-3 text-sm font-semibold text-primary-strong">Medidor {filters.meterId}</span>
          )}
          {filtered && (
            <Button variant="ghost" onClick={onClearFilters}>
              Borrar filtros
            </Button>
          )}
        </div>
      )}

      {state === 'loading' && <LoadingSkeleton />}
      {state === 'error' && <ErrorState title="No se pudieron cargar las anomalías" message={errorMessage ?? 'Error desconocido'} onRetry={onRetry} />}
      {noAnalysisYet && (
        <EmptyState
          icon={<SparklesIcon width={24} height={24} />}
          title="Aún no hay análisis"
          description="Ejecute el análisis con IA para detectar anomalías de consumo en la flota y obtener una acción recomendada para cada una."
          action={run}
        />
      )}
      {state === 'ready' && !noAnalysisYet && items.length === 0 && filtered && (
        <EmptyState
          icon={<AlertIcon width={24} height={24} />}
          title="Ninguna anomalía coincide con estos filtros"
          description="Flexibilice el filtro de tipo, severidad o medidor para ver más hallazgos."
          action={
            <Button variant="secondary" onClick={onClearFilters}>
              Borrar filtros
            </Button>
          }
        />
      )}
      {state === 'ready' && !noAnalysisYet && items.length === 0 && !filtered && (
        <EmptyState icon={<AlertIcon width={24} height={24} />} title="No se detectaron anomalías" description="El último análisis no encontró nada que requiera atención. Vuelva a ejecutarlo cuando lleguen nuevas lecturas." />
      )}
      {state === 'ready' && items.length > 0 && <AnomaliesTable anomalies={items} showMeter />}
    </div>
  )
}
