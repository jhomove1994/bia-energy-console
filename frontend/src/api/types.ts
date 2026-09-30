// DTOs mirror the public API field names from the specs.

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
