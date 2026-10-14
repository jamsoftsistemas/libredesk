import { computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { createRemoteLookup } from '@/utils/remote-lookup'
import api from '@/api'

// Backs the company picker on the contact form with a remote, searchable list.
export const useCompanyStore = defineStore('company', () => {
    const emitter = useEmitter()
    const showFetchError = (error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    const lookup = createRemoteLookup({
        fetchRows: (params) => api.getCompaniesCompact(params),
        onError: showFetchError
    })

    const toOptions = (companies) => companies.map(company => ({
        label: company.name,
        value: String(company.id),
    }))
    const companies = lookup.rows
    const options = computed(() => toOptions(companies.value))
    const searchCompanies = async (query) => toOptions(await lookup.search(query))

    return {
        companies,
        options,
        fetchCompanies: lookup.fetchFirstPage,
        searchCompanies,
        ensureCompanyIDs: lookup.ensureIDs,
    }
})
