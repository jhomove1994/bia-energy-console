import type { SeverityToken } from './status'

/*
 * Chart colors are read from the design tokens at paint time (`rgb(var(--token))`), so charts follow
 * the theme and no raw hex lives in components. The two line styles differ by dash pattern as well as
 * by color, so the baseline is never told apart by hue alone.
 */
const token = (name: string): string => `rgb(var(--${name}))`

export const chartTheme = {
  observed: token('primary'),
  baseline: token('muted'),
  grid: token('border'),
  axis: token('muted'),
  event: token('type-dq'),
  area: {
    'sev-high': token('sev-high'),
    'sev-medium': token('sev-medium'),
    'sev-low': token('sev-low'),
  } satisfies Record<SeverityToken, string>,
} as const
