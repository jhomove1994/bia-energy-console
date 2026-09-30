import React from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { AppProviders } from './app/providers'
import { AppRoutes, routerFuture } from './app/router'
import './styles/index.css'

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AppProviders>
      <BrowserRouter future={routerFuture}>
        <AppRoutes />
      </BrowserRouter>
    </AppProviders>
  </React.StrictMode>,
)
