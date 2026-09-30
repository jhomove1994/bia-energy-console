import { Button } from './Button'

export interface ErrorStateProps {
  title?: string
  message: string
  onRetry: () => void
}

/** Announced failure with a Retry action; used by every data view. */
export function ErrorState({ title = 'Something went wrong', message, onRetry }: ErrorStateProps) {
  return (
    <div role="alert" className="flex flex-col items-start gap-3 rounded-card border border-sev-high/25 bg-sev-high/5 p-5 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p className="text-sm font-semibold text-sev-high-ink">{title}</p>
        <p className="mt-0.5 text-sm text-text/80">{message}</p>
      </div>
      <Button variant="secondary" onClick={onRetry}>
        Retry
      </Button>
    </div>
  )
}
