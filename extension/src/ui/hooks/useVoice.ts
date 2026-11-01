import { useCallback, useEffect, useRef, useState } from 'react'

import { applyRules } from '@/engine/applyRules'
import { onEvent, request } from '@/shared/messages'
import { update } from '@/shared/storage'
import type { ApiErrorCode, ProposedRule } from '@/shared/types'

export type Stage = 'idle' | 'listening' | 'thinking' | 'review' | 'done'
export type VoiceError = { code: ApiErrorCode; message: string }

async function askForMic(inTab: boolean): Promise<boolean> {
  try {
    const state = (await navigator.permissions.query({ name: 'microphone' as PermissionName }))
      .state
    if (state === 'granted') return true
    if (state === 'denied') return false
  } catch {
    return true
  }
  if (!inTab) return false
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    stream.getTracks().forEach((t) => t.stop())
    return true
  } catch {
    return false
  }
}

export function useVoice({ inTab }: { inTab: boolean }) {
  const [stage, setStage] = useState<Stage>('idle')
  const [transcript, setTranscript] = useState('')
  const [rules, setRules] = useState<ProposedRule[]>([])
  const [off, setOff] = useState<Set<number>>(new Set())
  const [applied, setApplied] = useState(0)
  const [error, setError] = useState<VoiceError | null>(null)
  const listening = useRef(false)

  const parse = useCallback(async (text: string) => {
    setTranscript(text)
    setStage('thinking')
    const result = await request('voice:rules', { transcript: text })
    if (!result.ok) {
      setError({ code: result.code, message: result.message })
      setStage('idle')
      return
    }
    setRules(result.data.rules)
    setOff(new Set())
    setStage('review')
  }, [])

  useEffect(() => {
    const stopPartial = onEvent('voice:partial', ({ text }) => {
      if (listening.current) setTranscript(text)
    })
    const stopFinal = onEvent('voice:final', ({ transcript: text, error: problem }) => {
      if (!listening.current) return
      listening.current = false
      if (text) void parse(text)
      else {
        setError({ code: 'upstream', message: problem ?? "Didn't catch that." })
        setStage('idle')
      }
    })
    return () => {
      stopPartial()
      stopFinal()
      if (listening.current) void request('voice:cancel', {})
    }
  }, [parse])

  const start = useCallback(async () => {
    setError(null)
    setTranscript('')
    if (!(await askForMic(inTab))) {
      if (!inTab) {
        await chrome.tabs.create({ url: chrome.runtime.getURL('src/options/index.html?voice=1') })
        window.close()
        return
      }
      setError({
        code: 'mic_blocked',
        message: 'Microphone is blocked. Allow it in the site settings for Fuck Algo.',
      })
      return
    }
    listening.current = true
    setStage('listening')
    const result = await request('voice:start', {})
    if (!result.ok) {
      listening.current = false
      setError({ code: result.code, message: result.message })
      setStage('idle')
    }
  }, [inTab])

  const stop = useCallback(() => {
    void request('voice:stop', {})
  }, [])

  const submitText = useCallback(
    (text: string) => {
      const clean = text.trim()
      if (!clean) return
      setError(null)
      void parse(clean)
    },
    [parse],
  )

  const toggleRule = useCallback((i: number) => {
    setOff((current) => {
      const next = new Set(current)
      if (next.has(i)) next.delete(i)
      else next.add(i)
      return next
    })
  }, [])

  const apply = useCallback(async () => {
    const chosen = rules.filter((_, i) => !off.has(i))
    if (!chosen.length) return
    await update('settings', (s) => applyRules(s, chosen))
    setApplied(chosen.length)
    setStage('done')
  }, [rules, off])

  const reset = useCallback(() => {
    if (listening.current) void request('voice:cancel', {})
    listening.current = false
    setStage('idle')
    setTranscript('')
    setRules([])
    setError(null)
  }, [])

  return {
    stage,
    transcript,
    rules,
    off,
    applied,
    error,
    start,
    stop,
    submitText,
    toggleRule,
    apply,
    reset,
  }
}
