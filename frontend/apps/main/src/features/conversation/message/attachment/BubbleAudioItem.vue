<template>
  <div class="flex flex-col gap-1 w-72 max-w-full group">
    <div class="flex items-center gap-1">
      <audio :src="attachment.url" controls preload="metadata" class="w-full h-10" />
      <DownloadLink
        :url="attachment.url"
        class="shrink-0 transition-opacity can-hover:opacity-0 can-hover:group-hover:opacity-100"
      />
    </div>
    <p
      v-if="transcription?.status === 'done' && transcription.text"
      class="text-xs text-muted-foreground whitespace-pre-wrap break-words px-1"
    >
      <span class="font-medium">{{ t('conversation.transcription') }}:</span>
      {{ transcription.text }}
    </p>
    <p
      v-else-if="transcription?.status === 'failed'"
      class="text-xs text-muted-foreground italic px-1"
    >
      {{ t('conversation.transcriptionFailed') }}
    </p>
    <p v-else-if="pending" class="text-xs text-muted-foreground italic px-1">
      {{ t('conversation.transcribing') }}
    </p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DownloadLink from '@/components/DownloadLink.vue'

// Transcriptions normally land within seconds; past this window a missing one means transcription is off.
const PENDING_WINDOW_MS = 3 * 60 * 1000

const props = defineProps({
  attachment: { type: Object, required: true },
  transcription: { type: Object, default: null },
  createdAt: { type: String, default: '' }
})

const { t } = useI18n()

const pending = computed(() => {
  if (props.transcription || !props.createdAt) return false
  return Date.now() - new Date(props.createdAt).getTime() < PENDING_WINDOW_MS
})
</script>
