import type { ButtonHTMLAttributes, CSSProperties, ReactNode } from 'react'

import { ICONS } from '@/shared/icons'
import type { IconName } from '@/shared/icons'

import './components.css'

export function Icon({
  name,
  size = 18,
  stroke = 1.75,
  style,
}: {
  name: IconName
  size?: number
  stroke?: number
  style?: CSSProperties
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={stroke}
      strokeLinecap="round"
      strokeLinejoin="round"
      style={{ flex: 'none', ...style }}
      aria-hidden
    >
      <path d={ICONS[name]} />
    </svg>
  )
}

export function Mark({ size }: { size: number }) {
  return (
    <img
      src={chrome.runtime.getURL('assets/fa-mark.png')}
      alt=""
      width={size}
      height={size}
      style={{ borderRadius: size * 0.27, display: 'block' }}
    />
  )
}

export function Toggle({
  on,
  onChange,
  size = 'sm',
  label,
}: {
  on: boolean
  onChange: (next: boolean) => void
  size?: 'sm' | 'lg'
  label: string
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      aria-label={label}
      className={`fa-toggle fa-toggle-${size}`}
      data-on={on}
      onClick={() => onChange(!on)}
    >
      <span />
    </button>
  )
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'light' | 'ghost' | 'outline' | 'icon'
  icon?: IconName
  children?: ReactNode
}

export function Button({ variant = 'primary', icon, children, className, ...rest }: ButtonProps) {
  return (
    <button type="button" className={`fa-btn fa-btn-${variant} ${className ?? ''}`} {...rest}>
      {icon && (
        <Icon
          name={icon}
          size={variant === 'icon' ? 18 : 16}
          stroke={variant === 'icon' ? 1.75 : 2.25}
        />
      )}
      {children}
    </button>
  )
}

export function Segmented<T extends string>({
  value,
  options,
  onChange,
  size = 'md',
}: {
  value: T
  options: readonly (readonly [T, string])[]
  onChange: (next: T) => void
  size?: 'sm' | 'md'
}) {
  return (
    <div className={`fa-seg fa-seg-${size}`} role="radiogroup">
      {options.map(([key, label]) => (
        <button
          key={key}
          type="button"
          role="radio"
          aria-checked={value === key}
          data-on={value === key}
          onClick={() => onChange(key)}
        >
          {label}
        </button>
      ))}
    </div>
  )
}

export function StatTile({
  n,
  label,
  accent,
  dim,
}: {
  n: number | string
  label: string
  accent?: boolean
  dim?: boolean
}) {
  return (
    <div className="fa-stat" style={{ opacity: dim ? 0.35 : 1 }}>
      <span className="display" style={{ color: accent ? 'var(--acct)' : undefined }}>
        {n}
      </span>
      <span>{label}</span>
    </div>
  )
}

export function Pill({ live, children }: { live: boolean; children: ReactNode }) {
  return (
    <span className="fa-pill" data-live={live}>
      <span />
      {children}
    </span>
  )
}

export function StepSlider({
  value,
  labels,
  onChange,
}: {
  value: number
  labels: readonly string[]
  onChange: (next: number) => void
}) {
  const last = labels.length - 1
  return (
    <div className="fa-steps">
      <div className="fa-steps-track">
        <div className="fa-steps-rail" />
        <div className="fa-steps-fill" style={{ width: `${(value / last) * 100}%` }} />
        {labels.map((label, i) => (
          <button
            key={label}
            type="button"
            title={label}
            aria-label={label}
            className="fa-steps-stop"
            data-state={i === value ? 'active' : i < value ? 'filled' : 'empty'}
            style={{ left: `${(i / last) * 100}%` }}
            onClick={() => onChange(i)}
          />
        ))}
      </div>
      <div className="fa-steps-labels">
        {labels.map((label, i) => (
          <button key={label} type="button" data-on={i === value} onClick={() => onChange(i)}>
            {label}
          </button>
        ))}
      </div>
    </div>
  )
}

export function Range({
  value,
  min,
  max,
  onChange,
  label,
}: {
  value: number
  min: number
  max: number
  onChange: (next: number) => void
  label: string
}) {
  const pct = `${((value - min) / (max - min)) * 100}%`
  return (
    <div className="fa-range">
      <div className="fa-range-rail" />
      <div className="fa-range-fill" style={{ width: pct }} />
      <div className="fa-range-knob" style={{ left: pct }} />
      <input
        type="range"
        min={min}
        max={max}
        step={1}
        value={value}
        aria-label={label}
        onChange={(e) => onChange(Number(e.target.value))}
      />
    </div>
  )
}

export function PulseDot() {
  return <span className="fa-pulse-dot" />
}
