<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <div :class="FIELD_GRID_CLASS">
      <div :class="FIELD_COLUMN_CLASS">
        <FormField v-slot="{ componentField }" name="external_id">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.externalId') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <div :class="FIELD_COLUMN_CLASS">
        <FormField v-slot="{ componentField }" name="name">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.name') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <div :class="FIELD_GRID_CLASS">
      <div :class="FIELD_COLUMN_CLASS">
        <FormField v-slot="{ componentField }" name="document">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.document') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <div :class="FIELD_COLUMN_CLASS">
        <FormField v-slot="{ componentField }" name="city">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.city') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <div :class="FIELD_GRID_CLASS">
      <div :class="FIELD_COLUMN_CLASS">
        <FormField v-slot="{ componentField }" name="system">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.system') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <div>
      <Button type="submit" :isLoading="formLoading" :disabled="formLoading">
        {{ submitLabel }}
      </Button>
    </div>
  </form>
</template>

<script setup>
const FIELD_COLUMN_CLASS = 'min-w-0'
const FIELD_GRID_CLASS = 'grid gap-6 sm:grid-cols-2'

import {
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { useI18n } from 'vue-i18n'

defineProps(['formLoading', 'onSubmit', 'submitLabel'])

const { t } = useI18n()
</script>
