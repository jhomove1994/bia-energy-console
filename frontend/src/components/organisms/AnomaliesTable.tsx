import { Link } from 'react-router-dom'
import type { Anomaly } from '../../api/types'
import { formatWindow } from '../../lib/format'
import { actionLabel, kindLabel } from '../../lib/labels'
import { Card } from '../atoms/Card'
import { ConfidenceMeter } from '../molecules/ConfidenceMeter'
import { SeverityBadge } from '../molecules/SeverityBadge'
import { TypeBadge } from '../molecules/TypeBadge'

export interface AnomaliesTableProps {
  /** Already ranked by priority. */
  anomalies: Anomaly[]
  /** Hide the meter column when every row belongs to the same meter. */
  showMeter?: boolean
}

const th = 'px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-muted'

export function AnomaliesTable({ anomalies, showMeter = true }: AnomaliesTableProps) {
  return (
    <Card padded={false} className="overflow-x-auto">
      <table className="w-full min-w-[58rem] border-collapse text-sm">
        <thead className="border-b border-border bg-bg/60">
          <tr>
            <th scope="col" className={th}>
              Prioridad
            </th>
            {showMeter && (
              <th scope="col" className={th}>
                Medidor
              </th>
            )}
            <th scope="col" className={th}>
              Anomalía
            </th>
            <th scope="col" className={th}>
              Tipo
            </th>
            <th scope="col" className={th}>
              Severidad
            </th>
            <th scope="col" className={th}>
              Confianza
            </th>
            <th scope="col" className={th}>
              Ventana detectada
            </th>
            <th scope="col" className={th}>
              Acción recomendada
            </th>
            <th scope="col" className={th}>
              Investigar
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {anomalies.map((a) => (
            <tr key={a.id} className="relative transition-colors duration-fast ease-console hover:bg-bg/70">
              <td className="tabular px-4 py-3 font-semibold text-muted">#{a.priority_rank}</td>
              {showMeter && (
                <th scope="row" className="px-4 py-3 text-left font-semibold">
                  {a.meter_id}
                </th>
              )}
              {showMeter ? (
                <td className="px-4 py-3">{kindLabel(a.kind, a.anomaly)}</td>
              ) : (
                <th scope="row" className="px-4 py-3 text-left font-semibold">
                  {kindLabel(a.kind, a.anomaly)}
                </th>
              )}
              <td className="px-4 py-3">
                <TypeBadge type={a.type} />
              </td>
              <td className="px-4 py-3">
                <SeverityBadge severity={a.severity} />
              </td>
              <td className="px-4 py-3">
                <ConfidenceMeter value={a.confidence} />
              </td>
              <td className="tabular px-4 py-3 text-muted">{formatWindow(a.window_start, a.window_end)}</td>
              <td className="px-4 py-3">{actionLabel(a.action_code)}</td>
              <td className="px-4 py-3">
                {/* The link stretches over the whole row, so the entire row opens the investigation. */}
                <Link
                  to={`/anomalies/${encodeURIComponent(a.id)}`}
                  aria-label={`Investigar ${a.meter_id}`}
                  className="font-semibold text-primary-strong after:absolute after:inset-0 after:content-[''] hover:underline"
                >
                  Investigar
                </Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )
}
