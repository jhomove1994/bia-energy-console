import { formatAxisTick, formatNumber, formatConfidence, formatDate, formatDateTime, formatAxisNumber, formatKwh, formatSignedPercent, formatWindow } from './format'

describe('formatKwh', () => {
  it('groups thousands and keeps one decimal', () => {
    expect(formatKwh(12345.678)).toBe('12,345.7 kWh')
  })

  it('pads a whole number to one decimal', () => {
    expect(formatKwh(42)).toBe('42.0 kWh')
    expect(formatKwh(0)).toBe('0.0 kWh')
  })
})

describe('formatSignedPercent', () => {
  it('shows an explicit plus for increases', () => {
    expect(formatSignedPercent(110.46)).toBe('+110.5%')
  })

  it('shows a minus for decreases and no sign for zero', () => {
    expect(formatSignedPercent(-3.04)).toBe('-3.0%')
    expect(formatSignedPercent(0)).toBe('0.0%')
  })
})

describe('formatConfidence', () => {
  it('renders a 0..1 ratio as a whole percent', () => {
    expect(formatConfidence(0.914)).toBe('91%')
    expect(formatConfidence(0.75)).toBe('75%')
  })
})

describe('date formats', () => {
  const iso = '2026-09-12T14:05:00Z'

  it('formats a date in UTC', () => {
    expect(formatDate(iso)).toBe('12 Sep 2026')
  })

  it('formats a date and time in UTC', () => {
    expect(formatDateTime(iso)).toBe('12 Sep 2026, 14:05 UTC')
  })

  it('renders a placeholder for a missing value', () => {
    expect(formatDate(null)).toBe('—')
    expect(formatDateTime(undefined)).toBe('—')
  })
})

describe('formatAxisTick', () => {
  it('shows the UTC day and month for a timestamp in epoch milliseconds', () => {
    expect(formatAxisTick(Date.parse('2026-09-12T14:00:00Z'))).toBe('12 Sep')
    expect(formatAxisTick(Date.parse('2026-09-30T23:30:00Z'))).toBe('30 Sep')
  })
})

describe('formatNumber', () => {
  it('keeps the requested decimals with thousands separators', () => {
    expect(formatNumber(2180.456, 1)).toBe('2,180.5')
    expect(formatNumber(0.951, 2)).toBe('0.95')
    expect(formatNumber(230, 0)).toBe('230')
  })
})

describe('formatWindow', () => {
  it('shows one date and a time range when the window sits inside a day', () => {
    expect(formatWindow('2026-09-12T14:00:00Z', '2026-09-12T20:00:00Z')).toBe('12 Sep 2026, 14:00–20:00 UTC')
  })

  it('shows both ends when the window spans days', () => {
    expect(formatWindow('2026-09-12T14:00:00Z', '2026-09-14T23:00:00Z')).toBe('12 Sep 2026, 14:00 UTC – 14 Sep 2026, 23:00 UTC')
  })

  it('uses a single time for a one-reading window', () => {
    expect(formatWindow('2026-09-12T14:00:00Z', '2026-09-12T14:00:00Z')).toBe('12 Sep 2026, 14:00 UTC')
  })
})

describe('formatAxisNumber', () => {
  it('rounds to whole numbers when no decimals are allowed', () => {
    expect(formatAxisNumber(525, 0, 0)).toBe('525')
    expect(formatAxisNumber(524.6, 0, 0)).toBe('525')
    expect(formatAxisNumber(1525, 0, 0)).toBe('1,525')
  })

  it('drops a trailing decimal that adds nothing when decimals are optional', () => {
    expect(formatAxisNumber(230, 0, 1)).toBe('230')
    expect(formatAxisNumber(229.5, 0, 1)).toBe('229.5')
    expect(formatAxisNumber(229.54, 0, 1)).toBe('229.5')
  })

  it('keeps a fixed number of decimals when the minimum says so', () => {
    expect(formatAxisNumber(0.9, 2, 2)).toBe('0.90')
    expect(formatAxisNumber(1, 2, 2)).toBe('1.00')
  })
})
