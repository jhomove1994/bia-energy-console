import { formatConfidence } from '../../lib/format'

export interface ConfidenceMeterProps {
  /** 0..1, or null when no analysis has produced one yet. */
  value: number | null
  /** `lg` sizes the percent like a KPI figure. */
  size?: 'md' | 'lg'
}

export function ConfidenceMeter({ value, size = 'md' }: ConfidenceMeterProps) {
  if (value === null) return <span className="text-muted">—</span>
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
      <span className={`tabular font-semibold text-text ${size === 'lg' ? 'text-3xl leading-none' : 'text-sm'}`}>{formatConfidence(value)}</span>
    </div>
  )
}
