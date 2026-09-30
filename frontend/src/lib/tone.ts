import type { SeverityToken, StatusToken, TypeToken, VariationTone } from './status'

export type Tone = SeverityToken | TypeToken | StatusToken | 'neutral'

/*
 * Static class names per tone: Tailwind only emits classes it can read
 * verbatim, so these are spelled out instead of being interpolated.
 */
export const toneChip: Record<Tone, string> = {
  'sev-high': 'bg-sev-high/10 text-sev-high-ink ring-sev-high/25',
  'sev-medium': 'bg-sev-medium/10 text-sev-medium-ink ring-sev-medium/30',
  'sev-low': 'bg-sev-low/10 text-sev-low-ink ring-sev-low/25',
  'type-real': 'bg-type-real/10 text-type-real-ink ring-type-real/25',
  'type-dq': 'bg-type-dq/10 text-type-dq-ink ring-type-dq/25',
  'type-expl': 'bg-type-expl/10 text-type-expl-ink ring-type-expl/25',
  'type-fp': 'bg-type-fp/10 text-type-fp-ink ring-type-fp/25',
  'status-ok': 'bg-status-ok/10 text-status-ok-ink ring-status-ok/25',
  'status-alert': 'bg-status-alert/10 text-status-alert-ink ring-status-alert/30',
  'status-critical': 'bg-status-critical/10 text-status-critical-ink ring-status-critical/25',
  neutral: 'bg-muted/10 text-muted ring-muted/25',
}

export const toneDot: Record<Tone, string> = {
  'sev-high': 'bg-sev-high',
  'sev-medium': 'bg-sev-medium',
  'sev-low': 'bg-sev-low',
  'type-real': 'bg-type-real',
  'type-dq': 'bg-type-dq',
  'type-expl': 'bg-type-expl',
  'type-fp': 'bg-type-fp',
  'status-ok': 'bg-status-ok',
  'status-alert': 'bg-status-alert',
  'status-critical': 'bg-status-critical',
  neutral: 'bg-muted',
}

/** Text emphasis of a variation figure (daily variation, delta against the baseline). */
export const variationClass: Record<VariationTone, string> = {
  high: 'font-semibold text-sev-high-ink',
  medium: 'font-semibold text-sev-medium-ink',
  none: 'text-text',
}
