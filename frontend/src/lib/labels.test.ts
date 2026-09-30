import {
  actionLabel,
  analysisStatusLabel,
  apiErrorMessage,
  confidenceLabel,
  eventLabel,
  kindLabel,
  metricLabel,
  severityLabel,
  stageLabel,
  STAGE_ORDER,
  statusLabel,
  typeLabel,
  unitLabel,
} from './labels'

describe('enum labels', () => {
  it('names the anomaly types in Spanish', () => {
    expect(typeLabel('REAL_ANOMALY')).toBe('Anomalía real')
    expect(typeLabel('EXPLAINABLE_ANOMALY')).toBe('Anomalía explicable')
    expect(typeLabel('FALSE_POSITIVE')).toBe('Falso positivo')
    expect(typeLabel('DATA_QUALITY')).toBe('Calidad de datos')
  })

  it('names severities, meter statuses and analysis statuses', () => {
    expect([severityLabel('HIGH'), severityLabel('MEDIUM'), severityLabel('LOW')]).toEqual(['Alta', 'Media', 'Baja'])
    expect([statusLabel('OK'), statusLabel('Alert'), statusLabel('Critical')]).toEqual(['Normal', 'Alerta', 'Crítico'])
    expect(['pending', 'running', 'done', 'failed'].map((s) => analysisStatusLabel(s as never))).toEqual(['Pendiente', 'En ejecución', 'Completado', 'Fallido'])
  })

  it('names the seven pipeline stages in execution order', () => {
    expect(STAGE_ORDER.map(stageLabel)).toEqual(['Lecturas', 'Baseline', 'Detección', 'Correlación', 'Eventos', 'Explicación', 'Recomendación'])
  })

  it('names event types and falls back to Desconocido', () => {
    expect(eventLabel('OPERATIONAL_CHANGE')).toBe('Cambio operativo')
    expect(eventLabel('SCHEDULED_OUTAGE')).toBe('Parada programada')
    expect(eventLabel('data_quality')).toBe('Calidad de datos')
    expect(eventLabel('UNKNOWN')).toBe('Desconocido')
    expect(eventLabel('FIRMWARE')).toBe('Desconocido')
  })

  it('names confidence levels', () => {
    expect([confidenceLabel('High'), confidenceLabel('Medium'), confidenceLabel('Low')]).toEqual(['alta', 'media', 'baja'])
  })

  it('names anomaly kinds', () => {
    expect(kindLabel('LEVEL_SHIFT_UP', 'Level shift up')).toBe('Cambio de nivel al alza')
    expect(kindLabel('DATA_INCONSISTENCY', '')).toBe('Inconsistencia de datos')
    expect(kindLabel('PATTERN_CHANGE', '')).toBe('Cambio de patrón')
    expect(kindLabel('SOMETHING_NEW', 'Something new')).toBe('Something new')
  })
})

describe('evidence labels', () => {
  it('names each evidence metric and keeps the API label for unknown ones', () => {
    expect(metricLabel('consumption_kwh', 'x')).toBe('Consumo horario (promedio de la ventana)')
    expect(metricLabel('power_factor', 'x')).toBe('Factor de potencia (promedio de la ventana)')
    expect(metricLabel('voltage_jumps', 'x')).toBe('Saltos de voltaje')
    expect(metricLabel('event', 'x')).toBe('Evento relacionado')
    expect(metricLabel('brand_new', 'Brand new')).toBe('Brand new')
  })

  it('names the units that are words', () => {
    expect(unitLabel('count')).toBe('cantidad')
    expect(unitLabel('corr')).toBe('correlación')
    expect(unitLabel('kWh/h')).toBe('kWh/h')
    expect(unitLabel('')).toBe('')
  })
})

describe('actionLabel', () => {
  it('names each action code in a few words', () => {
    expect(actionLabel('INSPECT_LOAD_INCREASE')).toBe('Inspeccionar aumento de carga')
    expect(actionLabel('CHECK_SUPPLY_LOSS')).toBe('Verificar pérdida de suministro')
    expect(actionLabel('REVIEW_TRANSIENT')).toBe('Revisar transitorio')
    expect(actionLabel('REVIEW_SCHEDULE')).toBe('Revisar horario')
    expect(actionLabel('OPEN_DQ_TICKET')).toBe('Abrir ticket de calidad de datos')
    expect(actionLabel('UPDATE_BASELINE')).toBe('Actualizar baseline')
    expect(actionLabel('ACKNOWLEDGE_EXPECTED')).toBe('Reconocer como esperado')
  })

  it('humanizes a code it does not know', () => {
    expect(actionLabel('CALIBRATE_SENSOR')).toBe('Calibrate sensor')
  })
})

describe('apiErrorMessage', () => {
  it('explains each API error code in Spanish', () => {
    expect(apiErrorMessage('invalid_credentials', 401)).toBe('Usuario o contraseña incorrectos.')
    expect(apiErrorMessage('unauthorized', 401)).toBe('Su sesión expiró. Inicie sesión de nuevo.')
    expect(apiErrorMessage('not_found', 404)).toBe('No se encontró el recurso solicitado.')
    expect(apiErrorMessage('network', 0)).toBe('No se pudo conectar con el servidor. Revise su conexión e intente de nuevo.')
  })

  it('falls back by status for unknown codes', () => {
    expect(apiErrorMessage('http_502', 502)).toBe('El servidor no pudo completar la solicitud (estado 502).')
  })
})
