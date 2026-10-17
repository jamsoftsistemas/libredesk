<template>
  <!-- Idle: round mic button -->
  <button
    v-if="!active"
    type="button"
    :disabled="disabled || starting"
    :aria-label="$t('replyBox.recordAudio')"
    :title="$t('replyBox.recordAudio')"
    class="inline-flex items-center justify-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
    :class="buttonClass"
    @click="start"
  >
    <Mic class="h-4 w-4" />
  </button>

  <!-- Recording: WhatsApp-style bar -->
  <div v-else class="flex items-center gap-2 w-full min-w-0">
    <button
      type="button"
      :aria-label="$t('replyBox.cancelRecording')"
      :title="$t('replyBox.cancelRecording')"
      class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-muted-foreground hover:text-destructive hover:bg-muted transition-colors"
      @click="cancel"
    >
      <Trash2 class="h-4 w-4" />
    </button>

    <div class="flex items-center gap-2 shrink-0">
      <span class="h-2.5 w-2.5 rounded-full bg-red-500 animate-pulse" />
      <span class="text-sm tabular-nums text-foreground min-w-[2.75rem]">
        {{ formatRecordingDuration(recorder.state.elapsedMs) }}
      </span>
    </div>

    <div class="flex flex-1 min-w-0 h-8 items-center justify-end gap-[2px] overflow-hidden">
      <span
        v-for="(level, i) in bars"
        :key="i"
        class="w-[3px] shrink-0 rounded-full bg-muted-foreground/60"
        :style="{ height: `${Math.max(12, level * 100)}%` }"
      />
    </div>

    <button
      type="button"
      :disabled="stopping"
      :aria-label="$t('replyBox.sendAudio')"
      :title="$t('replyBox.sendAudio')"
      class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-[#25D366] text-white hover:bg-[#1ebe5b] transition-colors disabled:opacity-60"
      @click="send"
    >
      <SendHorizontal class="h-4 w-4" />
    </button>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Mic, Trash2, SendHorizontal } from 'lucide-vue-next'
import { useAudioRecorder, formatRecordingDuration } from '@shared-ui/composables/useAudioRecorder'

defineProps({
  disabled: { type: Boolean, default: false },
  // Classes for the idle mic button, so each app can match its own buttons.
  buttonClass: { type: [String, Array, Object], default: 'h-9 w-9 text-muted-foreground hover:bg-muted' }
})

// recorded(file): the user pressed send. error(code): 'insecure' | 'denied' | 'unsupported'.
// active(bool): the recording bar is shown, so the parent can hide its other controls.
const emit = defineEmits(['recorded', 'error', 'active'])

const recorder = useAudioRecorder()
const starting = ref(false)
const stopping = ref(false)

const active = computed(() => starting.value || recorder.state.isRecording)

watch(active, (v) => emit('active', v))

// Always render the same number of bars so the waveform scrolls in from the right.
const BAR_COUNT = 48
const bars = computed(() => {
  const levels = recorder.state.levels
  const pad = Math.max(0, BAR_COUNT - levels.length)
  return Array(pad).fill(0).concat(levels)
})

const start = async () => {
  starting.value = true
  try {
    await recorder.start()
  } catch (err) {
    emit('error', err.code || 'denied')
  } finally {
    starting.value = false
  }
}

const send = async () => {
  stopping.value = true
  try {
    const file = await recorder.stop()
    if (file) emit('recorded', file)
  } finally {
    stopping.value = false
  }
}

const cancel = () => recorder.cancel()
</script>
