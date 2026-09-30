import { toneDot } from '../../lib/tone'
import { typeToken, type AnomalyType } from '../../lib/status'
import { Card } from '../atoms/Card'
import { SparklesIcon } from '../atoms/Icon'

export interface ActionCardProps {
  action: string
  /** Machine code of the action, e.g. INSPECT_LOAD_INCREASE. */
  code: string
  /** Position of the finding in the priority ranking. */
  rank: number
  /** Tints the accent bar like the finding's type badge. */
  type: AnomalyType
}

/** The outcome of an investigation: what to do next. It leads the page's aside, larger than everything around it. */
export function ActionCard({ action, code, rank, type }: ActionCardProps) {
  return (
    <Card role="region" aria-labelledby="recommended-action-heading" padded={false} className="relative overflow-hidden shadow-raised">
      <span aria-hidden="true" className={`absolute inset-y-0 left-0 w-1.5 ${toneDot[typeToken(type)]}`} />
      <div className="flex flex-col gap-3 py-5 pl-7 pr-5">
        <div className="flex items-center justify-between gap-3">
          <h2 id="recommended-action-heading" className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-muted">
            <SparklesIcon aria-hidden="true" width={16} height={16} className="text-primary-strong" />
            Recommended action
          </h2>
          <span className="tabular text-xs font-semibold text-muted">Priority #{rank}</span>
        </div>
        <p className="text-lg font-semibold leading-snug text-text">{action}</p>
        <p>
          <code className="rounded-md bg-bg px-2 py-1 font-mono text-xs font-semibold text-text ring-1 ring-inset ring-border">{code}</code>
        </p>
      </div>
    </Card>
  )
}
