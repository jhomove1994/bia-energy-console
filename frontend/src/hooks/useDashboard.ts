import { useQuery } from '@tanstack/react-query'
import { getDashboardSummary } from '../api/dashboard'

export const DASHBOARD_KEY = ['dashboard'] as const

export function useDashboard() {
  return useQuery({ queryKey: DASHBOARD_KEY, queryFn: getDashboardSummary, staleTime: 10_000 })
}
