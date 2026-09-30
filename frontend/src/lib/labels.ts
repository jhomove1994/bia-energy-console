import type { AnalysisStage, AnalysisStatus } from '../api/types'
import type { AnomalyType, MeterStatus, Severity } from './status'

/*
 * Every user-facing name of an API value lives here, in neutral Spanish. The API keeps its English
 * enum values and codes; components never render those raw, they ask this module.
 */

const typeLabels: Record<AnomalyType, string> = {
  REAL_ANOMALY: 'Anomalía real',
  EXPLAINABLE_ANOMALY: 'Anomalía explicable',
  FALSE_POSITIVE: 'Falso positivo',
  DATA_QUALITY: 'Calidad de datos',
}

const severityLabels: Record<Severity, string> = { HIGH: 'Alta', MEDIUM: 'Media', LOW: 'Baja' }

const statusLabels: Record<MeterStatus, string> = { OK: 'Normal', Alert: 'Alerta', Critical: 'Crítico' }

const analysisStatusLabels: Record<AnalysisStatus, string> = {
  pending: 'Pendiente',
  running: 'En ejecución',
  done: 'Completado',
  failed: 'Fallido',
}

/** The pipeline stages in execution order; the API reports the current one as a 1-based `stage_index`. */
export const STAGE_ORDER: readonly AnalysisStage[] = ['readings', 'baseline', 'detection', 'correlation', 'events', 'explanation', 'recommendation']

const stageLabels: Record<AnalysisStage, string> = {
  readings: 'Lecturas',
  baseline: 'Baseline',
  detection: 'Detección',
  correlation: 'Correlación',
  events: 'Eventos',
  explanation: 'Explicación',
  recommendation: 'Recomendación',
}

const eventLabels: Record<string, string> = {
  OPERATIONAL_CHANGE: 'Cambio operativo',
  SCHEDULED_OUTAGE: 'Parada programada',
  DATA_QUALITY: 'Calidad de datos',
  UNKNOWN: 'Desconocido',
}

const confidenceLabels: Record<string, string> = { High: 'alta', Medium: 'media', Low: 'baja' }

const kindLabels: Record<string, string> = {
  CONSUMPTION_SPIKE: 'Pico de consumo',
  CONSUMPTION_DIP: 'Caída de consumo',
  SUSTAINED_INCREASE: 'Aumento sostenido',
  SUSTAINED_DECREASE: 'Disminución sostenida',
  LEVEL_SHIFT_UP: 'Cambio de nivel al alza',
  LEVEL_SHIFT_DOWN: 'Cambio de nivel a la baja',
  DATA_INCONSISTENCY: 'Inconsistencia de datos',
  PATTERN_CHANGE: 'Cambio de patrón',
}

const metricLabels: Record<string, string> = {
  consumption_kwh: 'Consumo horario (promedio de la ventana)',
  daily_kwh: 'Energía diaria (peor día)',
  current_a: 'Corriente (promedio de la ventana)',
  power_factor: 'Factor de potencia (promedio de la ventana)',
  voltage_v: 'Voltaje (promedio de la ventana)',
  electrical_ratio: 'Relación eléctrica (mediana de la ventana)',
  duration_hours: 'Duración',
  hourly_pattern: 'Correlación del patrón horario',
  event: 'Evento relacionado',
  voltage_jumps: 'Saltos de voltaje',
  current_jumps: 'Saltos de corriente',
  pf_jumps: 'Saltos de factor de potencia',
  gaps: 'Lecturas faltantes',
  status: 'Lecturas con estado distinto de OK',
}

const unitLabels: Record<string, string> = { count: 'cantidad', corr: 'correlación' }

const actionLabels: Record<string, string> = {
  INSPECT_LOAD_INCREASE: 'Inspeccionar aumento de carga',
  CHECK_SUPPLY_LOSS: 'Verificar pérdida de suministro',
  REVIEW_TRANSIENT: 'Revisar transitorio',
  REVIEW_SCHEDULE: 'Revisar horario',
  OPEN_DQ_TICKET: 'Abrir ticket de calidad de datos',
  UPDATE_BASELINE: 'Actualizar baseline',
  ACKNOWLEDGE_EXPECTED: 'Reconocer como esperado',
}

export const typeLabel = (t: AnomalyType): string => typeLabels[t]
export const severityLabel = (s: Severity): string => severityLabels[s]
export const statusLabel = (s: MeterStatus): string => statusLabels[s]
export const analysisStatusLabel = (s: AnalysisStatus): string => analysisStatusLabels[s]
export const stageLabel = (s: AnalysisStage): string => stageLabels[s]

/** Event type name; anything the API sends that is not a known type reads as "Desconocido". */
export function eventLabel(type: string): string {
  return eventLabels[type.toUpperCase()] ?? eventLabels.UNKNOWN!
}

/** Confidence level as a lowercase adjective ("alta"), for "confianza alta". */
export const confidenceLabel = (label: string): string => confidenceLabels[label] ?? label.toLowerCase()

/** Short name of an anomaly kind; the API's own (English) title is only a fallback for unknown kinds. */
export const kindLabel = (kind: string, fallback: string): string => kindLabels[kind] ?? fallback

/** Evidence row name by metric key; the API label is only a fallback for unknown metrics. */
export const metricLabel = (metric: string, fallback: string): string => metricLabels[metric] ?? fallback

export const unitLabel = (unit: string): string => unitLabels[unit] ?? unit

/** Short name of a recommended-action code for tables; unknown codes are humanized. */
export function actionLabel(code: string): string {
  const known = actionLabels[code]
  if (known) return known
  const text = code.replace(/_/g, ' ').toLowerCase()
  return text.charAt(0).toUpperCase() + text.slice(1)
}

const errorMessages: Record<string, string> = {
  invalid_credentials: 'Usuario o contraseña incorrectos.',
  unauthorized: 'Su sesión expiró. Inicie sesión de nuevo.',
  not_found: 'No se encontró el recurso solicitado.',
  invalid_body: 'La solicitud enviada no es válida.',
  invalid_query: 'Los filtros indicados no son válidos.',
  internal: 'Ocurrió un error interno en el servidor. Intente de nuevo.',
  network: 'No se pudo conectar con el servidor. Revise su conexión e intente de nuevo.',
}

/** User-facing text for an API error code; unknown codes fall back to the HTTP status. */
export function apiErrorMessage(code: string, status: number): string {
  return errorMessages[code] ?? `El servidor no pudo completar la solicitud (estado ${status}).`
}
