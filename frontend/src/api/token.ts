const KEY = 'energy.token'

export function getToken(): string | null {
  return window.localStorage.getItem(KEY)
}

export function setToken(token: string): void {
  window.localStorage.setItem(KEY, token)
}

export function clearToken(): void {
  window.localStorage.removeItem(KEY)
}
