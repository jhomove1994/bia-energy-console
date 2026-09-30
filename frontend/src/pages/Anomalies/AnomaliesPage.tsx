import { useSearchParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import type { AnomalyFilters } from '../../api/types'
import { useAnalysisJob } from '../../hooks/useAnalysisJob'
import { useAnomalies } from '../../hooks/useAnomalies'
import { AnomaliesView } from './AnomaliesView'

function describe(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Unexpected error'
}

/** Container: the filters live in the URL search string (`type`, `severity`, `meter_id`), so views are shareable links. */
export function AnomaliesPage() {
  const [params, setParams] = useSearchParams()
  const type = params.get('type') ?? ''
  const severity = params.get('severity') ?? ''
  const meterId = params.get('meter_id') ?? ''

  const filters: AnomalyFilters = { type, severity, meter_id: meterId }
  const anomalies = useAnomalies(filters)
  const job = useAnalysisJob()

  const update = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  return (
    <AnomaliesView
      state={anomalies.isPending ? 'loading' : anomalies.isError ? 'error' : 'ready'}
      list={anomalies.data}
      filters={{ type, severity, meterId }}
      errorMessage={anomalies.isError ? describe(anomalies.error) : undefined}
      onTypeChange={(value) => update('type', value)}
      onSeverityChange={(value) => update('severity', value)}
      onClearFilters={() => setParams(new URLSearchParams(), { replace: true })}
      onRetry={() => void anomalies.refetch()}
      onRun={job.start}
      starting={job.starting}
      busy={job.active}
      analysis={job.analysis}
      runError={job.error ? describe(job.error) : null}
    />
  )
}
