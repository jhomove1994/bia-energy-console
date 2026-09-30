import type { AnalysisStatus } from '../../api/types'

/** The pipeline stages in execution order; the API reports the current one as a 1-based `stage_index`. */
export const STAGES = ['Readings', 'Baseline', 'Detection', 'Correlation', 'Events', 'Explanation', 'Recommendation'] as const

type StepState = 'done' | 'current' | 'failed' | 'pending'

export interface StageStepperProps {
  /** 1-based index of the stage in progress; 0 while the run is pending. */
  stageIndex: number
  status: AnalysisStatus
}

const stateText: Record<StepState, string> = { done: 'done', current: 'in progress', failed: 'failed', pending: 'pending' }

function stepState(position: number, stageIndex: number, status: AnalysisStatus): StepState {
  if (status === 'done') return 'done'
  if (position < stageIndex) return 'done'
  if (position === stageIndex) return status === 'failed' ? 'failed' : 'current'
  return 'pending'
}

function Marker({ state, position }: { state: StepState; position: number }) {
  const base = 'flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold'
  if (state === 'done') {
    return (
      <span aria-hidden="true" className={`${base} bg-primary-strong text-white`}>
        <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M3 8.5 6.5 12 13 4.5" />
        </svg>
      </span>
    )
  }
  if (state === 'failed') {
    return (
      <span aria-hidden="true" className={`${base} bg-sev-high text-white`}>
        !
      </span>
    )
  }
  if (state === 'current') {
    return (
      <span aria-hidden="true" className={`${base} bg-primary/15 text-primary-strong ring-2 ring-primary-strong motion-safe:animate-pulse`}>
        {position}
      </span>
    )
  }
  return (
    <span aria-hidden="true" className={`${base} border border-border bg-surface text-muted`}>
      {position}
    </span>
  )
}

/** Seven named stages with a state each; the state is spelled out in text, never color alone. */
export function StageStepper({ stageIndex, status }: StageStepperProps) {
  return (
    <ol aria-label="Analysis stages" className="grid gap-2 sm:grid-cols-7">
      {STAGES.map((name, i) => {
        const position = i + 1
        const state = stepState(position, stageIndex, status)
        return (
          <li
            key={name}
            aria-current={state === 'current' ? 'step' : undefined}
            className="flex items-center gap-2.5 sm:flex-col sm:items-start sm:gap-2"
          >
            <Marker state={state} position={position} />
            <span className={`text-sm ${state === 'pending' ? 'text-muted' : 'font-medium text-text'}`}>
              {name}
              <span className="sr-only"> ({stateText[state]})</span>
            </span>
          </li>
        )
      })}
    </ol>
  )
}
