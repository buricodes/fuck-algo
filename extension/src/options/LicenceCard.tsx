import { useState } from 'react'

import { request } from '@/shared/messages'
import { Icon } from '@/ui/components'
import { useStore } from '@/ui/hooks/useStore'

export function LicenceCard() {
  const [licence] = useStore('licence')
  const [open, setOpen] = useState(false)
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (licence?.status === 'active') {
    return (
      <div className="set-licence">
        <Icon name="key" size={16} />
        <span>Lifetime licence</span>
        <span className="mono">··{licence.keyHint}</span>
      </div>
    )
  }

  const activate = async () => {
    if (!key.trim()) return
    setBusy(true)
    setError(null)
    const result = await request('licence:activate', { key: key.trim() })
    setBusy(false)
    if (result.ok) {
      setOpen(false)
      setKey('')
    } else setError(result.message || 'That key did not work.')
  }

  return (
    <div className="set-licence set-licence-trial">
      <div className="set-licence-row">
        <span>Free trial</span>
        <span className="mono">
          {licence
            ? `${licence.trial.voiceLeft} voice · ${licence.trial.checksLeft} checks`
            : 'offline'}
        </span>
      </div>
      {open ? (
        <div className="set-licence-form">
          <input
            value={key}
            onChange={(e) => setKey(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && void activate()}
            placeholder="FA-XXXX-XXXX-XXXX-XXXX"
            spellCheck={false}
            autoFocus
          />
          <button type="button" disabled={busy} onClick={() => void activate()}>
            {busy ? '…' : 'Activate'}
          </button>
          {error && <span className="set-licence-error">{error}</span>}
        </div>
      ) : (
        <div className="set-licence-row">
          <button
            type="button"
            className="set-unlock"
            onClick={() => void request('checkout:open', {})}
          >
            Unlock · $5 once
          </button>
          <button type="button" className="set-have-key" onClick={() => setOpen(true)}>
            I have a key
          </button>
        </div>
      )}
    </div>
  )
}
