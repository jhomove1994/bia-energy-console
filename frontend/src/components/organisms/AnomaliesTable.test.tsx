import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { routerFuture } from '../../app/router'
import { demoAnomalies } from '../../test/fixtures'
import { AnomaliesTable } from './AnomaliesTable'

function renderTable(props: Partial<Parameters<typeof AnomaliesTable>[0]> = {}) {
  render(
    <MemoryRouter future={routerFuture}>
      <AnomaliesTable anomalies={demoAnomalies()} {...props} />
    </MemoryRouter>,
  )
}

const bodyRows = () => within(screen.getByRole('table')).getAllByRole('row').slice(1)

describe('AnomaliesTable', () => {
  it('has labelled column headers', () => {
    renderTable()
    const headers = screen.getAllByRole('columnheader').map((h) => h.textContent)
    expect(headers).toEqual(['Priority', 'Meter', 'Anomaly', 'Type', 'Severity', 'Confidence', 'Detected window', 'Recommended action', 'Investigate'])
  })

  it('lists the anomalies in the order given, ranked by priority', () => {
    renderTable()
    const meters = bodyRows().map((row) => within(row).getByRole('rowheader').textContent)
    expect(meters).toEqual(['M-109', 'M-112', 'M-104', 'M-106'])
    expect(within(bodyRows()[0]!).getByText('#1')).toBeInTheDocument()
    expect(within(bodyRows()[3]!).getByText('#4')).toBeInTheDocument()
  })

  it('shows type, severity, confidence, window and a short recommended action per row', () => {
    renderTable()
    const first = within(bodyRows()[0]!)
    expect(first.getByText('Real anomaly')).toBeInTheDocument()
    expect(first.getByText('High')).toBeInTheDocument()
    expect(first.getByText('98%')).toBeInTheDocument()
    expect(first.getByText('Level shift up')).toBeInTheDocument()
    expect(first.getByText('12 Sep 2026, 14:00 UTC – 14 Sep 2026, 23:00 UTC')).toBeInTheDocument()
    expect(first.getByText('Inspect load increase')).toBeInTheDocument()

    const last = within(bodyRows()[3]!)
    expect(last.getByText('False positive')).toBeInTheDocument()
    expect(last.getByText('Low')).toBeInTheDocument()
    expect(last.getByText('Acknowledge as expected')).toBeInTheDocument()
  })

  it('links every row to its investigation', () => {
    renderTable()
    expect(screen.getByRole('link', { name: 'Investigate M-109' })).toHaveAttribute('href', '/anomalies/an-1')
    expect(screen.getByRole('link', { name: 'Investigate M-106' })).toHaveAttribute('href', '/anomalies/an-4')
    expect(screen.getAllByRole('link')).toHaveLength(4)
  })

  it('can hide the meter column when the list belongs to one meter', () => {
    renderTable({ showMeter: false })
    expect(screen.queryByRole('columnheader', { name: 'Meter' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Investigate M-109' })).toBeInTheDocument()
  })
})
