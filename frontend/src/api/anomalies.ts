import { apiFetch } from './client'
import type { Anomaly, AnomalyFilters, AnomalyList } from './types'

function query(filters: AnomalyFilters): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(filters)) {
    if (value) search.set(key, value)
  }
  const text = search.toString()
  return text ? `?${text}` : ''
}

/** Anomalies of the latest completed analysis, ranked by priority. */
export function listAnomalies(filters: AnomalyFilters): Promise<AnomalyList> {
  return apiFetch<AnomalyList>(`/api/anomalies${query(filters)}`)
}

export function getAnomaly(id: string): Promise<Anomaly> {
  return apiFetch<Anomaly>(`/api/anomalies/${encodeURIComponent(id)}`)
}
