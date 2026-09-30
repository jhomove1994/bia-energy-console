import type { ReactNode } from 'react'
import { toneChip, toneDot, type Tone } from '../../lib/tone'

export interface BadgeProps {
  tone?: Tone
  children: ReactNode
  /** Shows a colored dot before the label so state is never conveyed by color alone. */
  dot?: boolean
}

export function Badge({ tone = 'neutral', dot = false, children }: BadgeProps) {
  return (
    <span className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-semibold ring-1 ring-inset ${toneChip[tone]}`}>
      {dot && <span aria-hidden="true" className={`h-1.5 w-1.5 rounded-full ${toneDot[tone]}`} />}
      {children}
    </span>
  )
}
