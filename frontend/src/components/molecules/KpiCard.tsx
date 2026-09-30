import type { ReactNode } from 'react'
import { Card } from '../atoms/Card'

export interface KpiCardProps {
  label: string
  value: ReactNode
  caption?: ReactNode
  /** `md` for text values such as dates; `lg` (default) for figures. */
  size?: 'lg' | 'md'
  /** Optional decorative icon shown at the top right. */
  icon?: ReactNode
}

export function KpiCard({ label, value, caption, icon, size = 'lg' }: KpiCardProps) {
  return (
    <Card role="group" aria-label={label} className="flex flex-col gap-2">
      <div className="flex items-start justify-between gap-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted">{label}</p>
        {icon && (
          <span aria-hidden="true" className="text-primary-strong">
            {icon}
          </span>
        )}
      </div>
      <div className={`tabular font-semibold tracking-tight text-text ${size === 'lg' ? 'text-3xl leading-none' : 'text-lg leading-snug'}`}>{value}</div>
      {caption && <p className="text-xs text-muted">{caption}</p>}
    </Card>
  )
}
