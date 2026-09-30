import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getMeter, listMeters } from '../api/meters'
import type { MeterListParams } from '../api/types'

/** Meter list for the given filters; the previous rows stay visible while a new filter loads. */
export function useMeters(params: MeterListParams) {
  return useQuery({
    queryKey: ['meters', params],
    queryFn: () => listMeters(params),
    placeholderData: keepPreviousData,
    staleTime: 10_000,
  })
}

export function useMeter(id: string) {
  return useQuery({ queryKey: ['meter', id], queryFn: () => getMeter(id), staleTime: 10_000 })
}
