import { ApiError } from '../api/client'

/** One independently loading part of a page, so a failure in it does not hide the rest. */
export type SectionState<T> = { status: 'loading' } | { status: 'error'; message: string } | { status: 'ready'; data: T }

export function describeError(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Error inesperado'
}

export function toSection<T>(q: { isPending: boolean; isError: boolean; error: unknown; data: T | undefined }): SectionState<T> {
  if (q.isPending) return { status: 'loading' }
  if (q.isError) return { status: 'error', message: describeError(q.error) }
  return { status: 'ready', data: q.data as T }
}
