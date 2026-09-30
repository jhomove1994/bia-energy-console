import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useAnalysisJob } from '../../hooks/useAnalysisJob'
import { useAnomalies } from '../../hooks/useAnomalies'
import { useMeterEvents, useMeterReadings } from '../../hooks/useMeterReadings'
import { useMeter } from '../../hooks/useMeters'
import { toSection } from '../../lib/section'
import { MeterDetailView } from './MeterDetailView'

function describe(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Unexpected error'
}

/** Container: the three queries start together (parallel), the view only receives props. */
export function MeterDetailPage() {
  const { meterId = '' } = useParams()
  const meter = useMeter(meterId)
  const readings = useMeterReadings(meterId)
  const events = useMeterEvents(meterId)
  const anomalies = useAnomalies({ meter_id: meterId }, { keepPrevious: false })
  const job = useAnalysisJob()

  const notFound = meter.error instanceof ApiError && meter.error.status === 404

  return (
    <MeterDetailView
      meterId={meterId}
      state={meter.isPending ? 'loading' : notFound ? 'not-found' : meter.isError ? 'error' : 'ready'}
      meter={meter.data}
      errorMessage={meter.isError ? describe(meter.error) : undefined}
      readings={toSection(readings)}
      events={toSection(events)}
      anomalies={toSection({ isPending: anomalies.isPending, isError: anomalies.isError, error: anomalies.error, data: anomalies.data?.items })}
      onRetry={() => {
        if (meter.isError) void meter.refetch()
        if (readings.isError) void readings.refetch()
        if (events.isError) void events.refetch()
        if (anomalies.isError) void anomalies.refetch()
      }}
      onRun={job.start}
      starting={job.starting}
      busy={job.active}
      analysis={job.analysis}
      runError={job.error ? describe(job.error) : null}
    />
  )
}
