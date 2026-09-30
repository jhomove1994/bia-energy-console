import type { Analysis, DashboardSummary } from '../../api/types'
import { EmptyState } from '../../components/atoms/EmptyState'
import { ErrorState } from '../../components/atoms/ErrorState'
import { ActivityIcon, AlertIcon, ClockIcon, GaugeIcon, SparklesIcon, TargetIcon } from '../../components/atoms/Icon'
import { Skeleton } from '../../components/atoms/Skeleton'
import { ConfidenceMeter } from '../../components/molecules/ConfidenceMeter'
import { KpiCard } from '../../components/molecules/KpiCard'
import { RunAnalysisButton } from '../../components/molecules/RunAnalysisButton'
import { RunAnalysisFeedback } from '../../components/organisms/RunAnalysisFeedback'
import { formatDateTime, formatKwh } from '../../lib/format'

export interface DashboardViewProps {
  state: 'loading' | 'error' | 'ready'
  summary?: DashboardSummary
  errorMessage?: string
  onRetry: () => void
  onRun: () => void
  /** The start request is in flight. */
  starting: boolean
  /** A run is starting or in flight: the trigger stays disabled. */
  busy: boolean
  /** The run started or resumed on this page, with its live progress. */
  analysis?: Analysis
  /** Why the last start attempt failed, if it did. */
  runError: string | null
}

const NOT_ANALYZED = <span className="text-muted">—</span>

function LoadingSkeleton() {
  return (
    <div role="status" aria-label="Cargando dashboard" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
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

function RunButton({ onRun, starting, busy, analysis }: Pick<DashboardViewProps, 'onRun' | 'starting' | 'busy' | 'analysis'>) {
  return <RunAnalysisButton onRun={onRun} starting={starting} busy={busy} progress={analysis?.progress} />
}

function Kpis({ summary }: { summary: DashboardSummary }) {
  const analyzed = summary.last_analysis !== null
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <KpiCard label="Medidores" value={summary.meters_count} caption="Monitoreados en la flota" icon={<GaugeIcon />} />
      <KpiCard label="Consumo total" value={formatKwh(summary.total_kwh)} caption="En todas las lecturas" icon={<ActivityIcon />} />
      <KpiCard
        label="Anomalías"
        value={analyzed ? summary.anomalies_count : NOT_ANALYZED}
        caption={analyzed ? 'Detectadas en el último análisis' : 'Aún sin analizar'}
        icon={<AlertIcon />}
      />
      <KpiCard
        label="Alta prioridad"
        value={analyzed ? summary.high_priority_count : NOT_ANALYZED}
        caption={analyzed ? 'Requieren atención prioritaria' : 'Aún sin analizar'}
        icon={<TargetIcon />}
      />
      <KpiCard label="Confianza agregada" value={<ConfidenceMeter value={summary.aggregate_confidence} size="lg" />} caption="Promedio de las anomalías detectadas" />
      <KpiCard
        label="Último análisis"
        size="md"
        value={summary.last_analysis ? formatDateTime(summary.last_analysis.finished_at) : 'Nunca'}
        caption={summary.last_analysis ? 'Completado' : 'Ejecute el análisis para comenzar'}
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

export function DashboardView({ state, summary, errorMessage, onRetry, onRun, starting, busy, analysis, runError }: DashboardViewProps) {
  const needsFirstRun = state === 'ready' && summary?.last_analysis === null
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
          <p className="mt-1 text-sm text-muted">Consumo de la flota y último análisis de anomalías con IA.</p>
        </div>
        {state === 'ready' && !needsFirstRun && <RunButton onRun={onRun} starting={starting} busy={busy} analysis={analysis} />}
      </div>

      <RunAnalysisFeedback analysis={analysis} runError={runError} onRetry={onRun} />

      {state === 'loading' && <LoadingSkeleton />}
      {state === 'error' && <ErrorState title="No se pudo cargar el dashboard" message={errorMessage ?? 'Error desconocido'} onRetry={onRetry} />}
      {state === 'ready' && summary && (
        <>
          {/* The finished run's panel already carries this headline. */}
          {analysis?.status !== 'done' && <SummaryBanner summary={summary} />}
          <Kpis summary={summary} />
          {needsFirstRun && (
            <EmptyState
              icon={<SparklesIcon width={24} height={24} />}
              title="Aún no hay análisis"
              description={`Ejecute el análisis con IA para detectar anomalías de consumo en ${summary.meters_count} medidores y obtener una acción recomendada para cada una.`}
              action={<RunButton onRun={onRun} starting={starting} busy={busy} analysis={analysis} />}
            />
          )}
        </>
      )}
    </div>
  )
}
