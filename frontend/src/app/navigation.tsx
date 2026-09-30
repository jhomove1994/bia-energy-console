import { DashboardIcon, GaugeIcon } from '../components/atoms/Icon'
import type { NavItem } from '../components/organisms/AppShell'

/** Sidebar entries; later slices append Anomalies as their routes land. */
export const navItems: NavItem[] = [
  { to: '/', label: 'Dashboard', end: true, icon: <DashboardIcon /> },
  { to: '/meters', label: 'Meters', icon: <GaugeIcon /> },
]
