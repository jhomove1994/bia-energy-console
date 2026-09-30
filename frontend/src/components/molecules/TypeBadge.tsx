import { typeLabel } from '../../lib/labels'
import { typeToken, type AnomalyType } from '../../lib/status'
import { Badge } from '../atoms/Badge'

export function TypeBadge({ type }: { type: AnomalyType }) {
  return <Badge tone={typeToken(type)}>{typeLabel(type)}</Badge>
}
