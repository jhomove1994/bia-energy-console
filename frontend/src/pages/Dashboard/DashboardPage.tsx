import { ApiError } from '../../api/client'
import { useRunAnalysis } from '../../hooks/useRunAnalysis'
import { DashboardView } from './DashboardView'

function describe(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Unexpected error'
}

export function DashboardPage() {
  const { dashboard, start, starting, waiting, error } = useRunAnalysis({ observeDashboard: true })

  return (
    <DashboardView
      state={dashboard.isPending ? 'loading' : dashboard.isError ? 'error' : 'ready'}
      summary={dashboard.data}
      errorMessage={dashboard.isError ? describe(dashboard.error) : undefined}
      onRetry={() => void dashboard.refetch()}
      onRun={start}
      starting={starting}
      waiting={waiting}
      runError={error ? describe(error) : null}
    />
  )
}
