import { useQuery } from '@tanstack/react-query'
import { getMeterEvents, getMeterReadings } from '../api/meters'

/** Full readings series with its baseline (the detail page fetches it in parallel with the meter and events). An empty id waits: pages that learn the meter from another query pass '' until then. */
export function useMeterReadings(id: string) {
  return useQuery({ queryKey: ['readings', id], queryFn: () => getMeterReadings(id), staleTime: 60_000, enabled: id !== '' })
}

export function useMeterEvents(id: string) {
  return useQuery({ queryKey: ['events', id], queryFn: () => getMeterEvents(id), staleTime: 60_000 })
}
