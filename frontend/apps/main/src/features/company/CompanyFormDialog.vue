<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ company ? t('company.editCompany') : t('company.new') }}</DialogTitle>
      </DialogHeader>
      <CompanyForm
        :formLoading="loading"
        :onSubmit="onSubmit"
        :submitLabel="company ? t('globals.messages.update') : t('globals.messages.create')"
      />
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@shared-ui/components/ui/dialog'
import CompanyForm from './CompanyForm.vue'
import { createFormSchema } from './formSchema.js'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const props = defineProps({
  open: {
    type: Boolean,
    default: false
  },
  company: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:open', 'saved'])

const { t } = useI18n()
const emitter = useEmitter()
const loading = ref(false)

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

watch(
  () => props.open,
  (val) => {
    if (val) {
      form.resetForm({
        values: {
          external_id: props.company?.external_id || '',
          name: props.company?.name || '',
          document: props.company?.document || '',
          city: props.company?.city || '',
          system: props.company?.system || ''
        }
      })
    }
  }
)

const onSubmit = form.handleSubmit(async (values) => {
  loading.value = true
  try {
    const { data } = props.company
      ? await api.updateCompany(props.company.id, values)
      : await api.createCompany(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.savedSuccessfully') })
    emit('update:open', false)
    emit('saved', data.data)
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    loading.value = false
  }
})
</script>
