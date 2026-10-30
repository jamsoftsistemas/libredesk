<template>
  <form @submit.prevent="createConversation" novalidate class="flex flex-col flex-1 min-h-0">
    <div class="shrink-0">
      <div :class="[ROW_CLASS, 'flex items-center gap-2']">
        <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.from') }}</span>
        <Select v-model="inboxId">
          <SelectTrigger :class="ROW_INPUT_CLASS" :aria-label="t('globals.terms.inbox')">
            <SelectValue :placeholder="t('placeholders.selectInbox')" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem
                v-for="option in inboxStore.livechatOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <div :class="ROW_CLASS">
        <div class="flex items-center gap-2">
          <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.to') }}</span>
          <Input
            ref="contactInputRef"
            type="text"
            v-model="contactQuery"
            :class="ROW_INPUT_CLASS"
            :aria-label="t('globals.terms.contact', 1)"
            :placeholder="t('conversation.livechat.searchContact')"
            @input="onQueryInput"
            @keydown="handleSearchKeydown"
            @blur="clearSearchResults"
            autocomplete="off"
          />
        </div>

        <ContactSearchResults
          :results="searchResults"
          :highlighted-index="highlightedIndex"
          @select="selectContact"
        >
          <template #default="{ contact }">
            <div class="min-w-0">
              <p class="font-medium">{{ contact.first_name }} {{ contact.last_name }}</p>
              <p v-if="contact.email" class="text-xs text-muted-foreground truncate">
                {{ contact.email }}
              </p>
              <p v-if="contact.phone_number" class="text-xs text-muted-foreground truncate">
                {{ contact.phone_number }}
              </p>
            </div>
          </template>
        </ContactSearchResults>
        <p v-if="!selectedContact" class="px-4 pb-2 text-xs text-muted-foreground">
          {{ $t('conversation.livechat.registeredContactsOnly') }}
        </p>
      </div>

      <div :class="[ROW_CLASS, 'grid grid-cols-1 sm:grid-cols-2 sm:gap-4']">
        <div class="flex items-center gap-2 min-w-0">
          <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.team', 1) }}</span>
          <SelectTeamCombobox
            v-model="teamId"
            include-none
            :button-class="isUnset(teamId) ? ROW_COMBOBOX_EMPTY_CLASS : ROW_COMBOBOX_CLASS"
          />
        </div>
        <div class="flex items-center gap-2 min-w-0">
          <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.agent', 1) }}</span>
          <SelectAgentCombobox
            v-model="agentId"
            include-none
            :button-class="isUnset(agentId) ? ROW_COMBOBOX_EMPTY_CLASS : ROW_COMBOBOX_CLASS"
          />
        </div>
      </div>
    </div>

    <div class="flex-1 flex flex-col min-h-0 px-3 pt-3">
      <Textarea
        v-model="content"
        rows="6"
        class="flex-1 resize-none"
        :placeholder="t('conversation.uazapi.messagePlaceholder')"
      />
    </div>

    <div class="flex justify-end px-3 py-2 shrink-0">
      <Button type="submit" :disabled="!canSubmit || loading" :isLoading="loading">
        {{ $t('globals.messages.send') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useInboxStore } from '@main/stores/inbox'
import { useUserStore } from '@main/stores/user'
import { useContactSearch } from './useContactSearch.js'
import ContactSearchResults from './ContactSearchResults.vue'
import api from '@/api'
import { useNewConversationDraft } from './useNewConversationDraft.js'
import {
  ROW_CLASS,
  ROW_LABEL_CLASS,
  ROW_INPUT_CLASS,
  ROW_COMBOBOX_CLASS,
  ROW_COMBOBOX_EMPTY_CLASS,
  isUnset
} from '@/features/conversation/composerRowClasses.js'

const emit = defineEmits(['close'])
const props = defineProps({
  initialContact: { type: Object, default: null }
})

const { t } = useI18n()
const emitter = useEmitter()
const inboxStore = useInboxStore()
const userStore = useUserStore()

const inboxId = ref('')
const teamId = ref(null)
const agentId = ref(userStore.userID ? String(userStore.userID) : 'none')
const loading = ref(false)
const content = ref('')

const selectedContact = ref(null)
const contactQuery = ref('')
const contactInputRef = ref(null)

const contactLabel = (c) => [c.first_name, c.last_name].filter(Boolean).join(' ') || c.email || ''

const { searchResults, highlightedIndex, handleSearchContacts, handleSearchKeydown, selectContact, clearSearchResults } =
  useContactSearch({
    getQuery: () => contactQuery.value,
    onSelect: (contact) => {
      selectedContact.value = contact
      contactQuery.value = contactLabel(contact)
    }
  })

// Typing after a pick discards it: only a contact chosen from the results can be messaged.
const onQueryInput = () => {
  selectedContact.value = null
  handleSearchContacts()
}

const draft = useNewConversationDraft('livechat')
const DRAFT_FIELDS = { inboxId, teamId, agentId, contactQuery, content }

onMounted(() => {
  if (draft.value) {
    selectedContact.value = draft.value.contact || null
    Object.entries(DRAFT_FIELDS).forEach(([key, field]) => {
      if (key in draft.value) field.value = draft.value[key]
    })
  }
  if (props.initialContact) selectContact(props.initialContact)
})

defineExpose({ focus: () => contactInputRef.value?.$el?.focus() })

watch([selectedContact, ...Object.values(DRAFT_FIELDS)], () => {
  const hasContent = content.value.trim() || selectedContact.value
  draft.value = hasContent
    ? {
        contact: selectedContact.value,
        ...Object.fromEntries(Object.entries(DRAFT_FIELDS).map(([k, f]) => [k, f.value]))
      }
    : null
}, { deep: true })

watch(
  () => props.initialContact,
  (contact) => {
    if (contact) selectContact(contact)
  }
)

const canSubmit = computed(
  () => !!inboxId.value && !!selectedContact.value && content.value.trim() !== ''
)

const createConversation = async () => {
  if (!canSubmit.value) return
  loading.value = true
  try {
    await api.createConversation({
      inbox_id: Number(inboxId.value),
      team_id: teamId.value && teamId.value !== 'none' ? Number(teamId.value) : null,
      agent_id: agentId.value && agentId.value !== 'none' ? Number(agentId.value) : null,
      contact_id: selectedContact.value.id,
      content: content.value
    })
    draft.value = null
    emit('close')
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}
</script>
