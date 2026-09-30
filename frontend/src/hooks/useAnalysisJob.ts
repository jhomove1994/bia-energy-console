import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { getAnalysis, getLatestAnalysis, startAnalysis } from '../api/analysis'
import type { AnalysisStatus } from '../api/types'

const POLL_MS = 500

/** Query keys whose data a finished analysis makes stale. */
const REFRESHED_ON_DONE = ['dashboard', 'meters', 'meter', 'anomalies', 'anomaly'] as const

const isActive = (status: AnalysisStatus | undefined): boolean => status === 'pending' || status === 'running'

/** Poll every 500 ms while the run is pending or running; stop on a terminal status or without data. */
export function analysisRefetchInterval(status: AnalysisStatus | undefined): number | false {
  return isActive(status) ? POLL_MS : false
}

/*
 * One analysis run seen from the UI: POST starts it (or joins the one already running), the id is
 * polled until it is done or failed, and a finished run refreshes every view that shows its results.
 * A run already in flight when the page mounts is picked up from `/latest`, so navigating away and
 * back does not lose the progress.
 */
export function useAnalysisJob() {
  const queryClient = useQueryClient()
  const [id, setId] = useState<string | null>(null)

  const mutation = useMutation({ mutationFn: startAnalysis, onSuccess: ({ id: started }) => setId(started) })

  // No cache lifetime: a remount must ask the server, never trust a stale "running".
  const latest = useQuery({ queryKey: ['analysis', 'latest'], queryFn: getLatestAnalysis, gcTime: 0 })
  useEffect(() => {
    if (id === null && latest.data && isActive(latest.data.status)) setId(latest.data.id)
  }, [id, latest.data])

  const polled = useQuery({
    queryKey: ['analysis', id],
    queryFn: () => getAnalysis(id ?? ''),
    enabled: id !== null,
    refetchInterval: (query) => analysisRefetchInterval(query.state.data?.status),
  })

  const analysis = id === null ? undefined : polled.data
  const done = analysis?.status === 'done'

  useEffect(() => {
    if (!done) return
    for (const key of REFRESHED_ON_DONE) void queryClient.invalidateQueries({ queryKey: [key] })
  }, [done, id, queryClient])

  const waitingForFirstStatus = id !== null && polled.data === undefined && !polled.isError
  const active = mutation.isPending || isActive(analysis?.status) || waitingForFirstStatus

  return {
    /** Latest known state of the run; undefined until one was started or resumed. */
    analysis,
    /** A run is starting or in flight: the trigger must stay disabled. */
    active,
    starting: mutation.isPending,
    start: () => {
      if (!active) mutation.mutate()
    },
    /** Why the last start attempt failed, if it did. */
    error: mutation.isError ? mutation.error : null,
  }
}
