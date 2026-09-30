import { typeLabel, typeToken, type AnomalyType } from '../../lib/status'
import { Badge } from '../atoms/Badge'

export function TypeBadge({ type }: { type: AnomalyType }) {
  return <Badge tone={typeToken(type)}>{typeLabel(type)}</Badge>
}
