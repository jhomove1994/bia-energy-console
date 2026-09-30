import { Navigate, useRoutes, type RouteObject } from 'react-router-dom'
import { DashboardPage } from '../pages/Dashboard/DashboardPage'
import { LoginPage } from '../pages/Login/LoginPage'
import { MeterDetailPage } from '../pages/MeterDetail/MeterDetailPage'
import { MetersPage } from '../pages/Meters/MetersPage'
import { AppLayout } from './AppLayout'
import { AuthGuard } from './AuthGuard'

/** Opt in to the React Router v7 behaviors now, so upgrading is a no-op. */
export const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const

export const routes: RouteObject[] = [
  { path: '/login', element: <LoginPage /> },
  {
    element: <AuthGuard />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: '/', element: <DashboardPage /> },
          { path: '/meters', element: <MetersPage /> },
          { path: '/meters/:meterId', element: <MeterDetailPage /> },
        ],
      },
    ],
  },
  { path: '*', element: <Navigate to="/" replace /> },
]

/** Renders the route table; mount it inside any router (BrowserRouter in the app, MemoryRouter in tests). */
export function AppRoutes() {
  return useRoutes(routes)
}
