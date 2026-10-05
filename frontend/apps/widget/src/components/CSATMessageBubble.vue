<template>
  <div class="p-4 rounded-2xl text-sm bg-background text-foreground border border-border">
    <div v-if="!isSubmitted">
      <p class="mb-3">{{ csatMessage }}</p>

      <div class="flex gap-3 mb-4">
        <button
          v-for="rating in ratings"
          :key="rating.value"
          @click="selectedRating = rating.value"
          :aria-label="rating.text"
          class="flex flex-col items-center p-2 rounded-md cursor-pointer hover:bg-muted transition-all"
          :class="{ 'scale-125 bg-muted': selectedRating === rating.value }"
        >
          <span class="text-xl mb-1">{{ rating.emoji }}</span>
          <span class="text-xs text-muted-foreground">{{ rating.text }}</span>
        </button>
      </div>

      <div class="mb-4">
        <label class="text-xs text-muted-foreground mb-2 block">
          {{ t('globals.messages.additionalFeedback') }}
        </label>
        <Textarea
          v-model="feedback"
          :placeholder="$t('globals.terms.tellUsMore')"
          class="min-h-[60px]"
          maxlength="500"
        />
        <div class="text-xs text-muted-foreground text-right mt-1">{{ feedback.length }}/500</div>
      </div>

      <Button
        type="button"
        class="w-full"
        :disabled="(!selectedRating && !feedback.trim()) || isSubmitting"
        @click="submitRating"
      >
        <Spinner v-if="isSubmitting" size="sm" :absolute="false" :center="false" />
        {{ isSubmitting ? t('globals.messages.submitting') : t('globals.messages.submitFeedback') }}
      </Button>
    </div>

    <div v-else class="text-center py-2">
      <p class="mb-3">{{ t('globals.messages.thankYouFeedback') }}</p>
      
      <!-- Show submitted rating if provided -->
      <div v-if="csatMeta.submitted_rating" class="mb-2">
        <span class="text-lg">{{ csatRatingEmoji(csatMeta.submitted_rating, ratingOverrides) }}</span>
        <span class="text-xs text-muted-foreground ml-2">{{
          csatRatingLabel(csatMeta.submitted_rating, ratingOverrides, t)
        }}</span>
      </div>
      
      <!-- Show submitted feedback if provided -->
      <div v-if="csatMeta.submitted_feedback" class="text-xs text-muted-foreground italic">
        "{{ csatMeta.submitted_feedback }}"
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@widget/api/index.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { buildCSATRatings, csatRatingEmoji, csatRatingLabel } from '@shared-ui/utils/csat.js'

const props = defineProps({
  message: { type: Object, required: true }
})

const emit = defineEmits(['submitted'])

const widgetStore = useWidgetStore()
const selectedRating = ref(null)
const feedback = ref('')
const isSubmitting = ref(false)

const csatMeta = computed(() => {
  return props.message.meta
})

const isSubmitted = computed(() => csatMeta.value.csat_submitted === true)
const csatUuid = computed(() => csatMeta.value.csat_uuid || '')

const { t } = useI18n()

const ratingOverrides = computed(() => widgetStore.config.csat_ratings)
const csatMessage = computed(
  () => widgetStore.config.csat_message || t('globals.messages.pleaseRateConversation')
)
const ratings = computed(() => buildCSATRatings(ratingOverrides.value, t))

const submitRating = async () => {
  if ((!selectedRating.value && !feedback.value.trim()) || !csatUuid.value) return
  isSubmitting.value = true
  try {
    await api.submitCSATResponse(csatUuid.value, selectedRating.value || 0, feedback.value)
    emit('submitted', {
      rating: selectedRating.value,
      feedback: feedback.value,
      message_uuid: props.message.uuid
    })
  } finally {
    isSubmitting.value = false
  }
}

</script>
