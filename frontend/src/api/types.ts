// DTOs mirror the public API field names from the specs.

import type { MeterStatus } from '../lib/status'

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
