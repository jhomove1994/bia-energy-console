import { apiFetch } from './client'
import type { AnalysisStarted } from './types'

/** Starts an analysis, or returns the id of the one already running (single-flight on the server). */
export function startAnalysis(): Promise<AnalysisStarted> {
  return apiFetch<AnalysisStarted>('/api/ai/analyze', { method: 'POST' })
}
