import type { Analysis, Anomaly, AnomalyList } from '../api/types'

/** An anomaly as the API returns it; override only what a test cares about. */
export function makeAnomaly(over: Partial<Anomaly> = {}): Anomaly {
  return {
    id: 'an-1',
    analysis_id: 'a-1',
    meter_id: 'M-109',
    anomaly: 'Level shift up',
    kind: 'LEVEL_SHIFT_UP',
    type: 'REAL_ANOMALY',
    severity: 'HIGH',
    severity_score: 100,
    confidence: 0.975,
    confidence_label: 'Alta',
    priority_score: 97.5,
    priority_rank: 1,
    detected_at: '2026-09-12T14:00:00Z',
    window_start: '2026-09-12T14:00:00Z',
    window_end: '2026-09-14T23:00:00Z',
    reason: 'El consumo subió 110% y se mantiene; ningún evento registrado lo explica.',
    recommended_action: 'Enviar una inspección en sitio a M-109 en un plazo de 24 h.',
    action_code: 'INSPECT_LOAD_INCREASE',
    status: 'open',
    related_event: null,
    evidence: [],
    ...over,
  }
}

/** The four ranked outcomes of the demo dataset, in priority order. */
export function demoAnomalies(): Anomaly[] {
  return [
    makeAnomaly(),
    makeAnomaly({
      id: 'an-2',
      meter_id: 'M-112',
      anomaly: 'Data inconsistency',
      kind: 'DATA_INCONSISTENCY',
      type: 'DATA_QUALITY',
      severity: 'HIGH',
      confidence: 0.98,
      priority_rank: 2,
      action_code: 'OPEN_DQ_TICKET',
      recommended_action: 'Abrir un ticket de calidad de datos de medición para M-112.',
    }),
    makeAnomaly({
      id: 'an-3',
      meter_id: 'M-104',
      type: 'EXPLAINABLE_ANOMALY',
      severity: 'MEDIUM',
      confidence: 0.843,
      priority_rank: 3,
      action_code: 'UPDATE_BASELINE',
      recommended_action: 'Confirmar con operaciones que el cambio es permanente y recalcular el baseline de M-104.',
    }),
    makeAnomaly({
      id: 'an-4',
      meter_id: 'M-106',
      anomaly: 'Sustained decrease',
      kind: 'SUSTAINED_DECREASE',
      type: 'FALSE_POSITIVE',
      severity: 'LOW',
      confidence: 0.858,
      priority_rank: 4,
      action_code: 'ACKNOWLEDGE_EXPECTED',
      recommended_action: 'No se requiere acción.',
    }),
  ]
}

export function demoAnomalyList(items: Anomaly[] = demoAnomalies()): AnomalyList {
  return { analysis_id: 'a-1', summary_text: '4 anomalías detectadas · 2 requieren atención prioritaria', items }
}

export function makeAnalysis(over: Partial<Analysis> = {}): Analysis {
  return {
    id: 'a-1',
    status: 'running',
    stage: 'detection',
    stage_index: 3,
    stage_total: 7,
    progress: 28,
    created_at: '2026-09-30T10:00:00Z',
    started_at: '2026-09-30T10:00:01Z',
    finished_at: null,
    error: '',
    summary: null,
    ...over,
  }
}

export function doneAnalysis(text = '4 anomalías detectadas · 2 requieren atención prioritaria'): Analysis {
  return makeAnalysis({
    status: 'done',
    stage: 'recommendation',
    stage_index: 7,
    progress: 100,
    finished_at: '2026-09-30T10:00:05Z',
    summary: { total: 4, priority_attention: 2, meters_analyzed: 12, by_type: {}, by_severity: {}, avg_confidence: 0.914, text },
  })
}
