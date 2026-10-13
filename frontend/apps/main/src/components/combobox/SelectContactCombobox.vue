<template>
  <SelectComboBox
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="contactsStore.searchContacts"
    type="user"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useContactsStore } from '@/stores/contacts'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: undefined
  },
  placeholder: {
    type: String,
    default: ''
  },
  prependItems: {
    type: Array,
    default: () => []
  }
})

const { t } = useI18n()
const contactsStore = useContactsStore()

const placeholderText = computed(() => props.placeholder || t('globals.terms.contact'))

const items = computed(() => [...props.prependItems, ...contactsStore.options])

onMounted(contactsStore.fetchContacts)

watch(
  () => props.modelValue,
  (value) => {
    contactsStore.ensureContactIDs([value])
  },
  { immediate: true }
)
</script>
