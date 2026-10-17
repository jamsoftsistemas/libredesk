<template>
  <div class="flex items-center gap-1">
    <!-- File Upload Input -->
    <input
      type="file"
      class="hidden"
      ref="fileInput"
      @change="handleFileUpload"
      :accept="fileUploadEnabled ? '*/*' : ''"
      :disabled="!fileUploadEnabled"
    />

    <!-- File Upload Button -->
    <Button
      v-if="fileUploadEnabled && canUploadFiles"
      @click="triggerFileUpload"
      :disabled="uploading || disabled"
      :aria-label="$t('globals.messages.attachFile')"
      variant="ghost"
      size="sm"
      class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
    >
      <Paperclip class="h-4 w-4 text-muted-foreground" />
    </Button>

    <!-- Audio Recording -->
    <template v-if="fileUploadEnabled && canUploadFiles && canRecordAudio">
      <div v-if="recorder.isRecording.value" class="flex items-center gap-1 px-1">
        <span class="h-2 w-2 rounded-full bg-red-500 animate-pulse" />
        <span class="text-xs tabular-nums text-muted-foreground w-9">
          {{ formatRecordingDuration(recorder.duration.value) }}
        </span>
        <Button
          @click="cancelRecording"
          :aria-label="$t('replyBox.cancelRecording')"
          variant="ghost"
          size="sm"
          class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
        >
          <X class="h-4 w-4 text-muted-foreground" />
        </Button>
        <Button
          @click="stopRecording"
          :aria-label="$t('replyBox.stopRecording')"
          variant="ghost"
          size="sm"
          class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
        >
          <Check class="h-4 w-4 text-primary" />
        </Button>
      </div>
      <Button
        v-else
        @click="startRecording"
        :disabled="uploading || disabled"
        :aria-label="$t('replyBox.recordAudio')"
        :title="$t('replyBox.recordAudio')"
        variant="ghost"
        size="sm"
        class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
      >
        <Mic class="h-4 w-4 text-muted-foreground" />
      </Button>
    </template>

    <!-- Emoji Picker -->
    <div class="relative">
      <Button
        v-if="emojiEnabled"
        @click="toggleEmojiPicker"
        :disabled="disabled"
        :class="{ 'bg-muted': isEmojiPickerVisible }"
        :title="$t('globals.messages.addEmoji')"
        :aria-label="$t('globals.messages.addEmoji')"
        variant="ghost"
        size="sm"
        class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
      >
        <Smile class="h-4 w-4 text-muted-foreground" />
      </Button>

      <EmojiPicker
        v-if="isEmojiPickerVisible && emojiEnabled"
        ref="emojiPickerRef"
        :native="true"
        @select="onSelectEmoji"
        class="absolute bottom-12 left-0 z-50"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, defineAsyncComponent } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button'
import { Smile, Paperclip, Mic, Check, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import {
  useAudioRecorder,
  isAudioRecordingSupported,
  formatRecordingDuration
} from '@shared-ui/composables/useAudioRecorder'

const EmojiPicker = defineAsyncComponent(async () => {
  const [mod] = await Promise.all([
    import('vue3-emoji-picker'),
    import('vue3-emoji-picker/css'),
  ])
  return mod.default
})

const props = defineProps({
  fileUploadEnabled: {
    type: Boolean,
    default: false
  },
  emojiEnabled: {
    type: Boolean,
    default: false
  },
  uploading: {
    type: Boolean,
    default: false
  },
  canUploadFiles: {
    type: Boolean,
    default: true
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['fileUpload', 'emojiSelect', 'audioRecorded', 'error'])

const { t } = useI18n()
const recorder = useAudioRecorder()
const canRecordAudio = isAudioRecordingSupported()

const startRecording = async () => {
  try {
    await recorder.start()
  } catch {
    emit('error', t('replyBox.micPermissionDenied'))
  }
}

const stopRecording = async () => {
  const file = await recorder.stop()
  if (file) emit('audioRecorded', file)
}

const cancelRecording = () => recorder.cancel()

const fileInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerRef = ref(null)

// Close emoji picker when clicking outside
onClickOutside(emojiPickerRef, () => {
  isEmojiPickerVisible.value = false
})

const triggerFileUpload = () => {
  if (fileInput.value && props.fileUploadEnabled && !props.uploading) {
    fileInput.value.value = ''
    fileInput.value.click()
  }
}

const handleFileUpload = (event) => {
  const files = event.target.files
  if (files && files.length > 0) {
    emit('fileUpload', files)
  }
}

const toggleEmojiPicker = () => {
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

const onSelectEmoji = (emoji) => {
  emit('emojiSelect', emoji.i)
  isEmojiPickerVisible.value = false
}
</script>
