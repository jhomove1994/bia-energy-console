import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { routerFuture } from '../../app/router'
import type { MeterRow } from '../../api/types'
import { MetersView, type MetersViewProps } from './MetersView'

const rows: MeterRow[] = [{ meter_id: 'M-109', status: 'Critical', last_reading_at: null, total_kwh: 10, variation_pct: 110.5 }]

function setup(overrides: Partial<MetersViewProps> = {}) {
  const props: MetersViewProps = {
    state: 'ready',
    rows,
    filters: { query: '', status: '', sort: 'meter_id:asc' },
    onQueryChange: vi.fn(),
    onStatusChange: vi.fn(),
    onSortChange: vi.fn(),
    onRetry: vi.fn(),
    onClear: vi.fn(),
    ...overrides,
  }
  render(
    <MemoryRouter future={routerFuture}>
      <MetersView {...props} />
    </MemoryRouter>,
  )
  return props
}

describe('MetersView', () => {
  it('renders the table when ready', () => {
    setup()
    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(screen.getByText('1 meter')).toBeInTheDocument()
  })

  it('shows a loading status instead of the table', () => {
    setup({ state: 'loading', rows: undefined })
    expect(screen.getByRole('status', { name: 'Loading meters' })).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('announces an error and retries', async () => {
    const props = setup({ state: 'error', rows: undefined, errorMessage: 'Cannot reach the server.' })
    expect(screen.getByRole('alert')).toHaveTextContent('Cannot reach the server.')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('offers to clear the filters when nothing matches', async () => {
    const props = setup({ rows: [], filters: { query: 'zzz', status: '', sort: 'meter_id:asc' } })
    expect(screen.getByText('No meters match your filters')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Clear filters' }))
    expect(props.onClear).toHaveBeenCalled()
  })

  it('keeps the filter bar available while empty', () => {
    setup({ rows: [] })
    expect(screen.getByRole('searchbox', { name: 'Search meters' })).toBeInTheDocument()
  })
})
