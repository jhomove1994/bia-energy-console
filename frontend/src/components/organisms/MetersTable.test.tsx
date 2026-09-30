import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { routerFuture } from '../../app/router'
import type { MeterRow } from '../../api/types'
import { MetersTable } from './MetersTable'

const rows: MeterRow[] = [
  { meter_id: 'M-109', status: 'Critical', last_reading_at: '2026-09-14T23:00:00Z', total_kwh: 12345.678, variation_pct: 110.5 },
  { meter_id: 'M-101', status: 'OK', last_reading_at: '2026-09-14T23:00:00Z', total_kwh: 999.5, variation_pct: -0.3 },
  { meter_id: 'M-110', status: 'OK', last_reading_at: null, total_kwh: 10, variation_pct: null },
]

function renderTable() {
  render(
    <MemoryRouter future={routerFuture}>
      <MetersTable rows={rows} />
    </MemoryRouter>,
  )
}

describe('MetersTable', () => {
  it('lists one row per meter with kWh, variation and status', () => {
    renderTable()
    const row = screen.getByRole('row', { name: /M-109/ })
    expect(within(row).getByText('12,345.7 kWh')).toBeInTheDocument()
    expect(within(row).getByText('+110.5%')).toBeInTheDocument()
    expect(within(row).getByText('Critical')).toBeInTheDocument()
    expect(within(screen.getByRole('row', { name: /M-101/ })).getByText('-0.3%')).toBeInTheDocument()
  })

  it('links each meter id to its detail page', () => {
    renderTable()
    expect(screen.getByRole('link', { name: 'M-109' })).toHaveAttribute('href', '/meters/M-109')
    expect(screen.getByRole('link', { name: 'M-101' })).toHaveAttribute('href', '/meters/M-101')
  })

  it('shows a placeholder when a meter has no variation yet', () => {
    renderTable()
    expect(within(screen.getByRole('row', { name: /M-110/ })).getByText('—')).toBeInTheDocument()
  })

  it('labels the columns', () => {
    renderTable()
    for (const name of ['Meter', 'Total consumption', 'Daily variation', 'Status']) {
      expect(screen.getByRole('columnheader', { name })).toBeInTheDocument()
    }
  })
})
