import type { MeterEvent } from '../../api/types'
import { eventLabel } from '../../lib/chartData'
import { formatDateTime } from '../../lib/format'
import { Badge } from '../atoms/Badge'
import { Card } from '../atoms/Card'

/** An UNKNOWN event is what the logs say when nothing explains the change, so it counts as no event. */
export const explainsChange = (event: MeterEvent | null): event is MeterEvent => event !== null && event.type !== 'UNKNOWN'

export function RelatedEventCard({ event }: { event: MeterEvent | null }) {
  return (
    <Card role="region" aria-labelledby="related-event-heading" className="flex flex-col gap-2">
      <h2 id="related-event-heading" className="text-base font-semibold text-text">
        Related event
      </h2>
      {explainsChange(event) ? (
        <>
          <div className="flex flex-wrap items-center gap-2">
            <Badge>{eventLabel(event.type)}</Badge>
            <time dateTime={event.timestamp} className="tabular text-xs font-medium text-muted">
              {formatDateTime(event.timestamp)}
            </time>
          </div>
          <p className="text-sm text-text">{event.description}</p>
        </>
      ) : (
        <p className="text-sm text-muted">No operational event explains this change.</p>
      )}
    </Card>
  )
}
