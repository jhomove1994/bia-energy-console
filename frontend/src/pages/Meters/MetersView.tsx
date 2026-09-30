import type { MeterRow } from '../../api/types'
import { Button } from '../../components/atoms/Button'
import { EmptyState } from '../../components/atoms/EmptyState'
import { ErrorState } from '../../components/atoms/ErrorState'
import { GaugeIcon } from '../../components/atoms/Icon'
import { Skeleton } from '../../components/atoms/Skeleton'
import { SearchFilterBar } from '../../components/molecules/SearchFilterBar'
import { MetersTable } from '../../components/organisms/MetersTable'

export interface MeterFilters {
  query: string
  status: string
  sort: string
}

export interface MetersViewProps {
  state: 'loading' | 'error' | 'ready'
  rows?: MeterRow[]
  filters: MeterFilters
  errorMessage?: string
  onQueryChange: (query: string) => void
  onStatusChange: (status: string) => void
  onSortChange: (sort: string) => void
  onRetry: () => void
  onClear: () => void
}

function LoadingSkeleton() {
  return (
    <div role="status" aria-label="Cargando medidores" className="flex flex-col gap-2 rounded-card border border-border bg-surface p-4 shadow-card">
      {Array.from({ length: 6 }, (_, i) => (
        <Skeleton key={i} className="h-9 w-full" />
      ))}
    </div>
  )
}

const count = (n: number): string => `${n} ${n === 1 ? 'medidor' : 'medidores'}`

export function MetersView({ state, rows, filters, errorMessage, onQueryChange, onStatusChange, onSortChange, onRetry, onClear }: MetersViewProps) {
  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Medidores</h1>
        <p className="mt-1 text-sm text-muted">Consumo, variación diaria frente al baseline y estado de cada medidor.</p>
      </div>

      <SearchFilterBar
        query={filters.query}
        status={filters.status}
        sort={filters.sort}
        onQueryChange={onQueryChange}
        onStatusChange={onStatusChange}
        onSortChange={onSortChange}
      />

      {state === 'loading' && <LoadingSkeleton />}
      {state === 'error' && <ErrorState title="No se pudieron cargar los medidores" message={errorMessage ?? 'Error desconocido'} onRetry={onRetry} />}
      {state === 'ready' && rows && rows.length === 0 && (
        <EmptyState
          icon={<GaugeIcon width={24} height={24} />}
          title="Ningún medidor coincide con los filtros"
          description="Pruebe con otro ID de medidor o estado, o borre los filtros para ver toda la flota."
          action={
            <Button variant="secondary" onClick={onClear}>
              Borrar filtros
            </Button>
          }
        />
      )}
      {state === 'ready' && rows && rows.length > 0 && (
        <>
          <p aria-live="polite" className="text-sm text-muted">
            {count(rows.length)}
          </p>
          <MetersTable rows={rows} />
        </>
      )}
    </div>
  )
}
