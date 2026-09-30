import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactElement } from 'react'
import { cloneElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import type { MeterDetail, MeterEvent, ReadingPoint } from '../../api/types'
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

const events: MeterEvent[] = [{ id: 1, timestamp: '2026-09-12T14:00:00Z', type: 'UNKNOWN', description: 'Unexplained rise' }]

function setup(overrides: Partial<MeterDetailViewProps> = {}) {
  const props: MeterDetailViewProps = {
    meterId: 'M-109',
    state: 'ready',
    meter,
    readings: { status: 'ready', data: readings },
    events: { status: 'ready', data: events },
    onRetry: vi.fn(),
    onRun: vi.fn(),
    starting: false,
    waiting: false,
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
    expect(screen.getByText('Critical')).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Total consumption' })).toHaveTextContent('12,345.7 kWh')
    expect(screen.getByRole('group', { name: 'Daily variation' })).toHaveTextContent('+103.7%')
    expect(screen.getByRole('group', { name: 'Latest 24 h' })).toHaveTextContent('48.0 kWh')
    expect(screen.getByRole('group', { name: 'Latest 24 h' })).toHaveTextContent('Baseline ~24.0 kWh')
    expect(screen.getByRole('group', { name: 'Open findings' })).toHaveTextContent('1')
  })

  it('renders the four charts and the events', () => {
    setup()
    expect(screen.getAllByRole('img')).toHaveLength(4)
    expect(screen.getByText('Unexplained rise')).toBeInTheDocument()
  })

  it('shows the loading state', () => {
    setup({ state: 'loading', meter: undefined })
    expect(screen.getByRole('status', { name: 'Loading meter' })).toBeInTheDocument()
  })

  it('shows a not-found state that leads back to the list', () => {
    setup({ state: 'not-found', meter: undefined })
    expect(screen.getByText('Meter not found')).toBeInTheDocument()
    expect(screen.getByText(/M-109/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to meters' })).toHaveAttribute('href', '/meters')
  })

  it('announces an error and retries', async () => {
    const props = setup({ state: 'error', meter: undefined, errorMessage: 'Cannot reach the server.' })
    expect(screen.getByRole('alert')).toHaveTextContent('Cannot reach the server.')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('shows a section error for the charts without hiding the meter', async () => {
    const props = setup({ readings: { status: 'error', message: 'readings down' } })
    expect(screen.getByRole('heading', { level: 1, name: /M-109/ })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('readings down')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('shows loading placeholders while the charts and events load', () => {
    setup({ readings: { status: 'loading' }, events: { status: 'loading' } })
    expect(screen.getByRole('status', { name: 'Loading charts' })).toBeInTheDocument()
    expect(screen.getByRole('status', { name: 'Loading events' })).toBeInTheDocument()
  })

  it('runs the analysis from the header', async () => {
    const props = setup()
    await userEvent.click(screen.getByRole('button', { name: 'Run AI Analysis' }))
    expect(props.onRun).toHaveBeenCalled()
  })

  it('reports a failed start and an analysis in flight', () => {
    setup({ runError: 'busy', waiting: true })
    expect(screen.getByRole('alert')).toHaveTextContent('Could not start the analysis: busy')
    expect(screen.getByText(/Analysis started/)).toBeInTheDocument()
  })
})
