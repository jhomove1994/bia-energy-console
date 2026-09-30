import { statusLabel, statusToken, type MeterStatus } from '../../lib/status'
import { Badge } from '../atoms/Badge'

export function StatusPill({ status }: { status: MeterStatus }) {
  return (
    <Badge tone={statusToken(status)} dot>
      {statusLabel(status)}
    </Badge>
  )
}
