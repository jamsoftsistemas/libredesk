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

        <FormField v-slot="{ componentField }" name="visibility">
            <FormItem>
                <FormLabel>{{ $t('globals.terms.visibility') }}</FormLabel>
                <FormControl>
                    <Select v-bind="componentField">
                        <SelectTrigger>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem value="all">{{ $t('globals.terms.all') }}</SelectItem>
                                <SelectItem value="team">{{ $t('globals.terms.team') }}</SelectItem>
                                <SelectItem value="inbox">{{ $t('globals.terms.inbox') }}</SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField
            v-if="formValues.visibility === 'team'"
            v-slot="{ componentField }"
            name="team_id"
        >
            <FormItem>
                <FormLabel>{{ $t('globals.terms.team') }}</FormLabel>
                <FormControl>
                    <SelectTeamCombobox v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField
            v-if="formValues.visibility === 'inbox'"
            v-slot="{ componentField }"
            name="inbox_id"
        >
            <FormItem>
                <FormLabel>{{ $t('globals.terms.inbox') }}</FormLabel>
                <FormControl>
                    <SelectInboxCombobox v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <!-- Form submit button slot -->
        <slot name="footer" ></slot>
    </form>
</template>

<script setup>
import { useFormValues } from 'vee-validate'
import {
    FormControl,
    FormDescription,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@shared-ui/components/ui/form'
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue
} from '@shared-ui/components/ui/select'
import { Input } from '@shared-ui/components/ui/input'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import SelectInboxCombobox from '@main/components/combobox/SelectInboxCombobox.vue'

const formValues = useFormValues()
</script>
