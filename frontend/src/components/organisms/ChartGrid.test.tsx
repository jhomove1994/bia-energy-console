import { render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import { cloneElement } from 'react'
import type { MeterEvent, ReadingPoint } from '../../api/types'
import { ChartGrid } from './ChartGrid'

vi.mock('recharts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('recharts')>()
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: ReactElement<{ width?: number; height?: number }> }) =>
      cloneElement(children, { width: 600, height: 200 }),
  }
})

const readings: ReadingPoint[] = Array.from({ length: 4 }, (_, i) => ({
  timestamp: `2026-09-12T${String(10 + i).padStart(2, '0')}:00:00Z`,
  consumption_kwh: 40 + i,
  voltage_v: 230,
  current_a: 5,
  power_factor: 0.95,
  status: 'OK',
  baseline_kwh: 40,
  baseline_voltage_v: 231,
  baseline_current_a: 5,
  baseline_power_factor: 0.95,
}))

const events: MeterEvent[] = [{ id: 1, timestamp: '2026-09-12T11:00:00Z', type: 'UNKNOWN', description: 'x' }]

describe('ChartGrid', () => {
  it('renders one chart per metric: kWh, voltage, current and power factor', () => {
    render(<ChartGrid readings={readings} events={events} />)
    expect(screen.getAllByRole('img')).toHaveLength(4)
    for (const name of [/Consumo \(kWh\)/, /Voltaje \(V\)/, /Corriente \(A\)/, /Factor de potencia \(FP\)/]) {
      expect(screen.getByRole('img', { name })).toBeInTheDocument()
    }
  })

  it('zooms every chart to a time range', () => {
    // Readings at 10:00, 11:00, 12:00 and 13:00; the range keeps 11:00 to 12:00.
    render(<ChartGrid readings={readings} events={[]} range={{ from: Date.parse('2026-09-12T11:00:00Z'), to: Date.parse('2026-09-12T12:00:00Z') }} />)
    expect(screen.getAllByRole('img')).toHaveLength(4)
    for (const chart of screen.getAllByRole('img')) expect(chart).toHaveAccessibleName(/en 2 lecturas/)
  })

  it('shows the whole series without a range', () => {
    render(<ChartGrid readings={readings} events={[]} />)
    for (const chart of screen.getAllByRole('img')) expect(chart).toHaveAccessibleName(/en 4 lecturas/)
  })

  it('highlights anomaly windows on every chart', () => {
    render(<ChartGrid readings={readings} events={[]} windows={[{ start: '2026-09-12T11:00:00Z', end: '2026-09-12T12:00:00Z', severity: 'HIGH', label: 'Cambio de nivel al alza' }]} />)
    for (const chart of screen.getAllByRole('img')) expect(chart).toHaveAccessibleName(/1 ventana de anomalía resaltada: Cambio de nivel al alza, severidad alta/)
    expect(screen.getAllByText('Ventana de anomalía')).toHaveLength(4)
  })

  it('shows the empty message on every chart without readings', () => {
    render(<ChartGrid readings={[]} events={[]} />)
    expect(screen.getAllByText('Aún no hay lecturas para graficar.')).toHaveLength(4)
  })
})
