import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { startAnalysis } from '../../api/analysis'
import { ApiError } from '../../api/client'
import { DASHBOARD_KEY, useDashboard, type DashboardWatch } from '../../hooks/useDashboard'
import { DashboardView } from './DashboardView'

const WATCH_WINDOW_MS = 30_000

function describe(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Unexpected error'
}

/*
 * Temporary run trigger: it starts the analysis and watches the dashboard until a newer
 * completed analysis appears. Live stage progress replaces this in the analysis-progress slice.
 */
export function DashboardPage() {
  const queryClient = useQueryClient()
  const [watch, setWatch] = useState<DashboardWatch | null>(null)
  const dashboard = useDashboard(watch)

  const run = useMutation({
    mutationFn: startAnalysis,
    onMutate: () => setWatch(null),
    onSuccess: () => {
      setWatch({ fromAnalysisId: dashboard.data?.last_analysis?.id ?? null, until: Date.now() + WATCH_WINDOW_MS })
      return queryClient.invalidateQueries({ queryKey: DASHBOARD_KEY })
    },
  })

  const summary = dashboard.data
  const waiting = watch !== null && Date.now() <= watch.until && (summary?.last_analysis?.id ?? null) === watch.fromAnalysisId

  return (
    <DashboardView
      state={dashboard.isPending ? 'loading' : dashboard.isError ? 'error' : 'ready'}
      summary={summary}
      errorMessage={dashboard.isError ? describe(dashboard.error) : undefined}
      onRetry={() => void dashboard.refetch()}
      onRun={() => run.mutate()}
      starting={run.isPending}
      waiting={waiting}
      runError={run.isError ? describe(run.error) : null}
    />
  )
}
