import type { ReactNode } from 'react'
import { Card } from './Card'

export interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description: ReactNode
  /** The next step, typically a Button. */
  action?: ReactNode
}

/** Guides the user when a view has nothing to show yet: what is missing and what to do about it. */
export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <Card className="flex flex-col items-center gap-3 px-6 py-10 text-center">
      {icon && (
        <span aria-hidden="true" className="flex h-12 w-12 items-center justify-center rounded-full bg-accent/15 text-sev-medium-ink">
          {icon}
        </span>
      )}
      <h2 className="text-base font-semibold text-text">{title}</h2>
      <p className="max-w-md text-sm leading-relaxed text-muted">{description}</p>
      {action && <div className="mt-2">{action}</div>}
    </Card>
  )
}
