<template>
  <div class="space-y-6" :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
    <Spinner v-if="isLoading" />
    <div class="flex items-start justify-between gap-4">
      <p class="text-sm-muted">{{ $t('admin.automation.transfer.description') }}</p>
      <Button type="button" class="shrink-0" @click="addRow">
        {{ $t('admin.automation.transfer.add') }}
      </Button>
    </div>

    <div
      v-if="!isLoading && rows.length === 0"
      class="flex flex-col items-center justify-center py-12 px-4"
    >
      <p class="text-muted-foreground">{{ $t('admin.automation.noRulesFound') }}</p>
    </div>

    <div v-for="row in rows" :key="row.key" class="box p-5 space-y-4">
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <Badge v-if="row.id && row.enabled">{{ $t('globals.terms.enabled') }}</Badge>
          <Badge v-else-if="row.id" variant="secondary">{{ $t('globals.terms.disabled') }}</Badge>
        </div>
        <div class="flex items-center gap-2">
          <Button
            v-if="row.id"
            type="button"
            variant="outline"
            size="sm"
            @click="toggleRow(row)"
          >
            {{ row.enabled ? $t('globals.messages.disable') : $t('globals.messages.enable') }}
          </Button>
          <Button type="button" variant="ghost" size="sm" class="h-8 w-8 p-0" @click="askDelete(row)">
            <Trash size="16" />
          </Button>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label>{{ $t('admin.automation.transfer.companies') }}</Label>
          <SelectTag
            :model-value="row.companyIds"
            :items="companyStore.options"
            :placeholder="t('placeholders.selectCompany')"
            :search="companyStore.searchCompanies"
            @update:modelValue="(v) => onCompaniesChange(row, v)"
          />
        </div>
        <div class="space-y-2">
          <Label>{{ $t('admin.automation.transfer.agent') }}</Label>
          <SelectAgentCombobox
            multiple
            :model-value="row.agentIds"
            @update:modelValue="(v) => (row.agentIds = (v || []).map(String))"
          />
        </div>
      </div>

      <Button type="button" :isLoading="row.saving" @click="saveRow(row)">
        {{ $t('globals.messages.save') }}
      </Button>
    </div>
  </div>

  <AlertDialog :open="!!deleting" @update:open="(o) => !o && (deleting = null)">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>{{ $t('automation.deletionConfirmation') }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="confirmDelete">{{
          $t('globals.messages.delete')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Trash } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import { Label } from '@shared-ui/components/ui/label'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { SelectTag } from '@shared-ui/components/ui/select'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import SelectAgentCombobox from '@/components/combobox/SelectAgentCombobox.vue'
import { useCompanyStore } from '@/stores/company'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@/api'

const TYPE = 'transfer'

const { t } = useI18n()
const emitter = useEmitter()
const companyStore = useCompanyStore()
const isLoading = ref(false)
const rows = ref([])
const deleting = ref(null)
let keySeq = 0

const toast = (description, variant) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, variant ? { variant, description } : { description })

// Builds a row from a stored rule record.
const fromRecord = (rec) => {
  const rule = rec.rules?.[0]
  const cond = rule?.groups?.flatMap((g) => g.rules || []).find((r) => r.field === 'contact_company')
  const action = rule?.actions?.find((a) => a.type === 'assign_user')
  const companyIds = cond?.value ? cond.value.split(',').filter(Boolean) : []
  companyStore.ensureCompanyIDs(companyIds)
  return {
    key: ++keySeq,
    id: rec.id,
    name: rec.name,
    enabled: rec.enabled,
    companyIds,
    agentIds: (action?.value || []).map(String),
    saving: false
  }
}

// Builds the rule payload the engine expects: contact company in [ids] -> assign agent.
const toPayload = (row) => ({
  name: row.name || t('admin.automation.transfer.ruleName', { n: row.companyIds.length }),
  description: '',
  type: TYPE,
  enabled: row.id ? row.enabled : true,
  events: [],
  rules: [
    {
      group_operator: 'OR',
      groups: [
        {
          logical_op: 'OR',
          rules: [
            {
              field: 'contact_company',
              field_type: 'conversation',
              operator: 'in',
              value: row.companyIds.join(',')
            }
          ]
        },
        { logical_op: 'OR', rules: [] }
      ],
      actions: [{ type: 'assign_user', value: row.agentIds }]
    }
  ]
})

const fetchRows = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAutomationRules(TYPE)
    rows.value = resp.data.data.map(fromRecord)
  } finally {
    isLoading.value = false
  }
}

const addRow = () => {
  rows.value.push({
    key: ++keySeq,
    id: 0,
    name: '',
    enabled: true,
    companyIds: [],
    agentIds: [],
    saving: false
  })
}

const onCompaniesChange = (row, values) => {
  row.companyIds = values.map(String)
}

const saveRow = async (row) => {
  if (row.companyIds.length === 0 || row.agentIds.length === 0) {
    toast(t('automation.invalidTransferRule'), 'warning')
    return
  }
  try {
    row.saving = true
    const payload = toPayload(row)
    if (row.id) {
      await api.updateAutomationRule(row.id, payload)
    } else {
      const resp = await api.createAutomationRule(payload)
      row.id = resp.data.data.id
    }
    row.name = payload.name
    toast(t('globals.messages.savedSuccessfully'))
  } catch (error) {
    toast(handleHTTPError(error).message, 'destructive')
  } finally {
    row.saving = false
  }
}

const toggleRow = async (row) => {
  try {
    const resp = await api.toggleAutomationRule(row.id)
    row.enabled = resp.data.data.enabled
  } catch (error) {
    toast(handleHTTPError(error).message, 'destructive')
  }
}

const askDelete = (row) => {
  // Unsaved rows have nothing to delete on the server.
  if (!row.id) {
    rows.value = rows.value.filter((r) => r.key !== row.key)
    return
  }
  deleting.value = row
}

const confirmDelete = async () => {
  const row = deleting.value
  deleting.value = null
  try {
    await api.deleteAutomationRule(row.id)
    rows.value = rows.value.filter((r) => r.key !== row.key)
  } catch (error) {
    toast(handleHTTPError(error).message, 'destructive')
  }
}

onMounted(() => {
  companyStore.fetchCompanies()
  fetchRows()
})
</script>
