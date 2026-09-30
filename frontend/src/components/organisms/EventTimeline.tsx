import type { MeterEvent } from '../../api/types'
import { formatDateTime } from '../../lib/format'
import { eventLabel } from '../../lib/labels'
import { Badge } from '../atoms/Badge'
import { Card } from '../atoms/Card'

export function EventTimeline({ events }: { events: MeterEvent[] }) {
  const ordered = [...events].sort((a, b) => Date.parse(a.timestamp) - Date.parse(b.timestamp))
  return (
    <Card className="flex flex-col gap-3">
      <h2 className="text-base font-semibold text-text">Eventos</h2>
      {ordered.length === 0 ? (
        <p className="text-sm text-muted">No hay eventos registrados para este medidor.</p>
      ) : (
        <ol className="flex flex-col border-l-2 border-border pl-4">
          {ordered.map((event) => (
            <li key={event.id} className="relative flex flex-col gap-1 py-2">
              <span aria-hidden="true" className="absolute -left-[1.4rem] top-3.5 h-2.5 w-2.5 rounded-full border-2 border-surface bg-primary" />
              <div className="flex flex-wrap items-center gap-2">
                <time dateTime={event.timestamp} className="tabular text-xs font-medium text-muted">
                  {formatDateTime(event.timestamp)}
                </time>
                <Badge>{eventLabel(event.type)}</Badge>
              </div>
              <p className="text-sm text-text">{event.description}</p>
            </li>
          ))}
        </ol>
      )}
    </Card>
  )
}
