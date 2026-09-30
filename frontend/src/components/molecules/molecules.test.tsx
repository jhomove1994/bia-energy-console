import { render, screen } from '@testing-library/react'
import { ConfidenceMeter } from './ConfidenceMeter'
import { KpiCard } from './KpiCard'
import { SeverityBadge } from './SeverityBadge'
import { StatusPill } from './StatusPill'
import { TypeBadge } from './TypeBadge'

describe('KpiCard', () => {
  it('shows the label, the value and the caption', () => {
    render(<KpiCard label="Total consumption" value="12,345.7 kWh" caption="Across all meters" />)
    expect(screen.getByText('Total consumption')).toBeInTheDocument()
    expect(screen.getByText('12,345.7 kWh')).toBeInTheDocument()
    expect(screen.getByText('Across all meters')).toBeInTheDocument()
  })

  it('exposes label and value together to assistive tech', () => {
    render(<KpiCard label="Meters" value="12" />)
    expect(screen.getByRole('group', { name: 'Meters' })).toHaveTextContent('12')
  })
})

describe('SeverityBadge', () => {
  it.each([
    ['HIGH', 'High'],
    ['MEDIUM', 'Medium'],
    ['LOW', 'Low'],
  ] as const)('labels %s as %s', (severity, label) => {
    render(<SeverityBadge severity={severity} />)
    expect(screen.getByText(label)).toBeInTheDocument()
  })
})

describe('TypeBadge', () => {
  it('humanizes the anomaly type', () => {
    render(<TypeBadge type="DATA_QUALITY" />)
    expect(screen.getByText('Data quality')).toBeInTheDocument()
  })
})

describe('StatusPill', () => {
  it.each(['OK', 'Alert', 'Critical'] as const)('shows the %s status text', (status) => {
    render(<StatusPill status={status} />)
    expect(screen.getByText(status)).toBeInTheDocument()
  })
})

describe('ConfidenceMeter', () => {
  it('shows the percent and a meter with the same value', () => {
    render(<ConfidenceMeter value={0.914} />)
    expect(screen.getByText('91%')).toBeInTheDocument()
    expect(screen.getByRole('meter', { name: 'Confidence' })).toHaveAttribute('aria-valuenow', '91')
  })

  it('renders a placeholder and no meter when there is no confidence yet', () => {
    render(<ConfidenceMeter value={null} />)
    expect(screen.getByText('—')).toBeInTheDocument()
    expect(screen.queryByRole('meter')).not.toBeInTheDocument()
  })
})
