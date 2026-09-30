import { ApiError, apiFetch, setUnauthorizedHandler } from './client'
import { getToken, setToken } from './token'

function jsonResponse(status: number, body?: unknown): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: body === undefined ? {} : { 'Content-Type': 'application/json' },
  })
}

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  setUnauthorizedHandler(null)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('apiFetch', () => {
  it('sends the bearer token and parses the JSON body', async () => {
    setToken('tok-123')
    fetchMock.mockResolvedValue(jsonResponse(200, { meters_count: 12 }))

    const data = await apiFetch<{ meters_count: number }>('/api/dashboard/summary')

    expect(data).toEqual({ meters_count: 12 })
    const [url, init] = fetchMock.mock.calls[0]!
    expect(url).toBe('/api/dashboard/summary')
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer tok-123')
  })

  it('omits the Authorization header when there is no token', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {}))
    await apiFetch('/api/health')
    expect(new Headers(fetchMock.mock.calls[0]![1]?.headers).has('Authorization')).toBe(false)
  })

  it('serializes a JSON body with its content type', async () => {
    fetchMock.mockResolvedValue(jsonResponse(202, { id: 'abc' }))

    await apiFetch('/api/ai/analyze', { method: 'POST', body: { scope: 'all' } })

    const init = fetchMock.mock.calls[0]![1]!
    expect(init.method).toBe('POST')
    expect(init.body).toBe('{"scope":"all"}')
    expect(new Headers(init.headers).get('Content-Type')).toBe('application/json')
  })

  it('returns undefined for 204 No Content', async () => {
    fetchMock.mockResolvedValue(jsonResponse(204))
    await expect(apiFetch('/api/ai/analysis/latest')).resolves.toBeUndefined()
  })

  it('maps an error body to ApiError with status, code and message', async () => {
    fetchMock.mockResolvedValue(jsonResponse(404, { error: { code: 'not_found', message: 'meter not found' } }))

    const error = await apiFetch('/api/meters/X').catch((e: unknown) => e)

    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({ status: 404, code: 'not_found', message: 'No se encontró el recurso solicitado.', detail: 'meter not found' })
  })

  it('falls back to a generic ApiError when the error body is not JSON', async () => {
    fetchMock.mockResolvedValue(new Response('<html>bad gateway</html>', { status: 502 }))

    const error = await apiFetch('/api/meters').catch((e: unknown) => e)

    expect(error).toMatchObject({ status: 502, code: 'http_502', message: 'El servidor no pudo completar la solicitud (estado 502).' })
  })

  it('maps a network failure to ApiError with code "network"', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))

    const error = await apiFetch('/api/meters').catch((e: unknown) => e)

    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({ status: 0, code: 'network', message: 'No se pudo conectar con el servidor. Revise su conexión e intente de nuevo.' })
  })
})

describe('401 handling', () => {
  it('clears the token and calls the unauthorized handler', async () => {
    setToken('expired')
    const onUnauthorized = vi.fn()
    setUnauthorizedHandler(onUnauthorized)
    fetchMock.mockResolvedValue(jsonResponse(401, { error: { code: 'unauthorized', message: 'invalid token' } }))

    const error = await apiFetch('/api/meters').catch((e: unknown) => e)

    expect(error).toMatchObject({ status: 401, code: 'unauthorized' })
    expect(getToken()).toBeNull()
    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })

  it('leaves the session alone when the caller opts out (wrong credentials on login)', async () => {
    setToken('still-valid')
    const onUnauthorized = vi.fn()
    setUnauthorizedHandler(onUnauthorized)
    fetchMock.mockResolvedValue(jsonResponse(401, { error: { code: 'invalid_credentials', message: 'invalid credentials' } }))

    const error = await apiFetch('/api/auth/login', { method: 'POST', body: {}, handleUnauthorized: false }).catch((e: unknown) => e)

    expect(error).toMatchObject({ status: 401, code: 'invalid_credentials' })
    expect(getToken()).toBe('still-valid')
    expect(onUnauthorized).not.toHaveBeenCalled()
  })
})
