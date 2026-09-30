import { Link } from 'react-router-dom'
import type { Analysis, Anomaly, MeterDetail, MeterEvent, ReadingPoint } from '../../api/types'
import { EmptyState } from '../../components/atoms/EmptyState'
import { ErrorState } from '../../components/atoms/ErrorState'
import { AlertIcon, ActivityIcon, BoltIcon, GaugeIcon } from '../../components/atoms/Icon'
import { Skeleton } from '../../components/atoms/Skeleton'
import { KpiCard } from '../../components/molecules/KpiCard'
import { RunAnalysisButton } from '../../components/molecules/RunAnalysisButton'
import { StatusPill } from '../../components/molecules/StatusPill'
import { AnomaliesTable } from '../../components/organisms/AnomaliesTable'
import { ChartGrid } from '../../components/organisms/ChartGrid'
import { EventTimeline } from '../../components/organisms/EventTimeline'
import { RunAnalysisFeedback } from '../../components/organisms/RunAnalysisFeedback'
import { latestDayTotals } from '../../lib/chartData'
import { formatDateTime, formatKwh, formatSignedPercent } from '../../lib/format'
import { kindLabel } from '../../lib/labels'
import type { SectionState } from '../../lib/section'
import { variationTone } from '../../lib/status'

export interface MeterDetailViewProps {
  meterId: string
  state: 'loading' | 'error' | 'not-found' | 'ready'
  meter?: MeterDetail
  errorMessage?: string
  readings: SectionState<ReadingPoint[]>
  events: SectionState<MeterEvent[]>
  /** Findings of the latest analysis for this meter. */
  anomalies: SectionState<Anomaly[]>
  /** Refetches whatever failed. */
  onRetry: () => void
  onRun: () => void
  starting: boolean
  /** A run is starting or in flight: the trigger stays disabled. */
  busy: boolean
  /** The run started or resumed on this page, with its live progress. */
  analysis?: Analysis
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
      <KpiCard label="Consumo total" value={formatKwh(meter.total_kwh)} caption={`Última lectura ${formatDateTime(meter.last_reading_at)}`} icon={<ActivityIcon />} />
      <KpiCard
        label="Variación diaria"
        value={meter.variation_pct === null ? NO_DATA : <span className={variationText[tone]}>{formatSignedPercent(meter.variation_pct)}</span>}
        caption={meter.variation_pct === null ? 'No hay historial suficiente para un baseline' : 'Último día frente al baseline'}
        icon={<BoltIcon />}
      />
      <KpiCard
        label="Últimas 24 h"
        value={day ? formatKwh(day.observedKwh) : NO_DATA}
        caption={day?.baselineKwh != null ? `Baseline ~${formatKwh(day.baselineKwh)}` : 'Sin baseline para esta ventana'}
        icon={<GaugeIcon />}
      />
      <KpiCard
        label="Hallazgos abiertos"
        value={meter.anomaly_count}
        caption={
          meter.anomaly_count === 0 ? (
            'Ninguno en el último análisis'
          ) : (
            <Link to={`/anomalies?meter_id=${encodeURIComponent(meter.meter_id)}`} className="font-semibold text-primary-strong hover:underline">
              Ver hallazgos
            </Link>
          )
        }
        icon={<AlertIcon />}
      />
    </div>
  )
}

/** Anomaly windows for the charts; empty while the findings load or fail. */
function windowsOf(anomalies: SectionState<Anomaly[]>) {
  if (anomalies.status !== 'ready') return []
  return anomalies.data.map((a) => ({ start: a.window_start, end: a.window_end, severity: a.severity, label: kindLabel(a.kind, a.anomaly) }))
}

export function MeterDetailView({ meterId, state, meter, errorMessage, readings, events, anomalies, onRetry, onRun, starting, busy, analysis, runError }: MeterDetailViewProps) {
  if (state === 'not-found') {
    return (
      <EmptyState
        icon={<GaugeIcon width={24} height={24} />}
        title="Medidor no encontrado"
        description={`No existe un medidor con el ID ${meterId}. Revise la dirección o elija uno de la lista.`}
        action={
          <Link to="/meters" className="inline-flex h-10 items-center rounded-lg bg-primary-strong px-4 text-sm font-semibold text-white hover:bg-primary-strong/90">
            Volver a medidores
          </Link>
        }
      />
    )
  }

  const windows = windowsOf(anomalies)

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <Link to="/meters" className="text-sm font-medium text-primary-strong hover:underline">
            Medidores
          </Link>
          <div className="mt-1 flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-semibold tracking-tight">Medidor {meterId}</h1>
            {meter && <StatusPill status={meter.status} />}
          </div>
        </div>
        {state === 'ready' && <RunAnalysisButton onRun={onRun} starting={starting} busy={busy} progress={analysis?.progress} />}
      </div>

      <RunAnalysisFeedback analysis={analysis} runError={runError} onRetry={onRun} />

      {state === 'loading' && <Loading label="Cargando medidor" className="h-32" />}
      {state === 'error' && <ErrorState title="No se pudo cargar el medidor" message={errorMessage ?? 'Error desconocido'} onRetry={onRetry} />}
      {state === 'ready' && meter && (
        <>
          <Summary meter={meter} readings={readings} />

          <section aria-labelledby="meter-anomalies-heading" className="flex flex-col gap-3">
            <h2 id="meter-anomalies-heading" className="text-base font-semibold text-text">
              Anomalías de este medidor
            </h2>
            {anomalies.status === 'loading' && <Loading label="Cargando anomalías" className="h-24" />}
            {anomalies.status === 'error' && <ErrorState title="No se pudieron cargar las anomalías" message={anomalies.message} onRetry={onRetry} />}
            {anomalies.status === 'ready' && anomalies.data.length === 0 && (
              <p className="rounded-card border border-border bg-surface px-5 py-4 text-sm text-muted shadow-card">Sin anomalías en el último análisis para este medidor.</p>
            )}
            {anomalies.status === 'ready' && anomalies.data.length > 0 && <AnomaliesTable anomalies={anomalies.data} showMeter={false} />}
          </section>

          <section aria-labelledby="charts-heading" className="flex flex-col gap-3">
            <h2 id="charts-heading" className="text-base font-semibold text-text">
              Lecturas frente al baseline
            </h2>
            {readings.status === 'loading' && <Loading label="Cargando gráficos" className="h-72" />}
            {readings.status === 'error' && <ErrorState title="No se pudieron cargar las lecturas" message={readings.message} onRetry={onRetry} />}
            {readings.status === 'ready' && <ChartGrid readings={readings.data} events={events.status === 'ready' ? events.data : []} windows={windows} />}
          </section>

          {events.status === 'loading' && <Loading label="Cargando eventos" className="h-24" />}
          {events.status === 'error' && <ErrorState title="No se pudieron cargar los eventos" message={events.message} onRetry={onRetry} />}
          {events.status === 'ready' && <EventTimeline events={events.data} />}
        </>
      )}
    </div>
  )
}
