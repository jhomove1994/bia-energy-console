import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactElement } from 'react'
import { cloneElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import type { Analysis, MeterDetail, MeterEvent, ReadingPoint } from '../../api/types'
import { makeAnomaly } from '../../test/fixtures'
import { routerFuture } from '../../app/router'
import { MeterDetailView, type MeterDetailViewProps } from './MeterDetailView'

vi.mock('recharts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('recharts')>()
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: ReactElement<{ width?: number; height?: number }> }) =>
      cloneElement(children, { width: 600, height: 200 }),
  }
})

const meter: MeterDetail = {
  meter_id: 'M-109',
  status: 'Critical',
  last_reading_at: '2026-09-14T23:00:00Z',
  total_kwh: 12345.678,
  variation_pct: 103.7,
  anomaly_count: 1,
}

// 24 readings: 2 kWh observed each against a 1 kWh baseline.
const readings: ReadingPoint[] = Array.from({ length: 24 }, (_, i) => ({
  timestamp: new Date(Date.UTC(2026, 8, 14, i)).toISOString(),
  consumption_kwh: 2,
  voltage_v: 230,
  current_a: 5,
  power_factor: 0.95,
  status: 'OK',
  baseline_kwh: 1,
  baseline_voltage_v: 230,
  baseline_current_a: 5,
  baseline_power_factor: 0.95,
}))

const runningAnalysis: Analysis = {
  id: 'a-1',
  status: 'running',
  stage: 'detection',
  stage_index: 3,
  stage_total: 7,
  progress: 28,
  created_at: '2026-09-30T10:00:00Z',
  started_at: '2026-09-30T10:00:01Z',
  finished_at: null,
  error: '',
  summary: null,
}

const events: MeterEvent[] = [{ id: 1, timestamp: '2026-09-12T14:00:00Z', type: 'UNKNOWN', description: 'Unexplained rise' }]

function setup(overrides: Partial<MeterDetailViewProps> = {}) {
  const props: MeterDetailViewProps = {
    meterId: 'M-109',
    state: 'ready',
    meter,
    readings: { status: 'ready', data: readings },
    events: { status: 'ready', data: events },
    anomalies: { status: 'ready', data: [] },
    onRetry: vi.fn(),
    onRun: vi.fn(),
    starting: false,
    busy: false,
    runError: null,
    ...overrides,
  }
  render(
    <MemoryRouter future={routerFuture}>
      <MeterDetailView {...props} />
    </MemoryRouter>,
  )
  return props
}

describe('MeterDetailView', () => {
  it('shows the meter id, status and consumption against the baseline', () => {
    setup()
    expect(screen.getByRole('heading', { level: 1, name: /M-109/ })).toBeInTheDocument()
    expect(screen.getByText('Crítico')).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Consumo total' })).toHaveTextContent('12.345,7 kWh')
    expect(screen.getByRole('group', { name: 'Variación diaria' })).toHaveTextContent('+103,7%')
    expect(screen.getByRole('group', { name: 'Últimas 24 h' })).toHaveTextContent('48,0 kWh')
    expect(screen.getByRole('group', { name: 'Últimas 24 h' })).toHaveTextContent('Baseline ~24,0 kWh')
    expect(screen.getByRole('group', { name: 'Hallazgos abiertos' })).toHaveTextContent('1')
  })

  it('renders the four charts and the events', () => {
    setup()
    expect(screen.getAllByRole('img')).toHaveLength(4)
    expect(screen.getByText('Unexplained rise')).toBeInTheDocument()
  })

  it('turns the open findings figure into a link to the filtered anomalies list', () => {
    setup()
    const kpi = screen.getByRole('group', { name: 'Hallazgos abiertos' })
    expect(within(kpi).getByRole('link', { name: 'Ver hallazgos' })).toHaveAttribute('href', '/anomalies?meter_id=M-109')
  })

  it('offers no link when the meter has no findings', () => {
    setup({ meter: { ...meter, anomaly_count: 0 } })
    expect(within(screen.getByRole('group', { name: 'Hallazgos abiertos' })).queryByRole('link')).not.toBeInTheDocument()
  })

  it('lists the anomalies of this meter with a link to each investigation', () => {
    setup({ anomalies: { status: 'ready', data: [makeAnomaly()] } })
    const section = screen.getByRole('region', { name: 'Anomalías de este medidor' })
    expect(within(section).getByText('Anomalía real')).toBeInTheDocument()
    expect(within(section).getByText('Cambio de nivel al alza')).toBeInTheDocument()
    expect(within(section).queryByRole('columnheader', { name: 'Medidor' })).not.toBeInTheDocument()
    expect(within(section).getByRole('link', { name: 'Investigar M-109' })).toHaveAttribute('href', '/anomalies/an-1')
  })

  it('says so when the latest analysis found nothing for the meter', () => {
    setup()
    expect(screen.getByRole('region', { name: 'Anomalías de este medidor' })).toHaveTextContent('Sin anomalías en el último análisis para este medidor.')
  })

  it('highlights the anomaly windows on the charts', () => {
    setup({ anomalies: { status: 'ready', data: [makeAnomaly()] } })
    for (const chart of screen.getAllByRole('img')) expect(chart).toHaveAccessibleName(/1 ventana de anomalía resaltada: Cambio de nivel al alza, severidad alta/)
  })

  it('shows a section error for the anomalies without hiding the meter or the charts', async () => {
    const props = setup({ anomalies: { status: 'error', message: 'Anomalías no disponibles' } })
    expect(screen.getByRole('alert')).toHaveTextContent('Anomalías no disponibles')
    expect(screen.getAllByRole('img')).toHaveLength(4)
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('shows a placeholder while the anomalies load', () => {
    setup({ anomalies: { status: 'loading' } })
    expect(screen.getByRole('status', { name: 'Cargando anomalías' })).toBeInTheDocument()
  })

  it('shows the loading state', () => {
    setup({ state: 'loading', meter: undefined })
    expect(screen.getByRole('status', { name: 'Cargando medidor' })).toBeInTheDocument()
  })

  it('shows a not-found state that leads back to the list', () => {
    setup({ state: 'not-found', meter: undefined })
    expect(screen.getByText('Medidor no encontrado')).toBeInTheDocument()
    expect(screen.getByText(/M-109/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver a medidores' })).toHaveAttribute('href', '/meters')
  })

  it('announces an error and retries', async () => {
    const props = setup({ state: 'error', meter: undefined, errorMessage: 'No se pudo conectar con el servidor.' })
    expect(screen.getByRole('alert')).toHaveTextContent('No se pudo conectar con el servidor.')
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('shows a section error for the charts without hiding the meter', async () => {
    const props = setup({ readings: { status: 'error', message: 'Lecturas no disponibles' } })
    expect(screen.getByRole('heading', { level: 1, name: /M-109/ })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('Lecturas no disponibles')
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('shows loading placeholders while the charts and events load', () => {
    setup({ readings: { status: 'loading' }, events: { status: 'loading' } })
    expect(screen.getByRole('status', { name: 'Cargando gráficos' })).toBeInTheDocument()
    expect(screen.getByRole('status', { name: 'Cargando eventos' })).toBeInTheDocument()
  })

  it('runs the analysis from the header', async () => {
    const props = setup()
    await userEvent.click(screen.getByRole('button', { name: 'Ejecutar análisis IA' }))
    expect(props.onRun).toHaveBeenCalled()
  })

  it('reports a failed start', () => {
    setup({ runError: 'busy' })
    expect(screen.getByRole('alert')).toHaveTextContent('No se pudo iniciar el análisis: busy')
  })

  it('shows the live progress panel and disables the trigger while a run is active', () => {
    setup({ busy: true, analysis: runningAnalysis })
    expect(screen.getByRole('region', { name: 'Análisis con IA' })).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: 'Progreso del análisis' })).toHaveAttribute('aria-valuenow', '28')
    expect(screen.getByRole('button', { name: /Analizando/ })).toBeDisabled()
  })

  it('shows no panel before any run', () => {
    setup()
    expect(screen.queryByRole('region', { name: 'Análisis con IA' })).not.toBeInTheDocument()
  })
})
