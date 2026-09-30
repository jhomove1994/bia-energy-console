import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactElement } from 'react'
import { cloneElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import type { EvidenceItem, ReadingPoint } from '../../api/types'
import { routerFuture } from '../../app/router'
import { makeAnomaly } from '../../test/fixtures'
import { InvestigationView, type InvestigationViewProps } from './InvestigationView'

vi.mock('recharts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('recharts')>()
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: ReactElement<{ width?: number; height?: number }> }) =>
      cloneElement(children, { width: 600, height: 200 }),
  }
})

// Hourly readings from 1 Sep to 30 Sep (720 points).
const readings: ReadingPoint[] = Array.from({ length: 720 }, (_, i) => ({
  timestamp: new Date(Date.UTC(2026, 8, 1, i)).toISOString(),
  consumption_kwh: 44,
  voltage_v: 220,
  current_a: 200,
  power_factor: 0.94,
  status: 'OK',
  baseline_kwh: 44,
  baseline_voltage_v: 220,
  baseline_current_a: 200,
  baseline_power_factor: 0.94,
}))

const evidence: EvidenceItem[] = [
  { metric: 'consumption_kwh', label: 'Hourly consumption (window mean)', baseline: 44.07, observed: 92.77, delta_pct: 110.53, unit: 'kWh/h', detail: '' },
  { metric: 'event', label: 'Evento relacionado', baseline: null, observed: 0, delta_pct: null, unit: 'h', detail: 'UNKNOWN: No operational event reported' },
]

const m109 = makeAnomaly({
  evidence,
  related_event: { id: 3, timestamp: '2026-09-12T14:00:00Z', type: 'UNKNOWN', description: 'No operational event reported' },
})

const m106 = makeAnomaly({
  id: 'an-4',
  meter_id: 'M-106',
  anomaly: 'Sustained decrease',
  type: 'FALSE_POSITIVE',
  severity: 'LOW',
  confidence: 0.858,
  confidence_label: 'Alta',
  priority_rank: 4,
  window_start: '2026-09-08T00:00:00Z',
  window_end: '2026-09-08T11:00:00Z',
  action_code: 'ACKNOWLEDGE_EXPECTED',
  recommended_action: 'No se requiere acción. The deviation matches the scheduled outage.',
  reason: 'Consumption fell 80% during a scheduled maintenance outage and recovered.',
  related_event: { id: 2, timestamp: '2026-09-08T00:00:00Z', type: 'SCHEDULED_OUTAGE', description: 'Scheduled maintenance outage for 12 hours' },
})

function setup(overrides: Partial<InvestigationViewProps> = {}) {
  const props: InvestigationViewProps = {
    state: 'ready',
    anomaly: m109,
    readings: { status: 'ready', data: readings },
    onRetry: vi.fn(),
    ...overrides,
  }
  render(
    <MemoryRouter future={routerFuture}>
      <InvestigationView {...props} />
    </MemoryRouter>,
  )
  return props
}

describe('InvestigationView for an unexplained real anomaly', () => {
  it('shows the meter, the anomaly, its type, severity and confidence', () => {
    setup()
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('M-109')
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Cambio de nivel al alza')
    expect(screen.getByText('Anomalía real')).toBeInTheDocument()
    expect(screen.getByText('Alta')).toBeInTheDocument()
    expect(screen.getByRole('meter', { name: 'Confianza' })).toHaveAttribute('aria-valuenow', '98')
    expect(screen.getByText('98%')).toBeInTheDocument()
  })

  it('explains why it was flagged', () => {
    setup()
    expect(screen.getByRole('heading', { name: 'Por qué se marcó' })).toBeInTheDocument()
    expect(screen.getByText('El consumo subió 110% y se mantiene; ningún evento registrado lo explica.')).toBeInTheDocument()
  })

  it('lists the numeric evidence', () => {
    setup()
    const table = screen.getByRole('table')
    expect(within(table).getByText('Consumo horario (promedio de la ventana)')).toBeInTheDocument()
    expect(within(table).getByText('+110,5%')).toBeInTheDocument()
  })

  it('states that no operational event explains the change', () => {
    setup()
    const card = screen.getByRole('region', { name: 'Evento relacionado' })
    expect(card).toHaveTextContent('Ningún evento operativo explica este cambio')
    expect(within(card).queryByText('Unknown')).not.toBeInTheDocument()
  })

  it('makes the recommended action prominent with its code and rank', () => {
    setup()
    const card = screen.getByRole('region', { name: 'Acción recomendada' })
    expect(card).toHaveTextContent('Enviar una inspección en sitio a M-109 en un plazo de 24 h.')
    expect(card).toHaveTextContent('INSPECT_LOAD_INCREASE')
    expect(card).toHaveTextContent('Prioridad #1')
  })

  it('charts the affected window zoomed to 24 hours around it, highlighted', () => {
    setup()
    // 12 Sep 14:00 to 14 Sep 23:00 plus 24 h each side: 11 Sep 14:00 to 15 Sep 23:00 = 106 hourly readings.
    const charts = screen.getAllByRole('img')
    expect(charts).toHaveLength(4)
    for (const chart of charts) {
      expect(chart).toHaveAccessibleName(/en 106 lecturas/)
      expect(chart).toHaveAccessibleName(/1 ventana de anomalía resaltada: Cambio de nivel al alza, severidad alta/)
    }
  })

  it('links back to the anomalies list and to the meter', () => {
    setup()
    expect(screen.getByRole('link', { name: 'Anomalías IA' })).toHaveAttribute('href', '/anomalies')
    expect(screen.getByRole('link', { name: 'Ver medidor M-109' })).toHaveAttribute('href', '/meters/M-109')
  })
})

describe('InvestigationView for an explained false positive', () => {
  it('shows the false positive, its low severity and the scheduled outage as related event', () => {
    setup({ anomaly: m106 })
    expect(screen.getByText('Falso positivo')).toBeInTheDocument()
    expect(screen.getByText('Baja')).toBeInTheDocument()
    const card = screen.getByRole('region', { name: 'Evento relacionado' })
    expect(card).toHaveTextContent('Parada programada')
    expect(card).toHaveTextContent('Scheduled maintenance outage for 12 hours')
    expect(card).not.toHaveTextContent('Ningún evento operativo explica este cambio')
    expect(screen.getByRole('region', { name: 'Acción recomendada' })).toHaveTextContent('Prioridad #4')
  })
})

describe('InvestigationView without a related event', () => {
  it('says no event explains the change', () => {
    setup({ anomaly: makeAnomaly({ related_event: null }) })
    expect(screen.getByRole('region', { name: 'Evento relacionado' })).toHaveTextContent('Ningún evento operativo explica este cambio')
  })
})

describe('InvestigationView states', () => {
  it('shows the loading state', () => {
    setup({ state: 'loading', anomaly: undefined })
    expect(screen.getByRole('status', { name: 'Cargando anomalía' })).toBeInTheDocument()
  })

  it('shows a not-found state that leads back to the list', () => {
    setup({ state: 'not-found', anomaly: undefined })
    expect(screen.getByText('Anomalía no encontrada')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver a anomalías' })).toHaveAttribute('href', '/anomalies')
  })

  it('announces an error and retries', async () => {
    const props = setup({ state: 'error', anomaly: undefined, errorMessage: 'No se pudo conectar con el servidor.' })
    expect(screen.getByRole('alert')).toHaveTextContent('No se pudo conectar con el servidor.')
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('keeps the finding visible when only the readings fail', async () => {
    const props = setup({ readings: { status: 'error', message: 'Lecturas no disponibles' } })
    expect(screen.getByRole('region', { name: 'Acción recomendada' })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('Lecturas no disponibles')
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('shows a placeholder while the readings load', () => {
    setup({ readings: { status: 'loading' } })
    expect(screen.getByRole('status', { name: 'Cargando gráficos' })).toBeInTheDocument()
  })
})
