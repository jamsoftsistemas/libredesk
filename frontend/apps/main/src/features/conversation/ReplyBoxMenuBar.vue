<template>
  <div
    class="relative flex min-h-14 flex-wrap justify-between gap-y-2"
    :class="{ 'items-end': isFullscreen, 'items-center': !isFullscreen }"
  >
    <EmojiPicker
      ref="emojiPickerRef"
      :native="true"
      @select="onSelectEmoji"
      class="absolute bottom-14 left-0 md:left-14 z-20"
      v-if="isEmojiPickerVisible"
    />
    <div class="flex items-center gap-1">
      <!-- File inputs -->
      <input
        type="file"
        class="hidden"
        ref="attachmentInput"
        multiple
        :accept="attachmentAccept"
        @change="handleFileUpload"
      />
      <!-- <input
        type="file"
        class="hidden"
        ref="inlineImageInput"
        accept="image/*"
        @change="handleInlineImageUpload"
      /> -->
      <!-- Editor buttons -->
      <Tooltip>
        <TooltipTrigger as-child>
          <Toggle
            :class="ICON_BUTTON_CLASS"
            variant="outline"
            :aria-label="$t('globals.messages.attachFile')"
            @click="triggerFileUpload"
            :pressed="false"
          >
            <Paperclip class="h-4 w-4" />
          </Toggle>
        </TooltipTrigger>
        <TooltipContent>{{ $t('globals.messages.attachFile') }}</TooltipContent>
      </Tooltip>
      <template v-if="canRecordAudio">
        <div
          v-if="recorder.isRecording.value"
          class="flex items-center gap-1 rounded-md border border-border/70 px-2 h-8 max-md:h-11"
        >
          <span class="h-2 w-2 rounded-full bg-red-500 animate-pulse" />
          <span class="text-xs tabular-nums text-muted-foreground w-9">
            {{ formatRecordingDuration(recorder.duration.value) }}
          </span>
          <Button
            variant="ghost"
            size="icon"
            class="h-6 w-6"
            :aria-label="$t('replyBox.cancelRecording')"
            @click="cancelRecording"
          >
            <X class="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            class="h-6 w-6 text-primary"
            :aria-label="$t('replyBox.stopRecording')"
            @click="stopRecording"
          >
            <Check class="h-4 w-4" />
          </Button>
        </div>
        <Tooltip v-else>
          <TooltipTrigger as-child>
            <Toggle
              :class="ICON_BUTTON_CLASS"
              variant="outline"
              :aria-label="$t('replyBox.recordAudio')"
              @click="startRecording"
              :pressed="false"
            >
              <Mic class="h-4 w-4" />
            </Toggle>
          </TooltipTrigger>
          <TooltipContent>{{ $t('replyBox.recordAudio') }}</TooltipContent>
        </Tooltip>
      </template>
      <Tooltip>
        <TooltipTrigger as-child>
          <Toggle
            :class="ICON_BUTTON_CLASS"
            variant="outline"
            :aria-label="$t('globals.messages.addEmoji')"
            @click="toggleEmojiPicker"
            :pressed="isEmojiPickerVisible"
          >
            <Smile class="h-4 w-4" />
          </Toggle>
        </TooltipTrigger>
        <TooltipContent>{{ $t('globals.messages.addEmoji') }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Toggle
            :class="[ICON_BUTTON_CLASS, 'md:hidden']"
            variant="outline"
            :aria-label="$t('actions.applyMacro')"
            @click="openMacroPicker"
            :pressed="false"
          >
            <Zap class="h-4 w-4" />
          </Toggle>
        </TooltipTrigger>
        <TooltipContent>{{ $t('actions.applyMacro') }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="showWhatsAppTemplate">
        <TooltipTrigger as-child>
          <Toggle
            :class="ICON_BUTTON_CLASS"
            variant="outline"
            :aria-label="$t('globals.messages.sendTemplate')"
            @click="openTemplatePicker"
            :pressed="false"
          >
            <WhatsAppIcon class="h-4 w-4" />
          </Toggle>
        </TooltipTrigger>
        <TooltipContent>{{ $t('globals.messages.sendTemplate') }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="showGenerateReply">
        <TooltipTrigger as-child>
          <Toggle
            :class="ICON_BUTTON_CLASS"
            variant="outline"
            :pressed="false"
            :disabled="isGenerating"
            :aria-label="$t('replyBox.generateReply')"
            @click="emit('generateReply')"
          >
            <Loader2 v-if="isGenerating" class="h-4 w-4 animate-spin" />
            <Sparkles v-else class="h-4 w-4" />
          </Toggle>
        </TooltipTrigger>
        <TooltipContent>{{ $t('replyBox.generateReply') }}</TooltipContent>
      </Tooltip>
    </div>
    <div class="ml-auto flex items-center rounded-md shadow-sm">
      <Button
        class="h-8 max-md:h-11 px-4 rounded-r-none"
        @click="handleSend"
        :disabled="!enableSend"
        :isLoading="isSending"
        v-if="showSendButton"
      >
        {{ $t('globals.messages.send') }}
      </Button>
      <DropdownMenu v-if="showSendButton">
        <DropdownMenuTrigger as-child>
          <Button
            class="h-8 max-md:h-11 px-2 rounded-l-none border-l border-primary-foreground/30 [&[data-state=open]>svg]:rotate-180"
            :disabled="!enableSend"
          >
            <ChevronDownIcon class="text-primary-foreground transition-transform" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuLabel>{{ $t('replyBox.sendAndSetAs') }}</DropdownMenuLabel>
          <DropdownMenuItem
            v-for="status in conversationStore.statusOptionsNoSnooze"
            :key="status.value"
            @click="handleSendAndSetStatus(status.label)"
          >
            {{ conversationStore.translateStatusLabel(status.label) }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </div>
</template>

<script setup>
const ICON_BUTTON_CLASS =
  'border-border/70 bg-background px-2 py-2 text-muted-foreground shadow-none max-md:min-h-11 max-md:min-w-11'

import { ref, computed, defineAsyncComponent } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button'
import { Toggle } from '@shared-ui/components/ui/toggle'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Paperclip, Smile, Zap, ChevronDownIcon, Sparkles, Loader2, Mic, Check, X } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuItem,
  DropdownMenuContent,
  DropdownMenuLabel
} from '@shared-ui/components/ui/dropdown-menu'
import WhatsAppIcon from '@main/components/icons/WhatsAppIcon.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { WHATSAPP_MEDIA_ACCEPT } from '@main/features/conversation/whatsappMedia'
import { MACROS_COMMAND, useCommandPalette } from '@main/features/command/useCommandPalette'
import {
  useAudioRecorder,
  isAudioRecordingSupported,
  formatRecordingDuration
} from '@shared-ui/composables/useAudioRecorder'
import { useI18n } from 'vue-i18n'

const EmojiPicker = defineAsyncComponent(async () => {
  const [mod] = await Promise.all([import('vue3-emoji-picker'), import('vue3-emoji-picker/css')])
  return mod.default
})

const attachmentInput = ref(null)
// const inlineImageInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerRef = ref(null)
const emit = defineEmits(['emojiSelect', 'generateReply', 'audioRecorded'])

// Using defineProps for props that don't need two-way binding
const props = defineProps({
  isFullscreen: Boolean,
  isWhatsApp: Boolean,
  showWhatsAppTemplate: Boolean,
  isSending: Boolean,
  isGenerating: Boolean,
  enableSend: Boolean,
  handleSend: Function,
  handleSendAndSetStatus: Function,
  showSendButton: {
    type: Boolean,
    default: true
  },
  showGenerateReply: {
    type: Boolean,
    default: true
  },
  handleFileUpload: Function,
  handleInlineImageUpload: Function
})

onClickOutside(emojiPickerRef, () => {
  isEmojiPickerVisible.value = false
})

const triggerFileUpload = () => {
  if (attachmentInput.value) {
    // Clear the value to allow the same file to be uploaded again.
    attachmentInput.value.value = ''
    attachmentInput.value.click()
  }
}

const toggleEmojiPicker = () => {
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

const conversationStore = useConversationStore()
const emitter = useEmitter()
const palette = useCommandPalette()

const attachmentAccept = computed(() => (props.isWhatsApp ? WHATSAPP_MEDIA_ACCEPT : undefined))

const openMacroPicker = () => {
  palette.openPalette({ parent: MACROS_COMMAND })
}

const openTemplatePicker = () => {
  emitter.emit(EMITTER_EVENTS.WHATSAPP_TEMPLATE_PICKER_OPEN)
}

const { t } = useI18n()
const recorder = useAudioRecorder()
const canRecordAudio = isAudioRecordingSupported()

const startRecording = async () => {
  try {
    await recorder.start()
  } catch {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('replyBox.micPermissionDenied')
    })
  }
}

const stopRecording = async () => {
  const file = await recorder.stop()
  if (file) emit('audioRecorded', file)
}

const cancelRecording = () => recorder.cancel()

function onSelectEmoji(emoji) {
  emit('emojiSelect', emoji.i)
}
</script>
