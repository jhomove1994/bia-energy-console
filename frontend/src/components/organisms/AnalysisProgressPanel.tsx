import { Link } from 'react-router-dom'
import type { Analysis, AnalysisStatus } from '../../api/types'
import { analysisStatusLabel } from '../../lib/labels'
import type { Tone } from '../../lib/tone'
import { Badge } from '../atoms/Badge'
import { Card } from '../atoms/Card'
import { ErrorState } from '../atoms/ErrorState'
import { ProgressBar } from '../atoms/ProgressBar'
import { STAGES, StageStepper } from '../molecules/StageStepper'

export interface AnalysisProgressPanelProps {
  analysis: Analysis
  /** Starts a new run after a failure. */
  onRetry: () => void
}

const statusTone: Record<AnalysisStatus, Tone> = { pending: 'neutral', running: 'type-expl', done: 'status-ok', failed: 'sev-high' }

/** What a screen reader hears when the run moves on: a short sentence that changes per stage, not per percent. */
function announcement(a: Analysis): string {
  if (a.status === 'done') return 'Análisis completado'
  if (a.status === 'failed') return 'El análisis falló'
  const name = STAGES[a.stage_index - 1]
  return name ? `Etapa ${a.stage_index} de ${a.stage_total}: ${name}` : 'Esperando para iniciar'
}

/** Live view of one analysis run: status, stage stepper, progress, and the outcome (headline plus link, or the error). */
export function AnalysisProgressPanel({ analysis, onRetry }: AnalysisProgressPanelProps) {
  const { status } = analysis
  return (
    <Card aria-labelledby="analysis-progress-heading" role="region" className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id="analysis-progress-heading" className="text-base font-semibold text-text">
          Análisis con IA
        </h2>
        <Badge tone={statusTone[status]} dot>
          {analysisStatusLabel(status)}
        </Badge>
      </div>

      <StageStepper stageIndex={analysis.stage_index} status={status} />

      <div className="flex flex-col gap-2">
        <div className="flex items-baseline justify-between gap-3 text-sm">
          <p role="status" aria-live="polite" className="text-muted">
            {announcement(analysis)}
          </p>
          <span className="tabular font-semibold text-text">{analysis.progress}%</span>
        </div>
        <ProgressBar value={analysis.progress} label="Progreso del análisis" />
      </div>

      {status === 'done' && analysis.summary && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-primary/10 px-4 py-3">
          <p className="text-base font-semibold text-text">{analysis.summary.text}</p>
          <Link
            to="/anomalies"
            className="inline-flex h-10 items-center rounded-lg bg-primary-strong px-4 text-sm font-semibold text-white transition-colors duration-fast ease-console hover:bg-primary-strong/90"
          >
            Ver anomalías
          </Link>
        </div>
      )}
      {status === 'failed' && (
        <ErrorState title="El análisis falló" message="El análisis no se pudo completar. Intente de nuevo." onRetry={onRetry} />
      )}
    </Card>
  )
}
