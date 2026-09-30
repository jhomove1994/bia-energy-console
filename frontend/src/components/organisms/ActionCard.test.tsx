import { render, screen } from '@testing-library/react'
import { ActionCard } from './ActionCard'

describe('ActionCard', () => {
  it('leads with the recommended action, its code and the priority rank', () => {
    render(<ActionCard action="Dispatch an on-site inspection of M-109 within 24 h." code="INSPECT_LOAD_INCREASE" rank={1} type="REAL_ANOMALY" />)

    const card = screen.getByRole('region', { name: 'Recommended action' })
    expect(card).toHaveTextContent('Dispatch an on-site inspection of M-109 within 24 h.')
    expect(card).toHaveTextContent('INSPECT_LOAD_INCREASE')
    expect(card).toHaveTextContent('Priority #1')
    expect(screen.getByRole('heading', { level: 2, name: 'Recommended action' })).toBeInTheDocument()
  })

  it('shows another finding with its own rank', () => {
    render(<ActionCard action="No action required." code="ACKNOWLEDGE_EXPECTED" rank={4} type="FALSE_POSITIVE" />)
    expect(screen.getByText('No action required.')).toBeInTheDocument()
    expect(screen.getByText('Priority #4')).toBeInTheDocument()
    expect(screen.getByText('ACKNOWLEDGE_EXPECTED')).toBeInTheDocument()
  })
})
