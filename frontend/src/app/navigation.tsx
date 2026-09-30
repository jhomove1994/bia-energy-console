import { AlertIcon, DashboardIcon, GaugeIcon } from '../components/atoms/Icon'
import type { NavItem } from '../components/organisms/AppShell'

/** Sidebar entries in the order of the demo flow. */
export const navItems: NavItem[] = [
  { to: '/', label: 'Dashboard', end: true, icon: <DashboardIcon /> },
  { to: '/meters', label: 'Medidores', icon: <GaugeIcon /> },
  { to: '/anomalies', label: 'Anomalías IA', icon: <AlertIcon /> },
]
