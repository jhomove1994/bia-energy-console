import { useQuery } from '@tanstack/react-query'
import { getDashboardSummary } from '../api/dashboard'

export const DASHBOARD_KEY = ['dashboard'] as const

/** Dashboard KPIs; a finished analysis invalidates the query (see `useAnalysisJob`), so it refetches on its own. */
export function useDashboard() {
  return useQuery({ queryKey: DASHBOARD_KEY, queryFn: getDashboardSummary, staleTime: 10_000 })
}
