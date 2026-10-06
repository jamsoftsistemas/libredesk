<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem>
        <SwitchField
          :title="$t('globals.terms.enabled')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="prompt_tags_on_reply">
      <FormItem>
        <SwitchField
          :title="$t('admin.inbox.promptTagsOnReply')"
          :description="$t('admin.inbox.promptTagsOnReply.description')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
      <FormItem>
        <SwitchField
          :title="$t('admin.inbox.csatSurveys')"
          :description="$t('admin.inbox.csatSurveys.description_1')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <FormField v-slot="{ componentField }" name="reopen_window_hours">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.whatsapp.reopenWindow') }}</FormLabel>
          <FormControl>
            <Input type="number" min="0" placeholder="48" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.whatsapp.reopenWindow.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.uazapi.gatewaySettings') }}</h3>
      <p class="text-sm text-muted-foreground flex items-start gap-1.5">
        <Lightbulb class="size-4 mt-0.5 shrink-0" />
        <span>{{ $t('admin.inbox.uazapi.gatewaySettings.description') }}</span>
      </p>

      <FormField v-slot="{ componentField }" name="config.base_url">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.uazapi.baseUrl') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="https://free.uazapi.com" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.uazapi.baseUrl.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <div class="grid grid-cols-2 gap-4">
        <FormField v-slot="{ componentField }" name="config.instance_token">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.uazapi.instanceToken') }}</FormLabel>
            <FormControl>
              <Input type="password" placeholder="••••••••" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.uazapi.instanceToken.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.instance_name">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.uazapi.instanceName') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.uazapi.instanceName.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <FormField v-slot="{ componentField }" name="config.admin_token">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.uazapi.adminToken') }}</FormLabel>
          <FormControl>
            <Input type="password" placeholder="••••••••" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.uazapi.adminToken.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>

    <UazapiConnectPanel v-if="!isNewForm && initialValues?.id" :inbox-id="initialValues.id" />
  </form>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './uazapiFormSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import UazapiConnectPanel from './UazapiConnectPanel.vue'
import { Lightbulb } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: ''
  },
  isNewForm: {
    type: Boolean,
    default: false
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? t('globals.messages.create') : t('globals.messages.save'))
  )
})

const form = useForm({
  validationSchema: computed(() => toTypedSchema(createFormSchema(t))),
  initialValues: {
    name: '',
    enabled: true,
    csat_enabled: false,
    prompt_tags_on_reply: false,
    reopen_window_hours: 0,
    config: {
      base_url: '',
      instance_name: '',
      instance_token: '',
      admin_token: ''
    }
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  await props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (!newValues || Object.keys(newValues).length === 0) {
      return
    }
    form.setValues(
      {
        ...newValues,
        reopen_window_hours: newValues.reopen_window_hours ?? 0,
        config: { ...(newValues.config || {}) }
      },
      false
    )
  },
  { deep: true, immediate: true }
)
</script>
