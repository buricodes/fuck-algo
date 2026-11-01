import { request } from '@/shared/messages'
import { Icon } from '@/ui/components'
import { useStore } from '@/ui/hooks/useStore'

export function AccountCard() {
  const [account] = useStore('account')

  if (!account) {
    return (
      <div className="set-licence set-licence-trial">
        <div className="set-licence-row">
          <span>Not signed in</span>
          <span className="mono">local rules only</span>
        </div>
        <button
          type="button"
          className="set-unlock"
          onClick={() => void request('account:signin', {})}
        >
          Sign in for voice and AI
        </button>
      </div>
    )
  }

  if (account.licence.status === 'active') {
    return (
      <div className="set-licence set-licence-trial">
        <div className="set-licence-row">
          <span className="set-licence-who">
            <Icon name="key" size={16} />
            Lifetime licence
          </span>
          <span className="mono">{account.email}</span>
        </div>
      </div>
    )
  }

  return (
    <div className="set-licence set-licence-trial">
      <div className="set-licence-row">
        <span>Free trial</span>
        <span className="mono">
          {account.licence.trial.voiceLeft} voice · {account.licence.trial.checksLeft} checks
        </span>
        <span className="mono">{account.email}</span>
      </div>
      <button
        type="button"
        className="set-unlock"
        onClick={() => void request('checkout:open', {})}
      >
        Unlock · $5 once
      </button>
    </div>
  )
}
