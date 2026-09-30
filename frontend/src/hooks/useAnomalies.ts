import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getAnomaly, listAnomalies } from '../api/anomalies'
import type { AnomalyFilters } from '../api/types'

/**
 * Ranked anomalies for the filters. By default the previous rows stay visible while a new filter
 * loads; pass `keepPrevious: false` where the filter changes the subject (another meter).
 */
export function useAnomalies(filters: AnomalyFilters, { keepPrevious = true }: { keepPrevious?: boolean } = {}) {
  return useQuery({
    queryKey: ['anomalies', filters],
    queryFn: () => listAnomalies(filters),
    placeholderData: keepPrevious ? keepPreviousData : undefined,
    staleTime: 10_000,
  })
}

export function useAnomaly(id: string) {
  return useQuery({ queryKey: ['anomaly', id], queryFn: () => getAnomaly(id), staleTime: 10_000 })
}
