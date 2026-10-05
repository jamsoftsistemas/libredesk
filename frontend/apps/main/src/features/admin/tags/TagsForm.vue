<template>
    <form>
        <FormField v-slot="{ componentField }" name="name">
            <FormItem>
                <FormLabel>{{$t('globals.terms.name')}}</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="billing" v-bind="componentField" />
                </FormControl>
                <FormDescription></FormDescription>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField, handleChange }" name="team_ids">
            <FormItem>
                <FormLabel>{{ $t('globals.terms.team', 2) }}</FormLabel>
                <FormControl>
                    <SelectTeamCombobox
                        multiple
                        :model-value="componentField.modelValue"
                        @update:model-value="handleChange"
                    />
                </FormControl>
                <FormDescription>{{ $t('admin.conversationTags.teamIds.description') }}</FormDescription>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField, handleChange }" name="inbox_ids">
            <FormItem>
                <FormLabel>{{ $t('globals.terms.inbox', 2) }}</FormLabel>
                <FormControl>
                    <SelectInboxCombobox
                        multiple
                        :model-value="componentField.modelValue"
                        @update:model-value="handleChange"
                    />
                </FormControl>
                <FormDescription>{{ $t('admin.conversationTags.inboxIds.description') }}</FormDescription>
                <FormMessage />
            </FormItem>
        </FormField>

        <!-- Form submit button slot -->
        <slot name="footer" ></slot>
    </form>
</template>

<script setup>
import {
    FormControl,
    FormDescription,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import SelectInboxCombobox from '@main/components/combobox/SelectInboxCombobox.vue'
</script>
