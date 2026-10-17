import { ref, onBeforeUnmount } from 'vue'

// Preferred recording formats, best first. ogg/opus needs no server-side conversion.
const MIME_CANDIDATES = [
  { mime: 'audio/ogg;codecs=opus', ext: 'ogg' },
  { mime: 'audio/webm;codecs=opus', ext: 'webm' },
  { mime: 'audio/webm', ext: 'webm' },
  { mime: 'audio/mp4', ext: 'm4a' }
]

function pickFormat () {
  if (typeof MediaRecorder === 'undefined') return null
  return MIME_CANDIDATES.find((c) => MediaRecorder.isTypeSupported?.(c.mime)) || { mime: '', ext: 'webm' }
}

// isAudioRecordingSupported reports whether the browser can record audio.
export function isAudioRecordingSupported () {
  return typeof window !== 'undefined' &&
    typeof MediaRecorder !== 'undefined' &&
    !!navigator.mediaDevices?.getUserMedia
}

// useAudioRecorder records microphone audio and resolves a File on stop.
export function useAudioRecorder () {
  const isRecording = ref(false)
  const duration = ref(0)

  let recorder = null
  let stream = null
  let chunks = []
  let timer = null
  let startedAt = 0
  let format = null
  let stopResolver = null

  const cleanup = () => {
    clearInterval(timer)
    timer = null
    stream?.getTracks().forEach((t) => t.stop())
    stream = null
    recorder = null
    chunks = []
    isRecording.value = false
    duration.value = 0
  }

  // start asks for microphone access and begins recording; throws if denied or unsupported.
  const start = async () => {
    if (isRecording.value) return
    if (!isAudioRecordingSupported()) throw new Error('unsupported')
    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    format = pickFormat()
    recorder = format.mime ? new MediaRecorder(stream, { mimeType: format.mime }) : new MediaRecorder(stream)
    chunks = []
    recorder.ondataavailable = (e) => {
      if (e.data && e.data.size > 0) chunks.push(e.data)
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
    recorder.start(250)
    startedAt = Date.now()
    isRecording.value = true
    timer = setInterval(() => {
      duration.value = Math.floor((Date.now() - startedAt) / 1000)
    }, 250)
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

  return { isRecording, duration, start, stop, cancel }
}

// formatRecordingDuration renders seconds as m:ss.
export function formatRecordingDuration (seconds) {
  const m = Math.floor(seconds / 60)
  const s = String(seconds % 60).padStart(2, '0')
  return `${m}:${s}`
}
