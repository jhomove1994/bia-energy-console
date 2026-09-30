import type { EvidenceItem } from '../../api/types'
import { Card } from '../atoms/Card'
import { EvidenceRow } from '../molecules/EvidenceRow'

const th = 'px-4 py-3 text-xs font-semibold uppercase tracking-wide text-muted'

/** The numbers behind a finding: what was measured, what was expected, what was observed. The related event has its own card. */
export function EvidenceTable({ items }: { items: EvidenceItem[] }) {
  const rows = items.filter((item) => item.metric !== 'event')
  if (rows.length === 0) return <p className="text-sm text-muted">No numeric evidence was recorded for this finding.</p>
  return (
    <Card padded={false} className="overflow-x-auto">
      <table className="w-full min-w-[32rem] border-collapse text-sm">
        <thead className="border-b border-border bg-bg/60">
          <tr>
            <th scope="col" className={`${th} text-left`}>
              Metric
            </th>
            <th scope="col" className={`${th} text-right`}>
              Baseline
            </th>
            <th scope="col" className={`${th} text-right`}>
              Observed
            </th>
            <th scope="col" className={`${th} text-right`}>
              Delta
            </th>
            <th scope="col" className={`${th} text-left`}>
              Unit
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {rows.map((item) => (
            <EvidenceRow key={item.metric} item={item} />
          ))}
        </tbody>
      </table>
    </Card>
  )
}
