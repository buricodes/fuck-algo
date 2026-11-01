import { emit, handle, request } from '@/shared/messages'
import type { Result } from '@/shared/types'

import { startTranscriber, toBase64 } from './transcriber'
import type { Transcriber } from './transcriber'

const MAX_RECORDING_MS = 60_000
const SILENCE_AFTER_SPEECH_MS = 2_500
const SILENCE_RMS = 0.015

type Session = {
  stream: MediaStream
  recorder: MediaRecorder
  chunks: Blob[]
  transcriber: Transcriber | null
  cancelled: boolean
  timers: ReturnType<typeof setTimeout>[]
  meter: AudioContext
}

let session: Session | null = null

async function micState(): Promise<PermissionState> {
  try {
    return (await navigator.permissions.query({ name: 'microphone' as PermissionName })).state
  } catch {
    return 'prompt'
  }
}

function watchSilence(stream: MediaStream, onQuiet: () => void) {
  const meter = new AudioContext()
  const analyser = meter.createAnalyser()
  analyser.fftSize = 2048
  meter.createMediaStreamSource(stream).connect(analyser)
  const data = new Uint8Array(analyser.frequencyBinCount)
  let spoke = false
  let quietSince: number | null = null
  const timer = setInterval(() => {
    analyser.getByteTimeDomainData(data)
    let sum = 0
    for (const v of data) sum += ((v - 128) / 128) ** 2
    const loud = Math.sqrt(sum / data.length) > SILENCE_RMS
    if (loud) {
      spoke = true
      quietSince = null
      return
    }
    if (!spoke) return
    quietSince ??= Date.now()
    if (Date.now() - quietSince > SILENCE_AFTER_SPEECH_MS) onQuiet()
  }, 120)
  return { meter, timer }
}

async function start(token: string | null): Promise<Result<null>> {
  if (session) return { ok: true, data: null }
  if ((await micState()) === 'denied') {
    return { ok: false, code: 'mic_blocked', message: 'Microphone is blocked for Fuck Algo.' }
  }
  let stream: MediaStream
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  } catch {
    const state = await micState()
    return state === 'denied'
      ? { ok: false, code: 'mic_blocked', message: 'Microphone is blocked for Fuck Algo.' }
      : { ok: false, code: 'mic_needs_grant', message: 'Allow the microphone once to use voice.' }
  }

  const recorder = new MediaRecorder(
    stream,
    MediaRecorder.isTypeSupported('audio/webm') ? { mimeType: 'audio/webm' } : {},
  )
  const chunks: Blob[] = []
  recorder.ondataavailable = (e) => e.data.size && chunks.push(e.data)
  recorder.start(250)

  const silence = watchSilence(stream, () => void stop())
  session = {
    stream,
    recorder,
    chunks,
    transcriber: token
      ? startTranscriber(stream, token, (text) => emit('voice:partial', { text }))
      : null,
    cancelled: false,
    timers: [setTimeout(() => void stop(), MAX_RECORDING_MS), silence.timer],
    meter: silence.meter,
  }
  return { ok: true, data: null }
}

function release(s: Session) {
  s.timers.forEach((t) => clearTimeout(t))
  s.timers.forEach((t) => clearInterval(t))
  void s.meter.close()
  if (s.recorder.state !== 'inactive') s.recorder.stop()
  s.stream.getTracks().forEach((track) => track.stop())
}

async function recording(s: Session): Promise<Blob> {
  if (s.recorder.state === 'inactive') return new Blob(s.chunks, { type: 'audio/webm' })
  await new Promise<void>((resolve) => {
    s.recorder.onstop = () => resolve()
    s.recorder.stop()
  })
  return new Blob(s.chunks, { type: 'audio/webm' })
}

async function stop() {
  const s = session
  if (!s) return null
  session = null
  const transcript = s.transcriber ? await s.transcriber.finish() : null
  const audio = transcript ? null : await recording(s)
  release(s)
  if (s.cancelled) return null
  if (transcript) {
    emit('voice:final', { transcript, error: null })
    return null
  }
  if (!audio || audio.size < 2000) {
    emit('voice:final', { transcript: null, error: "Didn't catch that. Try again, or type it." })
    return null
  }
  const fallback = await request('offscreen:transcribe', {
    audio: toBase64(await audio.arrayBuffer()),
    contentType: 'audio/webm',
  })
  emit(
    'voice:final',
    fallback.ok && fallback.data.trim()
      ? { transcript: fallback.data.trim(), error: null }
      : {
          transcript: null,
          error: fallback.ok ? "Didn't catch that. Try again, or type it." : fallback.message,
        },
  )
  return null
}

function cancel() {
  const s = session
  if (!s) return null
  session = null
  s.cancelled = true
  s.transcriber?.cancel()
  release(s)
  return null
}

handle({
  'offscreen:start': ({ token }) => start(token),
  'offscreen:stop': () => stop(),
  'offscreen:cancel': () => cancel(),
})
