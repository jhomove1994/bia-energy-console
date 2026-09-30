import type { SelectHTMLAttributes } from 'react'

export interface SelectOption {
  value: string
  label: string
}

export interface SelectProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'id' | 'children'> {
  id: string
  label: string
  options: SelectOption[]
}

export function Select({ id, label, options, className = '', ...rest }: SelectProps) {
  return (
    <div className={className}>
      <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-text">
        {label}
      </label>
      <select
        {...rest}
        id={id}
        className="h-10 w-full cursor-pointer rounded-lg border border-border bg-surface px-3 text-sm text-text transition-colors duration-fast ease-console hover:border-muted/50 focus:border-primary-strong"
      >
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  )
}
