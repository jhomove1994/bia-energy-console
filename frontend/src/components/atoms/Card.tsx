import type { HTMLAttributes } from 'react'

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  padded?: boolean
}

export function Card({ padded = true, className = '', children, ...rest }: CardProps) {
  return (
    <div {...rest} className={`rounded-card border border-border bg-surface shadow-card ${padded ? 'p-5' : ''} ${className}`}>
      {children}
    </div>
  )
}
