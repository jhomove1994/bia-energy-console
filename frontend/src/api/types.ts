// DTOs mirror the public API field names from the specs.

import type { AnomalyType, MeterStatus, Severity } from '../lib/status'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_at: string
  user: { username: string }
}

export type AnalysisStatus = 'pending' | 'running' | 'done' | 'failed'

export interface LastAnalysis {
  id: string
  status: AnalysisStatus
  finished_at: string | null
}

export interface DashboardSummary {
  meters_count: number
  total_kwh: number
  anomalies_count: number
  high_priority_count: number
  aggregate_confidence: number | null
  last_analysis: LastAnalysis | null
  summary_text: string | null
}

export interface AnalysisStarted {
  id: string
}

export interface MeterRow {
  meter_id: string
  status: MeterStatus
  last_reading_at: string | null
  total_kwh: number
  /** Daily variation against the baseline, in percent points; null when the meter has no baseline yet. */
  variation_pct: number | null
}

export interface MeterDetail extends MeterRow {
  anomaly_count: number
}

export interface ReadingPoint {
  timestamp: string
  consumption_kwh: number
  voltage_v: number
  current_a: number
  power_factor: number
  status: string
  /** Hour-of-day baselines; 0 when the meter has too little history to build one. */
  baseline_kwh: number
  baseline_voltage_v?: number
  baseline_current_a?: number
  baseline_power_factor?: number
}

export interface MeterEvent {
  id: number
  timestamp: string
  type: string
  description: string
}

export type MeterSortKey = 'meter_id' | 'variation' | 'kwh' | 'status'
export type SortOrder = 'asc' | 'desc'

export interface MeterListParams {
  q?: string
  status?: string
  sort?: string
  order?: string
}

export type AnalysisStage = 'readings' | 'baseline' | 'detection' | 'correlation' | 'events' | 'explanation' | 'recommendation'

export interface AnalysisSummary {
  total: number
  priority_attention: number
  meters_analyzed: number
  by_type: Record<string, number>
  by_severity: Record<string, number>
  avg_confidence: number
  /** Ready-made headline, e.g. "4 anomalies detected · 2 require priority attention". */
  text: string
}

export interface Analysis {
  id: string
  status: AnalysisStatus
  /** Name of the stage in progress; empty while pending. */
  stage: AnalysisStage | ''
  /** 1-based position of the current stage; 0 while pending. */
  stage_index: number
  stage_total: number
  /** 0..100 */
  progress: number
  created_at: string
  started_at: string | null
  finished_at: string | null
  /** Empty unless the run failed. */
  error: string
  /** Present only when the run is done. */
  summary: AnalysisSummary | null
}

export interface EvidenceItem {
  metric: string
  label: string
  baseline: number | null
  observed: number
  delta_pct: number | null
  unit: string
  detail: string
}

export interface Anomaly {
  id: string
  analysis_id: string
  meter_id: string
  /** Short human title of the kind, e.g. "Level shift up". */
  anomaly: string
  kind: string
  type: AnomalyType
  severity: Severity
  severity_score: number
  confidence: number
  confidence_label: string
  priority_score: number
  priority_rank: number
  detected_at: string
  window_start: string
  window_end: string
  reason: string
  recommended_action: string
  action_code: string
  status: string
  related_event: MeterEvent | null
  evidence: EvidenceItem[]
}

export interface AnomalyList {
  /** Null until an analysis has completed. */
  analysis_id: string | null
  summary_text: string | null
  items: Anomaly[]
}

export interface AnomalyFilters {
  type?: string
  severity?: string
  meter_id?: string
}
