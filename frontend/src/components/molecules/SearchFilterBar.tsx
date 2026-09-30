import { Input } from '../atoms/Input'
import { statusLabel } from '../../lib/labels'
import { Select, type SelectOption } from '../atoms/Select'

export interface SearchFilterBarProps {
  query: string
  /** Empty string means every status. */
  status: string
  /** `<key>:<order>`, one of the sort options below. */
  sort: string
  onQueryChange: (query: string) => void
  onStatusChange: (status: string) => void
  onSortChange: (sort: string) => void
}

export const statusOptions: SelectOption[] = [
  { value: '', label: 'Todos los estados' },
  { value: 'OK', label: statusLabel('OK') },
  { value: 'Alert', label: statusLabel('Alert') },
  { value: 'Critical', label: statusLabel('Critical') },
]

export const sortOptions: SelectOption[] = [
  { value: 'meter_id:asc', label: 'ID del medidor (A a Z)' },
  { value: 'variation:desc', label: 'Variación (mayor primero)' },
  { value: 'variation:asc', label: 'Variación (menor primero)' },
  { value: 'kwh:desc', label: 'Consumo (mayor primero)' },
  { value: 'kwh:asc', label: 'Consumo (menor primero)' },
  { value: 'status:desc', label: 'Estado (más grave primero)' },
]

export function SearchFilterBar({ query, status, sort, onQueryChange, onStatusChange, onSortChange }: SearchFilterBarProps) {
  return (
    <div role="search" className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_11rem_15rem]">
      <Input
        id="meter-search"
        type="search"
        label="Buscar medidores"
        placeholder="ID del medidor, p. ej. 109"
        autoComplete="off"
        value={query}
        onChange={(e) => onQueryChange(e.target.value)}
      />
      <Select id="meter-status" label="Estado" options={statusOptions} value={status} onChange={(e) => onStatusChange(e.target.value)} />
      <Select id="meter-sort" label="Ordenar por" options={sortOptions} value={sort} onChange={(e) => onSortChange(e.target.value)} />
    </div>
  )
}
