import { useSearchParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import type { MeterListParams } from '../../api/types'
import { useMeters } from '../../hooks/useMeters'
import { MetersView } from './MetersView'

const DEFAULT_SORT = 'meter_id:asc'

/** `<key>:<order>` (the sort select) to the two API parameters. */
function splitSort(sort: string): { sort: string; order: string } {
  const [key = 'meter_id', order = 'asc'] = sort.split(':')
  return { sort: key, order }
}

/*
 * Container: the filters live in the URL search string (shareable links); the sort is stored as
 * separate `sort` and `order` parameters and defaults are left out to keep links short.
 */
export function MetersPage() {
  const [params, setParams] = useSearchParams()
  const query = params.get('q') ?? ''
  const status = params.get('status') ?? ''
  const sortKey = params.get('sort') ?? 'meter_id'
  const sortOrder = params.get('order') ?? 'asc'
  const sort = `${sortKey}:${sortOrder}`

  const apiParams: MeterListParams = { q: query, status, ...(sort === DEFAULT_SORT ? {} : splitSort(sort)) }
  const meters = useMeters(apiParams)

  const update = (changes: { q?: string; status?: string; sort?: string }) => {
    const next = new URLSearchParams(params)
    const merged = { q: query, status, sort, ...changes }
    const set = (key: string, value: string) => (value ? next.set(key, value) : next.delete(key))
    set('q', merged.q)
    set('status', merged.status)
    if (merged.sort === DEFAULT_SORT) {
      next.delete('sort')
      next.delete('order')
    } else {
      const s = splitSort(merged.sort)
      next.set('sort', s.sort)
      next.set('order', s.order)
    }
    setParams(next, { replace: true })
  }

  return (
    <MetersView
      state={meters.isPending ? 'loading' : meters.isError ? 'error' : 'ready'}
      rows={meters.data}
      filters={{ query, status, sort }}
      errorMessage={meters.isError ? (meters.error instanceof ApiError ? meters.error.message : 'Error inesperado') : undefined}
      onQueryChange={(q) => update({ q })}
      onStatusChange={(s) => update({ status: s })}
      onSortChange={(s) => update({ sort: s })}
      onRetry={() => void meters.refetch()}
      onClear={() => setParams(new URLSearchParams(), { replace: true })}
    />
  )
}
