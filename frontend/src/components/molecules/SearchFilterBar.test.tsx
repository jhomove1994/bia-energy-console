import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SearchFilterBar, type SearchFilterBarProps } from './SearchFilterBar'

function setup(overrides: Partial<SearchFilterBarProps> = {}) {
  const props: SearchFilterBarProps = {
    query: '',
    status: '',
    sort: 'meter_id:asc',
    onQueryChange: vi.fn(),
    onStatusChange: vi.fn(),
    onSortChange: vi.fn(),
    ...overrides,
  }
  render(<SearchFilterBar {...props} />)
  return props
}

describe('SearchFilterBar', () => {
  it('shows the current values', () => {
    setup({ query: '109', status: 'Critical', sort: 'variation:desc' })
    expect(screen.getByRole('searchbox', { name: 'Search meters' })).toHaveValue('109')
    expect(screen.getByRole('combobox', { name: 'Status' })).toHaveValue('Critical')
    expect(screen.getByRole('combobox', { name: 'Sort by' })).toHaveValue('variation:desc')
  })

  it('reports each typed character of the search', async () => {
    const props = setup()
    await userEvent.type(screen.getByRole('searchbox', { name: 'Search meters' }), '10')
    expect(props.onQueryChange).toHaveBeenNthCalledWith(1, '1')
    expect(props.onQueryChange).toHaveBeenNthCalledWith(2, '0')
  })

  it('reports the chosen status and sort', async () => {
    const props = setup()
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Status' }), 'Alert')
    expect(props.onStatusChange).toHaveBeenCalledWith('Alert')
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Sort by' }), 'kwh:desc')
    expect(props.onSortChange).toHaveBeenCalledWith('kwh:desc')
  })

  it('offers all statuses plus an "All" choice', () => {
    setup()
    const names = screen.getAllByRole('option').map((o) => o.textContent)
    expect(names).toEqual(expect.arrayContaining(['All statuses', 'OK', 'Alert', 'Critical']))
  })
})
