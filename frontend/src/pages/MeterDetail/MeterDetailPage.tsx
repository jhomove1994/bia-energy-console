import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useAnalysisJob } from '../../hooks/useAnalysisJob'
import { useMeterEvents, useMeterReadings } from '../../hooks/useMeterReadings'
import { useMeter } from '../../hooks/useMeters'
import { MeterDetailView, type SectionState } from './MeterDetailView'

function describe(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Unexpected error'
}

function section<T>(q: { isPending: boolean; isError: boolean; error: unknown; data: T | undefined }): SectionState<T> {
  if (q.isPending) return { status: 'loading' }
  if (q.isError) return { status: 'error', message: describe(q.error) }
  return { status: 'ready', data: q.data as T }
}

/** Container: the three queries start together (parallel), the view only receives props. */
export function MeterDetailPage() {
  const { meterId = '' } = useParams()
  const meter = useMeter(meterId)
  const readings = useMeterReadings(meterId)
  const events = useMeterEvents(meterId)
  const job = useAnalysisJob()

  const notFound = meter.error instanceof ApiError && meter.error.status === 404

  return (
    <MeterDetailView
      meterId={meterId}
      state={meter.isPending ? 'loading' : notFound ? 'not-found' : meter.isError ? 'error' : 'ready'}
      meter={meter.data}
      errorMessage={meter.isError ? describe(meter.error) : undefined}
      readings={section(readings)}
      events={section(events)}
      onRetry={() => {
        if (meter.isError) void meter.refetch()
        if (readings.isError) void readings.refetch()
        if (events.isError) void events.refetch()
      }}
      onRun={job.start}
      starting={job.starting}
      busy={job.active}
      analysis={job.analysis}
      runError={job.error ? describe(job.error) : null}
    />
  )
}
