import { render, screen, within } from '@testing-library/react'
import type { EvidenceItem } from '../../api/types'
import { EvidenceTable } from './EvidenceTable'

const item = (over: Partial<EvidenceItem> = {}): EvidenceItem => ({
  metric: 'consumption_kwh',
  label: 'Hourly consumption (window mean)',
  baseline: 44.06637931034483,
  observed: 92.77241379310348,
  delta_pct: 110.52878690063979,
  unit: 'kWh/h',
  detail: '',
  ...over,
})

const bodyRows = () => within(screen.getByRole('table')).getAllByRole('row').slice(1)

describe('EvidenceTable', () => {
  it('has the metric, baseline, observed, delta and unit columns', () => {
    render(<EvidenceTable items={[item()]} />)
    expect(screen.getAllByRole('columnheader').map((h) => h.textContent)).toEqual(['Metric', 'Baseline', 'Observed', 'Delta', 'Unit'])
  })

  it('formats a consumption row with its delta and unit', () => {
    render(<EvidenceTable items={[item()]} />)
    const row = within(bodyRows()[0]!)
    expect(row.getByRole('rowheader')).toHaveTextContent('Hourly consumption (window mean)')
    expect(row.getByText('44.1')).toBeInTheDocument()
    expect(row.getByText('92.8')).toBeInTheDocument()
    expect(row.getByText('+110.5%')).toBeInTheDocument()
    expect(row.getByText('kWh/h')).toBeInTheDocument()
  })

  it('shows a negative delta with its sign and a power factor with two decimals', () => {
    render(<EvidenceTable items={[item({ metric: 'power_factor', label: 'Power factor (window mean)', baseline: 0.9401724, observed: 0.7396897, delta_pct: -21.324, unit: '' })]} />)
    const row = within(bodyRows()[0]!)
    expect(row.getByText('0.94')).toBeInTheDocument()
    expect(row.getByText('0.74')).toBeInTheDocument()
    expect(row.getByText('-21.3%')).toBeInTheDocument()
  })

  it('shows placeholders when there is no baseline or delta, and whole numbers for counts and hours', () => {
    render(<EvidenceTable items={[item({ metric: 'duration_hours', label: 'Duration', baseline: null, observed: 58, delta_pct: null, unit: 'h' })]} />)
    const cells = within(bodyRows()[0]!).getAllByRole('cell')
    expect(cells.map((c) => c.textContent)).toEqual(['—', '58', '—', 'h'])
  })

  it('keeps the detail of a row under its label', () => {
    render(<EvidenceTable items={[item({ metric: 'voltage_jumps', label: 'Voltage jumps', baseline: null, observed: 32, delta_pct: null, unit: 'count', detail: 'min 201.56 V, max 241.23 V' })]} />)
    expect(within(bodyRows()[0]!).getByRole('rowheader')).toHaveTextContent('min 201.56 V, max 241.23 V')
  })

  it('leaves the related event to its own card', () => {
    render(<EvidenceTable items={[item(), item({ metric: 'event', label: 'Related event', baseline: null, observed: 0, delta_pct: null, unit: 'h', detail: 'UNKNOWN: x' })]} />)
    expect(bodyRows()).toHaveLength(1)
    expect(screen.queryByText('Related event')).not.toBeInTheDocument()
  })

  it('says so when there is no numeric evidence', () => {
    render(<EvidenceTable items={[]} />)
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByText('No numeric evidence was recorded for this finding.')).toBeInTheDocument()
  })
})
