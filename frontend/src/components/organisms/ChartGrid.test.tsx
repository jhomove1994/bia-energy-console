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
    for (const name of [/Consumption \(kWh\)/, /Voltage \(V\)/, /Current \(A\)/, /Power factor \(PF\)/]) {
      expect(screen.getByRole('img', { name })).toBeInTheDocument()
    }
  })

  it('shows the empty message on every chart without readings', () => {
    render(<ChartGrid readings={[]} events={[]} />)
    expect(screen.getAllByText('No readings to chart yet.')).toHaveLength(4)
  })
})
