<template>
  <SelectTag
    v-if="multiple"
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
  />
  <SelectComboBox
    v-else
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useInboxStore } from '@/stores/inbox'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { SelectTag } from '@shared-ui/components/ui/select'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: {
    type: [String, Number, Array],
    default: undefined
  },
  placeholder: {
    type: String,
    default: ''
  },
  multiple: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()
const inboxStore = useInboxStore()

const placeholderText = computed(() => props.placeholder || t('placeholders.selectInbox'))

const items = computed(() => inboxStore.options)

onMounted(inboxStore.fetchInboxes)
</script>
