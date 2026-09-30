export interface SkeletonProps {
  className?: string
}

/** Placeholder block; decorative, so it is hidden from assistive tech. */
export function Skeleton({ className = 'h-4 w-full' }: SkeletonProps) {
  return <div aria-hidden="true" className={`rounded-md bg-border/70 motion-safe:animate-pulse ${className}`} />
}
