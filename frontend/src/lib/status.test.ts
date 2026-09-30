import { severityLabel, severityToken, statusLabel, statusToken, typeLabel, typeToken } from './status'

describe('severityToken', () => {
  it('maps each severity to its design token', () => {
    expect(severityToken('HIGH')).toBe('sev-high')
    expect(severityToken('MEDIUM')).toBe('sev-medium')
    expect(severityToken('LOW')).toBe('sev-low')
  })
})

describe('typeToken', () => {
  it('maps each anomaly type to its design token', () => {
    expect(typeToken('REAL_ANOMALY')).toBe('type-real')
    expect(typeToken('DATA_QUALITY')).toBe('type-dq')
    expect(typeToken('EXPLAINABLE_ANOMALY')).toBe('type-expl')
    expect(typeToken('FALSE_POSITIVE')).toBe('type-fp')
  })
})

describe('statusToken', () => {
  it('maps each meter status to its design token', () => {
    expect(statusToken('OK')).toBe('status-ok')
    expect(statusToken('Alert')).toBe('status-alert')
    expect(statusToken('Critical')).toBe('status-critical')
  })
})

describe('labels', () => {
  it('humanizes the API enums', () => {
    expect(typeLabel('REAL_ANOMALY')).toBe('Real anomaly')
    expect(typeLabel('EXPLAINABLE_ANOMALY')).toBe('Explainable')
    expect(typeLabel('DATA_QUALITY')).toBe('Data quality')
    expect(typeLabel('FALSE_POSITIVE')).toBe('False positive')
    expect(severityLabel('HIGH')).toBe('High')
    expect(statusLabel('Critical')).toBe('Critical')
  })
})
