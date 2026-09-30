import { DashboardIcon } from '../components/atoms/Icon'
import type { NavItem } from '../components/organisms/AppShell'

/** Sidebar entries; later slices append Meters and Anomalies as their routes land. */
export const navItems: NavItem[] = [{ to: '/', label: 'Dashboard', end: true, icon: <DashboardIcon /> }]
