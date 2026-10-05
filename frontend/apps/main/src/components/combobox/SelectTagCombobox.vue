<template>
  <SelectTag
    v-if="multiple"
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="searchTags"
  />
  <SelectComboBox
    v-else
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="searchTags"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTagStore } from '@/stores/tag'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { SelectTag } from '@shared-ui/components/ui/select'
import api from '@/api'

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
  },
  // Conversations store tags as names; filters store them as ids.
  valueField: {
    type: String,
    default: 'name',
    validator: (value) => ['name', 'id'].includes(value)
  },
  // When set, restricts suggestions to tags visible in this team/inbox's context (plus global
  // tags), e.g. the team/inbox of the conversation currently being tagged.
  teamId: {
    type: [String, Number],
    default: undefined
  },
  inboxId: {
    type: [String, Number],
    default: undefined
  }
})

const { t } = useI18n()
const tagStore = useTagStore()

const placeholderText = computed(() => props.placeholder || t('placeholders.selectTags'))

const isScoped = computed(() => props.teamId !== undefined || props.inboxId !== undefined)

const toOption = (tag) =>
  props.valueField === 'id'
    ? { label: tag.name, value: String(tag.id) }
    : { label: tag.name, value: tag.name }

const scopedTags = ref([])
const fetchScopedTags = async (query = '') => {
  try {
    const params = { q: query }
    if (props.teamId !== undefined) params.team_id = props.teamId
    if (props.inboxId !== undefined) params.inbox_id = props.inboxId
    const response = await api.getTags(params)
    scopedTags.value = response?.data?.data || []
  } catch {
    scopedTags.value = []
  }
  return scopedTags.value
}

const items = computed(() => {
  if (isScoped.value) return scopedTags.value.map(toOption)
  return props.valueField === 'id'
    ? tagStore.tagOptions
    : tagStore.tagNames.map((name) => ({ label: name, value: name }))
})

const searchTags = (query) =>
  isScoped.value
    ? fetchScopedTags(query).then((tags) => tags.map(toOption))
    : props.valueField === 'id'
      ? tagStore.searchTagOptions(query)
      : tagStore.searchTagNames(query)

onMounted(() => (isScoped.value ? fetchScopedTags() : tagStore.fetchTags()))

watch(
  () => [props.teamId, props.inboxId],
  () => {
    if (isScoped.value) fetchScopedTags()
  }
)

watch(
  () => props.modelValue,
  (value) => {
    if (props.valueField !== 'id') return
    const ids = Array.isArray(value) ? value : [value]
    tagStore.ensureTagIDs(ids)
  },
  { immediate: true }
)
</script>
