import { Input } from '../atoms/Input'
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
  { value: '', label: 'All statuses' },
  { value: 'OK', label: 'OK' },
  { value: 'Alert', label: 'Alert' },
  { value: 'Critical', label: 'Critical' },
]

export const sortOptions: SelectOption[] = [
  { value: 'meter_id:asc', label: 'Meter id (A to Z)' },
  { value: 'variation:desc', label: 'Variation (highest first)' },
  { value: 'variation:asc', label: 'Variation (lowest first)' },
  { value: 'kwh:desc', label: 'Consumption (highest first)' },
  { value: 'kwh:asc', label: 'Consumption (lowest first)' },
  { value: 'status:desc', label: 'Status (most severe first)' },
]

export function SearchFilterBar({ query, status, sort, onQueryChange, onStatusChange, onSortChange }: SearchFilterBarProps) {
  return (
    <div role="search" className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_11rem_15rem]">
      <Input
        id="meter-search"
        type="search"
        label="Search meters"
        placeholder="Meter id, e.g. 109"
        autoComplete="off"
        value={query}
        onChange={(e) => onQueryChange(e.target.value)}
      />
      <Select id="meter-status" label="Status" options={statusOptions} value={status} onChange={(e) => onStatusChange(e.target.value)} />
      <Select id="meter-sort" label="Sort by" options={sortOptions} value={sort} onChange={(e) => onSortChange(e.target.value)} />
    </div>
  )
}
