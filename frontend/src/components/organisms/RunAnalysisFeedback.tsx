import type { Analysis } from '../../api/types'
import { AnalysisProgressPanel } from './AnalysisProgressPanel'

export interface RunAnalysisFeedbackProps {
  analysis?: Analysis
  /** Why the last start attempt failed, if it did. */
  runError: string | null
  onRetry: () => void
}

/** Everything a page shows about a run it started: the failed-start alert and the live progress panel. */
export function RunAnalysisFeedback({ analysis, runError, onRetry }: RunAnalysisFeedbackProps) {
  return (
    <>
      {runError && (
        <p role="alert" className="rounded-lg bg-sev-high/10 px-4 py-3 text-sm font-medium text-sev-high-ink">
          Could not start the analysis: {runError}
        </p>
      )}
      {analysis && <AnalysisProgressPanel analysis={analysis} onRetry={onRetry} />}
    </>
  )
}
