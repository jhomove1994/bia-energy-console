import { ApiError } from '../../api/client'
import { useAnalysisJob } from '../../hooks/useAnalysisJob'
import { useDashboard } from '../../hooks/useDashboard'
import { DashboardView } from './DashboardView'

function describe(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Error inesperado'
}

export function DashboardPage() {
  const dashboard = useDashboard()
  const job = useAnalysisJob()

  return (
    <DashboardView
      state={dashboard.isPending ? 'loading' : dashboard.isError ? 'error' : 'ready'}
      summary={dashboard.data}
      errorMessage={dashboard.isError ? describe(dashboard.error) : undefined}
      onRetry={() => void dashboard.refetch()}
      onRun={job.start}
      starting={job.starting}
      busy={job.active}
      analysis={job.analysis}
      runError={job.error ? describe(job.error) : null}
    />
  )
}
