const PLACEHOLDER = '—'

const kwhFormat = new Intl.NumberFormat('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
const percentFormat = new Intl.NumberFormat('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
// Months come from en-US ("Sep"); en-GB abbreviates September as "Sept" in recent ICU builds.
const monthFormat = new Intl.DateTimeFormat('en-US', { month: 'short', timeZone: 'UTC' })
const timeFormat = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit', hourCycle: 'h23', timeZone: 'UTC' })

const dateOf = (date: Date): string => `${date.getUTCDate()} ${monthFormat.format(date)} ${date.getUTCFullYear()}`

export function formatKwh(value: number): string {
  return `${kwhFormat.format(value)} kWh`
}

/** Formats a percentage already expressed in percent points (110.5 -> "+110.5%"). */
export function formatSignedPercent(value: number): string {
  const rounded = Math.round(value * 10) / 10
  const sign = rounded > 0 ? '+' : ''
  return `${sign}${percentFormat.format(rounded)}%`
}

/** Formats a 0..1 confidence ratio as a whole percent. */
export function formatConfidence(ratio: number): string {
  return `${Math.round(ratio * 100)}%`
}

export function formatDate(iso: string | null | undefined): string {
  return iso ? dateOf(new Date(iso)) : PLACEHOLDER
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return PLACEHOLDER
  const date = new Date(iso)
  return `${dateOf(date)}, ${timeFormat.format(date)} UTC`
}

/** Day and month (UTC) of an epoch-millisecond timestamp, for chart axis ticks. */
export function formatAxisTick(ms: number): string {
  const date = new Date(ms)
  return `${date.getUTCDate()} ${monthFormat.format(date)}`
}

export function formatNumber(value: number, decimals: number): string {
  return new Intl.NumberFormat('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals }).format(value)
}

const timeOf = (date: Date): string => timeFormat.format(date)

/** A time window in UTC: one date with a time range inside a day, both ends otherwise. */
export function formatWindow(startIso: string, endIso: string): string {
  const start = new Date(startIso)
  const end = new Date(endIso)
  if (start.getTime() === end.getTime()) return formatDateTime(startIso)
  if (dateOf(start) === dateOf(end)) return `${dateOf(start)}, ${timeOf(start)}–${timeOf(end)} UTC`
  return `${formatDateTime(startIso)} – ${formatDateTime(endIso)}`
}

/** Axis tick: at least `minDecimals`, at most `maxDecimals` fraction digits. */
export function formatAxisNumber(value: number, minDecimals: number, maxDecimals: number): string {
  return new Intl.NumberFormat('en-US', { minimumFractionDigits: minDecimals, maximumFractionDigits: maxDecimals }).format(value)
}
