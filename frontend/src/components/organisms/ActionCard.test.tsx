import { render, screen } from '@testing-library/react'
import { ActionCard } from './ActionCard'

describe('ActionCard', () => {
  it('leads with the recommended action, its code and the priority rank', () => {
    render(<ActionCard action="Enviar una inspección en sitio a M-109 en un plazo de 24 h." code="INSPECT_LOAD_INCREASE" rank={1} type="REAL_ANOMALY" />)

    const card = screen.getByRole('region', { name: 'Acción recomendada' })
    expect(card).toHaveTextContent('Enviar una inspección en sitio a M-109 en un plazo de 24 h.')
    expect(card).toHaveTextContent('INSPECT_LOAD_INCREASE')
    expect(card).toHaveTextContent('Prioridad #1')
    expect(screen.getByRole('heading', { level: 2, name: 'Acción recomendada' })).toBeInTheDocument()
  })

  it('shows another finding with its own rank', () => {
    render(<ActionCard action="No se requiere acción." code="ACKNOWLEDGE_EXPECTED" rank={4} type="FALSE_POSITIVE" />)
    expect(screen.getByText('No se requiere acción.')).toBeInTheDocument()
    expect(screen.getByText('Prioridad #4')).toBeInTheDocument()
    expect(screen.getByText('ACKNOWLEDGE_EXPECTED')).toBeInTheDocument()
  })
})
