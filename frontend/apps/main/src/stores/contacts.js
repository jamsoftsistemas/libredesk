import { computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { createRemoteLookup } from '@/utils/remote-lookup'
import api from '@/api'

// Backs contact pickers (e.g. the search/filter "contact" field) with a remote, searchable list.
export const useContactsStore = defineStore('contacts', () => {
    const emitter = useEmitter()
    const showFetchError = (error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    const lookup = createRemoteLookup({
        fetchRows: (params) => api.getContactsCompact(params),
        onError: showFetchError
    })

    const toOptions = (contacts) => contacts.map(contact => ({
        label: [contact.first_name, contact.last_name].filter(Boolean).join(' ') || contact.email,
        value: String(contact.id),
        avatar_url: contact.avatar_url,
    }))
    const contacts = lookup.rows
    const options = computed(() => toOptions(contacts.value))
    const searchContacts = async (query) => toOptions(await lookup.search(query))

    return {
        contacts,
        options,
        fetchContacts: lookup.fetchFirstPage,
        searchContacts,
        ensureContactIDs: lookup.ensureIDs,
    }
})
