import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getAnomaly, listAnomalies } from '../api/anomalies'
import type { AnomalyFilters } from '../api/types'

/** Ranked anomalies for the filters; the previous rows stay visible while a new filter loads. */
export function useAnomalies(filters: AnomalyFilters) {
  return useQuery({
    queryKey: ['anomalies', filters],
    queryFn: () => listAnomalies(filters),
    placeholderData: keepPreviousData,
    staleTime: 10_000,
  })
}

export function useAnomaly(id: string) {
  return useQuery({ queryKey: ['anomaly', id], queryFn: () => getAnomaly(id), staleTime: 10_000 })
}
