import type { EvidenceItem } from '../../api/types'
import { formatNumber, formatSignedPercent } from '../../lib/format'
import { metricLabel, unitLabel } from '../../lib/labels'
import { variationTone } from '../../lib/status'
import { variationClass } from '../../lib/tone'

const PLACEHOLDER = <span className="text-muted">—</span>

/** Whole numbers for counts and hours, two decimals for ratios and correlations, one for physical quantities. */
function decimalsFor(unit: string): number {
  if (unit === 'h' || unit === 'count') return 0
  if (unit === '' || unit === 'corr') return 2
  return 1
}

export function EvidenceRow({ item }: { item: EvidenceItem }) {
  const decimals = decimalsFor(item.unit)
  return (
    <tr>
      <th scope="row" className="px-4 py-3 text-left font-medium text-text">
        {metricLabel(item.metric, item.label)}
        {item.detail && <span className="mt-0.5 block text-xs font-normal text-muted">{item.detail}</span>}
      </th>
      <td className="tabular px-4 py-3 text-right text-muted">{item.baseline === null ? PLACEHOLDER : formatNumber(item.baseline, decimals)}</td>
      <td className="tabular px-4 py-3 text-right font-semibold text-text">{formatNumber(item.observed, decimals)}</td>
      <td className={`tabular px-4 py-3 text-right ${item.delta_pct === null ? '' : variationClass[variationTone(item.delta_pct)]}`}>
        {item.delta_pct === null ? PLACEHOLDER : formatSignedPercent(item.delta_pct)}
      </td>
      <td className="px-4 py-3 text-muted">{unitLabel(item.unit) || PLACEHOLDER}</td>
    </tr>
  )
}
