import { render, screen, within } from '@testing-library/react'
import type { MeterEvent } from '../../api/types'
import { EventTimeline } from './EventTimeline'

const events: MeterEvent[] = [
  { id: 2, timestamp: '2026-09-12T14:00:00Z', type: 'UNKNOWN', description: 'Unexplained rise' },
  { id: 1, timestamp: '2026-09-08T00:00:00Z', type: 'SCHEDULED_OUTAGE', description: 'Scheduled maintenance outage for 12 hours' },
]

describe('EventTimeline', () => {
  it('lists events chronologically with time, type and description', () => {
    render(<EventTimeline events={events} />)
    const items = screen.getAllByRole('listitem')
    expect(items).toHaveLength(2)
    expect(within(items[0]!).getByText('8 Sep 2026, 00:00 UTC')).toBeInTheDocument()
    expect(within(items[0]!).getByText('Scheduled outage')).toBeInTheDocument()
    expect(within(items[0]!).getByText('Scheduled maintenance outage for 12 hours')).toBeInTheDocument()
    expect(within(items[1]!).getByText('Unknown')).toBeInTheDocument()
  })

  it('says so when the meter has no events', () => {
    render(<EventTimeline events={[]} />)
    expect(screen.getByText('No events recorded for this meter.')).toBeInTheDocument()
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
  })
})
