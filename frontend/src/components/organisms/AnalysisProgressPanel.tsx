import { Link } from 'react-router-dom'
import type { Analysis, AnalysisStatus } from '../../api/types'
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
const statusLabel: Record<AnalysisStatus, string> = { pending: 'Pending', running: 'Running', done: 'Done', failed: 'Failed' }

/** What a screen reader hears when the run moves on: a short sentence that changes per stage, not per percent. */
function announcement(a: Analysis): string {
  if (a.status === 'done') return 'Analysis complete'
  if (a.status === 'failed') return 'Analysis failed'
  const name = STAGES[a.stage_index - 1]
  return name ? `Stage ${a.stage_index} of ${a.stage_total}: ${name}` : 'Waiting to start'
}

/** Live view of one analysis run: status, stage stepper, progress, and the outcome (headline plus link, or the error). */
export function AnalysisProgressPanel({ analysis, onRetry }: AnalysisProgressPanelProps) {
  const { status } = analysis
  return (
    <Card aria-labelledby="analysis-progress-heading" role="region" className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id="analysis-progress-heading" className="text-base font-semibold text-text">
          AI analysis
        </h2>
        <Badge tone={statusTone[status]} dot>
          {statusLabel[status]}
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
        <ProgressBar value={analysis.progress} label="Analysis progress" />
      </div>

      {status === 'done' && analysis.summary && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-primary/10 px-4 py-3">
          <p className="text-base font-semibold text-text">{analysis.summary.text}</p>
          <Link
            to="/anomalies"
            className="inline-flex h-10 items-center rounded-lg bg-primary-strong px-4 text-sm font-semibold text-white transition-colors duration-fast ease-console hover:bg-primary-strong/90"
          >
            View anomalies
          </Link>
        </div>
      )}
      {status === 'failed' && (
        <ErrorState title="Analysis failed" message={analysis.error || 'The analysis failed without a reason. Try again.'} onRetry={onRetry} />
      )}
    </Card>
  )
}
