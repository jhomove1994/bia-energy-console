export type Severity = 'HIGH' | 'MEDIUM' | 'LOW'
export type AnomalyType = 'REAL_ANOMALY' | 'EXPLAINABLE_ANOMALY' | 'FALSE_POSITIVE' | 'DATA_QUALITY'
export type MeterStatus = 'OK' | 'Alert' | 'Critical'

/** Design token names (see styles/tokens.css); Tailwind exposes each as a color. */
export type SeverityToken = 'sev-high' | 'sev-medium' | 'sev-low'
export type TypeToken = 'type-real' | 'type-dq' | 'type-expl' | 'type-fp'
export type StatusToken = 'status-ok' | 'status-alert' | 'status-critical'

const severityTokens: Record<Severity, SeverityToken> = {
  HIGH: 'sev-high',
  MEDIUM: 'sev-medium',
  LOW: 'sev-low',
}

const typeTokens: Record<AnomalyType, TypeToken> = {
  REAL_ANOMALY: 'type-real',
  DATA_QUALITY: 'type-dq',
  EXPLAINABLE_ANOMALY: 'type-expl',
  FALSE_POSITIVE: 'type-fp',
}

const statusTokens: Record<MeterStatus, StatusToken> = {
  OK: 'status-ok',
  Alert: 'status-alert',
  Critical: 'status-critical',
}

export const severityToken = (s: Severity): SeverityToken => severityTokens[s]
export const typeToken = (t: AnomalyType): TypeToken => typeTokens[t]
export const statusToken = (s: MeterStatus): StatusToken => statusTokens[s]

export type VariationTone = 'high' | 'medium' | 'none'

/** How loudly to show a daily variation (percent points): big moves in either direction stand out. */
export function variationTone(pct: number | null): VariationTone {
  if (pct === null) return 'none'
  const magnitude = Math.abs(pct)
  if (magnitude >= 25) return 'high'
  if (magnitude >= 10) return 'medium'
  return 'none'
}
