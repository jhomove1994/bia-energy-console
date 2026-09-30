import { formatConfidence } from '../../lib/format'

export interface ConfidenceMeterProps {
  /** 0..1, or null when no analysis has produced one yet. */
  value: number | null
}

export function ConfidenceMeter({ value }: ConfidenceMeterProps) {
  if (value === null) return <span className="tabular text-muted">—</span>
  const percent = Math.round(value * 100)
  return (
    <div className="flex items-center gap-2.5">
      <div
        role="meter"
        aria-label="Confidence"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent}
        className="h-1.5 w-20 overflow-hidden rounded-full bg-border"
      >
        <div className="h-full rounded-full bg-primary" style={{ width: `${percent}%` }} />
      </div>
      <span className="tabular text-sm font-semibold text-text">{formatConfidence(value)}</span>
    </div>
  )
}
