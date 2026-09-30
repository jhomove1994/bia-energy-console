import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LoginView } from './LoginView'

describe('LoginView', () => {
  it('submits the entered credentials', async () => {
    const onSubmit = vi.fn()
    render(<LoginView onSubmit={onSubmit} submitting={false} error={null} />)

    await userEvent.type(screen.getByLabelText('Username'), 'admin')
    await userEvent.type(screen.getByLabelText('Password'), 'admin123')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(onSubmit).toHaveBeenCalledWith('admin', 'admin123')
  })

  it('flags empty fields inline and does not submit', async () => {
    const onSubmit = vi.fn()
    render(<LoginView onSubmit={onSubmit} submitting={false} error={null} />)

    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(screen.getByText('Enter your username')).toBeInTheDocument()
    expect(screen.getByText('Enter your password')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('announces a server error', () => {
    render(<LoginView onSubmit={vi.fn()} submitting={false} error="Incorrect username or password." />)
    expect(screen.getByRole('alert')).toHaveTextContent('Incorrect username or password.')
  })

  it('disables the button while submitting', () => {
    render(<LoginView onSubmit={vi.fn()} submitting error={null} />)
    expect(screen.getByRole('button', { name: /Sign in/ })).toBeDisabled()
  })

  it('shows the demo credentials hint', () => {
    render(<LoginView onSubmit={vi.fn()} submitting={false} error={null} />)
    expect(screen.getByText(/admin \/ admin123/)).toBeInTheDocument()
  })
})
