import type { Config } from 'tailwindcss'

const channel = (name: string) => `rgb(var(--${name}) / <alpha-value>)`

/** A tone maps to a base color (dots, bars, tints) and an ink color (text on a tint). */
const tone = (name: string) => ({ DEFAULT: channel(name), ink: channel(`${name}-ink`) })

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: channel('bg'),
        surface: channel('surface'),
        border: channel('border'),
        text: channel('text'),
        muted: channel('muted'),
        sidebar: {
          DEFAULT: channel('sidebar'),
          hover: channel('sidebar-hover'),
          text: channel('sidebar-text'),
        },
        primary: { DEFAULT: channel('primary'), strong: channel('primary-strong') },
        accent: channel('accent'),
        'sev-high': tone('sev-high'),
        'sev-medium': tone('sev-medium'),
        'sev-low': tone('sev-low'),
        'type-real': tone('type-real'),
        'type-dq': tone('type-dq'),
        'type-expl': tone('type-expl'),
        'type-fp': tone('type-fp'),
        'status-ok': tone('status-ok'),
        'status-alert': tone('status-alert'),
        'status-critical': tone('status-critical'),
      },
      borderRadius: { card: 'var(--radius)' },
      boxShadow: { card: 'var(--shadow)', raised: 'var(--shadow-raised)' },
      fontFamily: { sans: ["'Inter Variable'", 'ui-sans-serif', 'system-ui', 'sans-serif'] },
      transitionTimingFunction: { console: 'var(--ease)' },
      transitionDuration: { fast: 'var(--duration-fast)' },
    },
  },
} satisfies Config
