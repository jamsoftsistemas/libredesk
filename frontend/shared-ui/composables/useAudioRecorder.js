import { reactive, onBeforeUnmount } from 'vue'

// Preferred recording formats, best first. ogg/opus needs no server-side conversion.
const MIME_CANDIDATES = [
  { mime: 'audio/ogg;codecs=opus', ext: 'ogg' },
  { mime: 'audio/webm;codecs=opus', ext: 'webm' },
  { mime: 'audio/webm', ext: 'webm' },
  { mime: 'audio/mp4', ext: 'm4a' }
]

// Number of waveform bars kept while recording.
const LEVEL_COUNT = 48
const TICK_MS = 100

function pickFormat () {
  if (typeof MediaRecorder === 'undefined') return null
  return MIME_CANDIDATES.find((c) => MediaRecorder.isTypeSupported?.(c.mime)) || { mime: '', ext: 'webm' }
}

// isAudioRecordingSupported reports whether the browser has a recorder at all. The microphone itself may
// still be unavailable (e.g. plain HTTP on a non-localhost host); start() then throws an Error with code 'insecure'.
export function isAudioRecordingSupported () {
  return typeof window !== 'undefined' && typeof MediaRecorder !== 'undefined'
}

function recorderError (code) {
  const err = new Error(code)
  err.code = code
  return err
}

// useAudioRecorder records microphone audio and resolves a File on stop.
// state.elapsedMs is the recorded time; state.levels holds recent input levels (0..1) for a waveform.
export function useAudioRecorder () {
  const state = reactive({
    isRecording: false,
    elapsedMs: 0,
    levels: []
  })

  let recorder = null
  let stream = null
  let chunks = []
  let timer = null
  let startedAt = 0
  let format = null
  let stopResolver = null
  let audioCtx = null
  let analyser = null
  let sampleBuf = null

  const cleanup = () => {
    clearInterval(timer)
    timer = null
    stream?.getTracks().forEach((t) => t.stop())
    stream = null
    audioCtx?.close().catch(() => {})
    audioCtx = null
    analyser = null
    recorder = null
    chunks = []
    state.isRecording = false
    state.elapsedMs = 0
    state.levels = []
  }

  // currentLevel returns the RMS input level scaled to 0..1.
  const currentLevel = () => {
    if (!analyser) return 0
    analyser.getByteTimeDomainData(sampleBuf)
    let sum = 0
    for (let i = 0; i < sampleBuf.length; i++) {
      const v = (sampleBuf[i] - 128) / 128
      sum += v * v
    }
    return Math.min(1, Math.sqrt(sum / sampleBuf.length) * 4)
  }

  const tick = () => {
    state.elapsedMs = performance.now() - startedAt
    const levels = state.levels.concat(currentLevel())
    state.levels = levels.length > LEVEL_COUNT ? levels.slice(-LEVEL_COUNT) : levels
  }

  const setupAnalyser = () => {
    try {
      const Ctx = window.AudioContext || window.webkitAudioContext
      if (!Ctx) return
      audioCtx = new Ctx()
      analyser = audioCtx.createAnalyser()
      analyser.fftSize = 512
      sampleBuf = new Uint8Array(analyser.fftSize)
      audioCtx.createMediaStreamSource(stream).connect(analyser)
    } catch {
      analyser = null
    }
  }

  // start asks for microphone access and begins recording; throws if denied or unsupported.
  const start = async () => {
    if (state.isRecording || recorder) return
    if (!isAudioRecordingSupported()) throw recorderError('unsupported')
    // Browsers only expose the microphone on HTTPS or localhost.
    if (!window.isSecureContext || !navigator.mediaDevices?.getUserMedia) throw recorderError('insecure')
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    } catch {
      throw recorderError('denied')
    }
    format = pickFormat()
    recorder = format.mime ? new MediaRecorder(stream, { mimeType: format.mime }) : new MediaRecorder(stream)
    chunks = []
    recorder.ondataavailable = (e) => {
      if (e.data && e.data.size > 0) chunks.push(e.data)
    }
    // Count time from when the recorder actually starts capturing, not from the click.
    recorder.onstart = () => {
      startedAt = performance.now()
      state.elapsedMs = 0
      state.isRecording = true
      clearInterval(timer)
      timer = setInterval(tick, TICK_MS)
    }
    recorder.onstop = () => {
      const type = (recorder?.mimeType || format.mime || 'audio/webm').split(';')[0]
      const blob = new Blob(chunks, { type })
      const file = blob.size > 0
        ? new File([blob], `audio-${Date.now()}.${format.ext}`, { type })
        : null
      const resolve = stopResolver
      stopResolver = null
      cleanup()
      resolve?.(file)
    }
    setupAnalyser()
    recorder.start(250)
  }

  // stop finishes recording and resolves the recorded File (null when empty).
  const stop = () => new Promise((resolve) => {
    if (!recorder || recorder.state === 'inactive') {
      cleanup()
      resolve(null)
      return
    }
    stopResolver = resolve
    recorder.stop()
  })

  // cancel discards the recording.
  const cancel = () => {
    if (recorder && recorder.state !== 'inactive') {
      stopResolver = null
      recorder.onstop = null
      recorder.stop()
    }
    cleanup()
  }

  onBeforeUnmount(cancel)

  return { state, start, stop, cancel }
}

// formatRecordingDuration renders milliseconds as m:ss.
export function formatRecordingDuration (ms) {
  const total = Math.floor((ms || 0) / 1000)
  const m = Math.floor(total / 60)
  const s = String(total % 60).padStart(2, '0')
  return `${m}:${s}`
}
