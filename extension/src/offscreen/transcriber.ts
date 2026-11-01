const REALTIME_URL = 'wss://api.elevenlabs.io/v1/speech-to-text/realtime'
const SAMPLE_RATE = 16000
const FRAME_SIZE = 4096
const COMMIT_PADDING_SAMPLES = 1600
const COMMIT_TIMEOUT_MS = 2500

export type Transcriber = { finish: () => Promise<string | null>; cancel: () => void }

type RealtimeMessage = { message_type?: string; text?: string; error?: string }

export function toBase64(buffer: ArrayBuffer): string {
  let binary = ''
  const bytes = new Uint8Array(buffer)
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
}

function toPcm16(samples: Float32Array): ArrayBuffer {
  const pcm = new Int16Array(samples.length)
  samples.forEach((sample, i) => {
    const clamped = Math.max(-1, Math.min(1, sample))
    pcm[i] = clamped < 0 ? clamped * 0x8000 : clamped * 0x7fff
  })
  return pcm.buffer
}

function socketUrl(token: string): string {
  const params = new URLSearchParams({
    model_id: 'scribe_v2_realtime',
    audio_format: 'pcm_16000',
    commit_strategy: 'manual',
    language_code: 'en',
    token,
  })
  return `${REALTIME_URL}?${params}`
}

export function startTranscriber(
  stream: MediaStream,
  token: string,
  onText: (text: string) => void,
): Transcriber {
  const context = new AudioContext({ sampleRate: SAMPLE_RATE })
  const source = context.createMediaStreamSource(stream)
  const processor = context.createScriptProcessor(FRAME_SIZE, 1, 1)
  const queued: string[] = []
  const committed: string[] = []
  let partial = ''
  let failed = false
  let onCommitted: (() => void) | null = null

  const socket = new WebSocket(socketUrl(token))
  const fail = () => {
    failed = true
    onCommitted?.()
  }
  const report = () => onText([...committed, partial].join(' ').trim())

  function send(samples: Float32Array, commit = false) {
    const message = JSON.stringify({
      message_type: 'input_audio_chunk',
      audio_base_64: toBase64(toPcm16(samples)),
      sample_rate: SAMPLE_RATE,
      commit,
    })
    if (socket.readyState === WebSocket.OPEN) socket.send(message)
    else queued.push(message)
  }

  socket.onopen = () => queued.splice(0).forEach((m) => socket.send(m))
  socket.onerror = fail
  socket.onclose = () => onCommitted?.()
  socket.onmessage = (event) => {
    let message: RealtimeMessage
    try {
      message = JSON.parse(String(event.data)) as RealtimeMessage
    } catch {
      return fail()
    }
    if (message.error || message.message_type?.endsWith('error')) return fail()
    if (message.message_type === 'partial_transcript') {
      partial = message.text ?? ''
      report()
    } else if (message.message_type === 'committed_transcript') {
      if (message.text) committed.push(message.text)
      partial = ''
      report()
      onCommitted?.()
    }
  }

  processor.onaudioprocess = (event) => {
    if (!failed) send(new Float32Array(event.inputBuffer.getChannelData(0)))
  }
  source.connect(processor)
  processor.connect(context.destination)

  function release() {
    processor.onaudioprocess = null
    processor.disconnect()
    source.disconnect()
    void context.close()
    socket.close()
  }

  async function finish(): Promise<string | null> {
    processor.onaudioprocess = null
    if (failed) {
      release()
      return null
    }
    send(new Float32Array(COMMIT_PADDING_SAMPLES), true)
    await new Promise<void>((resolve) => {
      onCommitted = resolve
      setTimeout(resolve, COMMIT_TIMEOUT_MS)
    })
    release()
    const text = [...committed, partial].join(' ').trim()
    return failed || !text ? null : text
  }

  return { finish, cancel: release }
}
