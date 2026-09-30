import { severityLabel } from '../../lib/labels'
import { severityToken, type Severity } from '../../lib/status'
import { Badge } from '../atoms/Badge'

export function SeverityBadge({ severity }: { severity: Severity }) {
  return (
    <Badge tone={severityToken(severity)} dot>
      {severityLabel(severity)}
    </Badge>
  )
}
