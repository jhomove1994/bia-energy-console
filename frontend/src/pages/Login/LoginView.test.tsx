import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LoginView } from './LoginView'

describe('LoginView', () => {
  it('submits the entered credentials', async () => {
    const onSubmit = vi.fn()
    render(<LoginView onSubmit={onSubmit} submitting={false} error={null} />)

    await userEvent.type(screen.getByLabelText('Usuario'), 'admin')
    await userEvent.type(screen.getByLabelText('Contraseña'), 'admin123')
    await userEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))

    expect(onSubmit).toHaveBeenCalledWith('admin', 'admin123')
  })

  it('flags empty fields inline and does not submit', async () => {
    const onSubmit = vi.fn()
    render(<LoginView onSubmit={onSubmit} submitting={false} error={null} />)

    await userEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))

    expect(screen.getByText('Ingrese su usuario')).toBeInTheDocument()
    expect(screen.getByText('Ingrese su contraseña')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('announces a server error', () => {
    render(<LoginView onSubmit={vi.fn()} submitting={false} error="Usuario o contraseña incorrectos." />)
    expect(screen.getByRole('alert')).toHaveTextContent('Usuario o contraseña incorrectos.')
  })

  it('disables the button while submitting', () => {
    render(<LoginView onSubmit={vi.fn()} submitting error={null} />)
    expect(screen.getByRole('button', { name: /Iniciar sesión/ })).toBeDisabled()
  })

  it('shows the demo credentials hint', () => {
    render(<LoginView onSubmit={vi.fn()} submitting={false} error={null} />)
    expect(screen.getByText(/admin \/ admin123/)).toBeInTheDocument()
  })
})
