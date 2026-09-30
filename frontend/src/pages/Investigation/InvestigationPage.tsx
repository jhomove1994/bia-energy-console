import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useAnomaly } from '../../hooks/useAnomalies'
import { useMeterReadings } from '../../hooks/useMeterReadings'
import { describeError, toSection } from '../../lib/section'
import { InvestigationView } from './InvestigationView'

/** Container: the readings of the anomaly's meter load once the anomaly says which meter that is. */
export function InvestigationPage() {
  const { id = '' } = useParams()
  const anomaly = useAnomaly(id)
  const readings = useMeterReadings(anomaly.data?.meter_id ?? '')

  const notFound = anomaly.error instanceof ApiError && anomaly.error.status === 404

  return (
    <InvestigationView
      state={anomaly.isPending ? 'loading' : notFound ? 'not-found' : anomaly.isError ? 'error' : 'ready'}
      anomaly={anomaly.data}
      errorMessage={anomaly.isError ? describeError(anomaly.error) : undefined}
      readings={toSection(readings)}
      onRetry={() => {
        if (anomaly.isError) void anomaly.refetch()
        if (readings.isError) void readings.refetch()
      }}
    />
  )
}
