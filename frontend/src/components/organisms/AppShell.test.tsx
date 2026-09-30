import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AppShell } from './AppShell'

const nav = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/meters', label: 'Meters' },
]

function renderShell(path = '/', onLogout = vi.fn()) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <AppShell navItems={nav} username="admin" onLogout={onLogout}>
        <p>page content</p>
      </AppShell>
    </MemoryRouter>,
  )
  return onLogout
}

describe('AppShell', () => {
  it('renders the page content and the signed-in user', () => {
    renderShell()
    expect(screen.getByText('page content')).toBeInTheDocument()
    expect(screen.getByText('admin')).toBeInTheDocument()
  })

  it('lists the navigation and marks the current page', () => {
    renderShell('/meters')
    expect(screen.getByRole('link', { name: 'Meters' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Dashboard' })).not.toHaveAttribute('aria-current')
  })

  it('calls onLogout from the sign out button', async () => {
    const onLogout = renderShell()
    await userEvent.click(screen.getByRole('button', { name: 'Sign out' }))
    expect(onLogout).toHaveBeenCalledTimes(1)
  })

  it('offers a skip link to the main content', () => {
    renderShell()
    expect(screen.getByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#main')
    expect(screen.getByRole('main')).toHaveAttribute('id', 'main')
  })
})
