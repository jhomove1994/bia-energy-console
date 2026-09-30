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

const typeLabels: Record<AnomalyType, string> = {
  REAL_ANOMALY: 'Real anomaly',
  EXPLAINABLE_ANOMALY: 'Explainable',
  FALSE_POSITIVE: 'False positive',
  DATA_QUALITY: 'Data quality',
}

const severityLabels: Record<Severity, string> = { HIGH: 'High', MEDIUM: 'Medium', LOW: 'Low' }

export const severityToken = (s: Severity): SeverityToken => severityTokens[s]
export const typeToken = (t: AnomalyType): TypeToken => typeTokens[t]
export const statusToken = (s: MeterStatus): StatusToken => statusTokens[s]
export const typeLabel = (t: AnomalyType): string => typeLabels[t]
export const severityLabel = (s: Severity): string => severityLabels[s]
export const statusLabel = (s: MeterStatus): string => s
