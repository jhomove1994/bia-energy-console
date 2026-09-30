import { apiFetch } from './client'
import type { Analysis, AnalysisStarted } from './types'

/** Starts an analysis, or returns the id of the one already running (single-flight on the server). */
export function startAnalysis(): Promise<AnalysisStarted> {
  return apiFetch<AnalysisStarted>('/api/ai/analyze', { method: 'POST' })
}

export function getAnalysis(id: string): Promise<Analysis> {
  return apiFetch<Analysis>(`/api/ai/analysis/${encodeURIComponent(id)}`)
}

/** The most recent analysis in any state, or null before the first run (the API answers 204). */
export async function getLatestAnalysis(): Promise<Analysis | null> {
  return (await apiFetch<Analysis | undefined>('/api/ai/analysis/latest')) ?? null
}
