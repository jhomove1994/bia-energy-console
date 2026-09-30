import type { ReactNode } from 'react'
import { Card } from '../atoms/Card'

export interface KpiCardProps {
  label: string
  value: ReactNode
  caption?: ReactNode
  /** Optional decorative icon shown at the top right. */
  icon?: ReactNode
}

export function KpiCard({ label, value, caption, icon }: KpiCardProps) {
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
      <p className="tabular text-3xl font-semibold leading-none tracking-tight text-text">{value}</p>
      {caption && <p className="text-xs text-muted">{caption}</p>}
    </Card>
  )
}
