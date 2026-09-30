import { Button } from '../atoms/Button'
import { SparklesIcon } from '../atoms/Icon'

export interface RunAnalysisButtonProps {
  onRun: () => void
  /** The start request is in flight. */
  starting: boolean
  /** A run is starting or in flight: the button stays disabled. */
  busy: boolean
  /** 0..100 progress of the active run, shown in the label. */
  progress?: number
}

/** The accent trigger of the demo flow; while a run is active it is disabled and shows the current progress. */
export function RunAnalysisButton({ onRun, starting, busy, progress }: RunAnalysisButtonProps) {
  const running = busy && !starting
  return (
    <Button variant="accent" onClick={onRun} loading={starting} disabled={busy}>
      {!starting && <SparklesIcon width={16} height={16} />}
      {running ? `Analyzing… ${progress ?? 0}%` : 'Run AI Analysis'}
    </Button>
  )
}
