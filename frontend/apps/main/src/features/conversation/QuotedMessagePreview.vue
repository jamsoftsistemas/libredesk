<template>
  <div class="flex items-stretch gap-2 rounded-md bg-muted/60 pr-2 mb-2 overflow-hidden">
    <div class="w-[3px] shrink-0 bg-primary" />
    <div class="min-w-0 flex-1 py-1.5">
      <p class="text-xs font-medium text-foreground truncate">{{ authorName }}</p>
      <p class="text-xs text-muted-foreground truncate">{{ snippet }}</p>
    </div>
    <button
      type="button"
      class="shrink-0 self-center text-muted-foreground hover:text-foreground transition-colors duration-150"
      :aria-label="$t('globals.messages.cancel')"
      @click="emit('cancel')"
    >
      <X class="w-4 h-4" />
    </button>
  </div>
</template>

<script setup>
import { X } from 'lucide-vue-next'

defineProps({
  authorName: { type: String, default: '' },
  snippet: { type: String, default: '' }
})

const emit = defineEmits(['cancel'])
</script>
