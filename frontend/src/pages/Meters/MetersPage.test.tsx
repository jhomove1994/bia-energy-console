import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { setToken } from '../../api/token'
import type { MeterRow } from '../../api/types'
import { AppProviders } from '../../app/providers'
import { routerFuture } from '../../app/router'
import { createTestQueryClient, jsonResponse } from '../../test/utils'
import { MetersPage } from './MetersPage'

const fetchMock = vi.fn<typeof fetch>()

const all: MeterRow[] = [
  { meter_id: 'M-101', status: 'OK', last_reading_at: null, total_kwh: 100, variation_pct: 1 },
  { meter_id: 'M-109', status: 'Critical', last_reading_at: null, total_kwh: 200, variation_pct: 110 },
  { meter_id: 'M-104', status: 'Alert', last_reading_at: null, total_kwh: 150, variation_pct: 47 },
]

/** A tiny server: applies `q` and `status` like the API does. */
function serve() {
  fetchMock.mockImplementation(async (input) => {
    const url = new URL(String(input), 'http://x')
    const q = (url.searchParams.get('q') ?? '').toLowerCase()
    const status = url.searchParams.get('status')
    const items = all.filter((m) => m.meter_id.toLowerCase().includes(q) && (!status || m.status === status))
    return jsonResponse(200, { items })
  })
}

const requestedUrls = () => fetchMock.mock.calls.map((c) => new URL(String(c[0]), 'http://x'))

function Location() {
  const l = useLocation()
  return <output aria-label="location">{l.pathname + l.search}</output>
}

function renderAt(path: string) {
  render(
    <AppProviders queryClient={createTestQueryClient()}>
      <MemoryRouter initialEntries={[path]} future={routerFuture}>
        <MetersPage />
        <Location />
      </MemoryRouter>
    </AppProviders>,
  )
}

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  setToken('tok')
  serve()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('MetersPage', () => {
  it('lists every meter at first', async () => {
    renderAt('/meters')
    expect(await screen.findByRole('link', { name: 'M-101' })).toBeInTheDocument()
    expect(screen.getAllByRole('row')).toHaveLength(4)
  })

  it('typing 109 leaves only M-109 and puts the search in the URL', async () => {
    renderAt('/meters')
    await screen.findByRole('link', { name: 'M-101' })
    await userEvent.type(screen.getByRole('searchbox', { name: 'Buscar medidores' }), '109')

    await waitFor(() => expect(screen.getAllByRole('row')).toHaveLength(2))
    expect(screen.getByRole('link', { name: 'M-109' })).toBeInTheDocument()
    expect(screen.getByLabelText('location')).toHaveTextContent('/meters?q=109')
    expect(requestedUrls().at(-1)!.searchParams.get('q')).toBe('109')
  })

  it('reads the filters from a shared link', async () => {
    renderAt('/meters?status=Critical&sort=variation&order=desc')
    expect(await screen.findByRole('link', { name: 'M-109' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'M-101' })).not.toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Estado' })).toHaveValue('Critical')
    expect(screen.getByRole('combobox', { name: 'Ordenar por' })).toHaveValue('variation:desc')
    const params = requestedUrls()[0]!.searchParams
    expect(params.get('status')).toBe('Critical')
    expect(params.get('sort')).toBe('variation')
    expect(params.get('order')).toBe('desc')
  })

  it('filters by status and sorts through the URL', async () => {
    renderAt('/meters')
    await screen.findByRole('link', { name: 'M-101' })
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Estado' }), 'Alert')
    await waitFor(() => expect(screen.getAllByRole('row')).toHaveLength(2))
    expect(screen.getByLabelText('location')).toHaveTextContent('status=Alert')

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Ordenar por' }), 'kwh:desc')
    await waitFor(() => expect(requestedUrls().at(-1)!.searchParams.get('sort')).toBe('kwh'))
    expect(requestedUrls().at(-1)!.searchParams.get('order')).toBe('desc')
  })

  it('clears every filter from the empty state', async () => {
    renderAt('/meters?q=zzz')
    expect(await screen.findByText('Ningún medidor coincide con los filtros')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Borrar filtros' }))
    expect(await screen.findByRole('link', { name: 'M-101' })).toBeInTheDocument()
    expect(screen.getByLabelText('location')).toHaveTextContent(/^\/meters$/)
  })

  it('shows the error and recovers on retry', async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(500, { error: { code: 'internal', message: 'boom' } }))
    renderAt('/meters')
    expect(await screen.findByRole('alert')).toHaveTextContent('Ocurrió un error interno en el servidor')
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(await screen.findByRole('link', { name: 'M-101' })).toBeInTheDocument()
  })
})
