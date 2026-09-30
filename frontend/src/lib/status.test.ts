import { actionLabel, variationTone, severityLabel, severityToken, statusLabel, statusToken, typeLabel, typeToken } from './status'

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

describe('variationTone', () => {
  it('flags large moves in either direction and keeps small ones quiet', () => {
    expect(variationTone(103.7)).toBe('high')
    expect(variationTone(-40)).toBe('high')
    expect(variationTone(12)).toBe('medium')
    expect(variationTone(-10)).toBe('medium')
    expect(variationTone(1.1)).toBe('none')
    expect(variationTone(-0.3)).toBe('none')
    expect(variationTone(null)).toBe('none')
  })
})

describe('actionLabel', () => {
  it('names each action code in a few words', () => {
    expect(actionLabel('INSPECT_LOAD_INCREASE')).toBe('Inspect load increase')
    expect(actionLabel('OPEN_DQ_TICKET')).toBe('Open data-quality ticket')
    expect(actionLabel('ACKNOWLEDGE_EXPECTED')).toBe('Acknowledge as expected')
  })

  it('humanizes a code it does not know', () => {
    expect(actionLabel('CALIBRATE_SENSOR')).toBe('Calibrate sensor')
  })
})
