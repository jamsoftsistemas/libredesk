<template>
  <div v-if="audioAttachments.length" class="flex flex-col gap-2">
    <BubbleAudioItem
      v-for="attachment in audioAttachments"
      :key="attachment.uuid"
      :attachment="attachment"
      :transcription="transcriptions?.[attachment.uuid] || null"
      :created-at="createdAt"
    />
  </div>
  <div v-if="otherAttachments.length" class="flex flex-row flex-wrap gap-2 break-all">
    <BubbleAttachmentItem
      v-for="attachment in otherAttachments"
      :key="attachment.uuid"
      :attachment="attachment"
      @preview="openLightbox"
    />
  </div>

  <ImageLightbox
    v-model="lightboxOpen"
    :images="imageAttachments"
    :start-index="lightboxIndex"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
import BubbleAttachmentItem from '@/features/conversation/message/attachment/BubbleAttachmentItem.vue'
import BubbleAudioItem from '@/features/conversation/message/attachment/BubbleAudioItem.vue'
import ImageLightbox from '@/components/ImageLightbox.vue'

const props = defineProps({
  attachments: { type: Array, required: true },
  transcriptions: { type: Object, default: null },
  createdAt: { type: String, default: '' }
})

const isAudio = (attachment) => (attachment.content_type || '').startsWith('audio/')

const audioAttachments = computed(() => (props.attachments || []).filter(isAudio))
const otherAttachments = computed(() => (props.attachments || []).filter((a) => !isAudio(a)))

const isImage = (attachment) => (attachment.content_type || '').startsWith('image/')

const imageAttachments = computed(() =>
  (props.attachments || []).filter(isImage)
)

const lightboxOpen = ref(false)
const lightboxIndex = ref(0)

const openLightbox = (attachment) => {
  const idx = imageAttachments.value.findIndex((a) => a.uuid === attachment.uuid)
  lightboxIndex.value = idx >= 0 ? idx : 0
  lightboxOpen.value = true
}
</script>
