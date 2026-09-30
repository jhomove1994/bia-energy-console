import { apiFetch } from './client'
import type { MeterDetail, MeterEvent, MeterListParams, MeterRow, ReadingPoint } from './types'

function query(params: MeterListParams): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value) search.set(key, value)
  }
  const text = search.toString()
  return text ? `?${text}` : ''
}

const meterPath = (id: string): string => `/api/meters/${encodeURIComponent(id)}`

export async function listMeters(params: MeterListParams): Promise<MeterRow[]> {
  const res = await apiFetch<{ items: MeterRow[] }>(`/api/meters${query(params)}`)
  return res.items
}

export function getMeter(id: string): Promise<MeterDetail> {
  return apiFetch<MeterDetail>(meterPath(id))
}

export async function getMeterReadings(id: string): Promise<ReadingPoint[]> {
  const res = await apiFetch<{ points: ReadingPoint[] }>(`${meterPath(id)}/readings`)
  return res.points
}

export async function getMeterEvents(id: string): Promise<MeterEvent[]> {
  const res = await apiFetch<{ items: MeterEvent[] }>(`${meterPath(id)}/events`)
  return res.items
}
