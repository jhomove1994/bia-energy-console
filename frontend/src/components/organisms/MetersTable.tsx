import { Link } from 'react-router-dom'
import type { MeterRow } from '../../api/types'
import { formatKwh, formatSignedPercent } from '../../lib/format'
import { variationTone } from '../../lib/status'
import { variationClass } from '../../lib/tone'
import { Card } from '../atoms/Card'
import { StatusPill } from '../molecules/StatusPill'

const th = 'px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-muted'
const thRight = `${th} text-right`

export function MetersTable({ rows }: { rows: MeterRow[] }) {
  return (
    <Card padded={false} className="overflow-x-auto">
      <table className="w-full min-w-[34rem] border-collapse text-sm">
        <thead className="border-b border-border bg-bg/60">
          <tr>
            <th scope="col" className={th}>
              Medidor
            </th>
            <th scope="col" className={thRight}>
              Consumo total
            </th>
            <th scope="col" className={thRight}>
              Variación diaria
            </th>
            <th scope="col" className={th}>
              Estado
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {rows.map((row) => (
            <tr key={row.meter_id} className="relative transition-colors duration-fast ease-console hover:bg-bg/70">
              <th scope="row" className="px-4 py-3 text-left font-semibold">
                {/* The link stretches over the whole row, so the entire row opens the detail. */}
                <Link
                  to={`/meters/${encodeURIComponent(row.meter_id)}`}
                  className="text-primary-strong after:absolute after:inset-0 after:content-[''] hover:underline"
                >
                  {row.meter_id}
                </Link>
              </th>
              <td className="tabular px-4 py-3 text-right">{formatKwh(row.total_kwh)}</td>
              <td className={`tabular px-4 py-3 text-right ${variationClass[variationTone(row.variation_pct)]}`}>
                {row.variation_pct === null ? <span className="text-muted">—</span> : formatSignedPercent(row.variation_pct)}
              </td>
              <td className="px-4 py-3">
                <StatusPill status={row.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )
}
