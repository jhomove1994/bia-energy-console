import { useQuery } from '@tanstack/react-query'
import { getDashboardSummary } from '../api/dashboard'

export const DASHBOARD_KEY = ['dashboard'] as const

const POLL_MS = 1000

export interface DashboardWatch {
  /** Id of the last completed analysis when the run started; a different id means the run finished. */
  fromAnalysisId: string | null
  /** Give up polling at this epoch ms. */
  until: number
}

/** Dashboard KPIs. While `watch` is set, refetches every second until a newer analysis shows up. */
export function useDashboard(watch: DashboardWatch | null = null) {
  return useQuery({
    queryKey: DASHBOARD_KEY,
    queryFn: getDashboardSummary,
    staleTime: 10_000,
    refetchInterval: (query) => {
      if (!watch || Date.now() > watch.until) return false
      const current = query.state.data?.last_analysis?.id ?? null
      return current === watch.fromAnalysisId ? POLL_MS : false
    },
  })
}
