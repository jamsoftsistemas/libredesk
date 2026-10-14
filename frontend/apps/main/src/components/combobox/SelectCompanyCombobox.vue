<template>
  <SelectComboBox
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="companyStore.searchCompanies"
    type="company"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useCompanyStore } from '@/stores/company'
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
  includeNone: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()
const companyStore = useCompanyStore()

const placeholderText = computed(() => props.placeholder || t('placeholders.selectCompany'))

const items = computed(() => {
  const prepended = props.includeNone
    ? [{ value: 'none', label: t('globals.terms.none') }]
    : []
  return [...prepended, ...companyStore.options]
})

onMounted(companyStore.fetchCompanies)

watch(
  () => props.modelValue,
  (value) => {
    companyStore.ensureCompanyIDs([value])
  },
  { immediate: true }
)
</script>
