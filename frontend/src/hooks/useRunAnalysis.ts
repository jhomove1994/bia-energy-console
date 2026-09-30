import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { startAnalysis } from '../api/analysis'
import { getDashboardSummary } from '../api/dashboard'
import { DASHBOARD_KEY, useDashboard, type DashboardWatch } from './useDashboard'

const WATCH_WINDOW_MS = 30_000

/*
 * Temporary run trigger shared by the dashboard and the meter detail: it starts the analysis and
 * watches the dashboard summary until a newer completed analysis appears, then refreshes the meter
 * queries. Live stage progress replaces this in the analysis-progress slice.
 */
export function useRunAnalysis({ observeDashboard }: { observeDashboard: boolean }) {
  const queryClient = useQueryClient()
  const [watch, setWatch] = useState<DashboardWatch | null>(null)
  const dashboard = useDashboard(watch, observeDashboard || watch !== null)

  const run = useMutation({
    mutationFn: async () => {
      // The id of the last completed analysis before this run tells us when a new one lands.
      const before = await queryClient.fetchQuery({ queryKey: DASHBOARD_KEY, queryFn: getDashboardSummary, staleTime: 10_000 })
      await startAnalysis()
      return before.last_analysis?.id ?? null
    },
    onMutate: () => setWatch(null),
    onSuccess: (fromAnalysisId) => {
      setWatch({ fromAnalysisId, until: Date.now() + WATCH_WINDOW_MS })
      return queryClient.invalidateQueries({ queryKey: DASHBOARD_KEY })
    },
  })

  const currentId = dashboard.data?.last_analysis?.id ?? null
  const finished = watch !== null && currentId !== watch.fromAnalysisId
  const waiting = watch !== null && Date.now() <= watch.until && !finished

  useEffect(() => {
    if (!finished) return
    void queryClient.invalidateQueries({ queryKey: ['meters'] })
    void queryClient.invalidateQueries({ queryKey: ['meter'] })
  }, [finished, queryClient])

  return { dashboard, start: () => run.mutate(), starting: run.isPending, waiting, error: run.isError ? run.error : null }
}
