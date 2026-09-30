const TOKEN_KEY = 'energy.token'
const USERNAME_KEY = 'energy.username'

export function getToken(): string | null {
  return window.localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  window.localStorage.setItem(TOKEN_KEY, token)
}

export function getUsername(): string | null {
  return window.localStorage.getItem(USERNAME_KEY)
}

export function setUsername(username: string): void {
  window.localStorage.setItem(USERNAME_KEY, username)
}

/** Ends the stored session: token and the display name that goes with it. */
export function clearToken(): void {
  window.localStorage.removeItem(TOKEN_KEY)
  window.localStorage.removeItem(USERNAME_KEY)
}
