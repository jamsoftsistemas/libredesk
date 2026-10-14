<template>
  <div class="min-h-screen flex flex-col">
    <div class="flex flex-wrap gap-4 pb-4">
      <div class="flex items-center gap-4 mb-4 w-full">
        <Input
          type="text"
          class="max-w-sm"
          v-model="searchTerm"
          :placeholder="$t('company.searchByName')"
          @input="fetchCompaniesDebounced"
        />

        <Button
          v-if="userStore.can('companies:write')"
          size="sm"
          class="flex items-center h-8 ml-auto"
          @click="openCreateDialog"
        >
          {{ $t('company.new') }}
        </Button>
      </div>

      <div v-if="loading" class="flex flex-col gap-4 w-full">
        <Card v-for="i in perPage" :key="i" class="p-4 flex-shrink-0">
          <div class="flex items-center gap-4">
            <Skeleton class="h-10 w-10 rounded-full" />
            <div class="space-y-2">
              <Skeleton class="h-3 w-[160px]" />
              <Skeleton class="h-3 w-[140px]" />
            </div>
          </div>
        </Card>
      </div>

      <template v-else>
        <Card v-for="company in companies" :key="company.id" class="p-4 w-full">
          <div class="flex items-center gap-4">
            <Avatar class="h-10 w-10 border">
              <AvatarFallback class="text-sm font-medium">
                <Building2 size="18" />
              </AvatarFallback>
            </Avatar>

            <div class="space-y-1 overflow-hidden flex-1">
              <h4 class="text-sm font-semibold truncate">{{ company.name }}</h4>
              <div class="flex items-center gap-3 text-xs text-muted-foreground">
                <span v-if="company.document" class="truncate">{{ company.document }}</span>
                <span v-if="company.city" class="truncate">{{ company.city }}</span>
                <span v-if="company.system" class="truncate">{{ company.system }}</span>
              </div>
              <div
                v-if="company.external_id"
                class="flex items-center gap-1 text-xs text-muted-foreground"
              >
                <IdCardIcon size="12" class="flex-shrink-0" />
                <span class="truncate">{{ company.external_id }}</span>
              </div>
            </div>

            <DropdownMenu v-if="userStore.can('companies:write')">
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" class="w-8 h-8 p-0">
                  <MoreVertical class="w-4 h-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem @click="openEditDialog(company)">
                  <Pencil class="mr-2 h-4 w-4" />
                  {{ $t('globals.messages.edit') }}
                </DropdownMenuItem>
                <DropdownMenuSeparator v-if="userStore.can('companies:delete')" />
                <DropdownMenuItem
                  v-if="userStore.can('companies:delete')"
                  @click="confirmDelete(company)"
                  class="text-destructive focus:text-destructive"
                >
                  <Trash class="mr-2 h-4 w-4" />
                  {{ $t('globals.messages.delete') }}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </Card>
        <div v-if="companies.length === 0" class="flex items-center justify-center w-full h-32">
          <p class="text-lg text-muted-foreground">{{ $t('company.noCompaniesFound') }}</p>
        </div>
      </template>
    </div>

    <PaginationBar v-model:page="page" v-model:per-page="perPage" :total-pages="totalPages" />
  </div>

  <CompanyFormDialog
    v-model:open="showFormDialog"
    :company="selectedCompany"
    @saved="fetchCompanies"
  />

  <AlertDialog :open="deleteAlertOpen" @update:open="deleteAlertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('company.deleteCompany') }}</AlertDialogTitle>
        <AlertDialogDescription>{{ $t('confirm.deleteCompany') }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDelete">
          {{ $t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { Card } from '@shared-ui/components/ui/card'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { Avatar, AvatarFallback } from '@shared-ui/components/ui/avatar'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
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
import { Building2, IdCardIcon, MoreVertical, Pencil, Trash } from 'lucide-vue-next'
import { useDebounceFn } from '@vueuse/core'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import CompanyFormDialog from './CompanyFormDialog.vue'
import { useUserStore } from '@main/stores/user'
import api from '@main/api'

const companies = ref([])
const loading = ref(false)
const page = ref(1)
const perPage = ref(15)
const totalPages = ref(0)
const searchTerm = ref('')
const showFormDialog = ref(false)
const selectedCompany = ref(null)
const deleteAlertOpen = ref(false)
const companyToDelete = ref(null)
const emitter = useEmitter()
const userStore = useUserStore()
let fetchRequestId = 0

const fetchCompaniesDebounced = useDebounceFn(() => {
  if (page.value === 1) {
    fetchCompanies()
  } else {
    page.value = 1
  }
}, 300)

const fetchCompanies = async () => {
  const requestId = ++fetchRequestId
  loading.value = true
  try {
    const response = await api.getCompanies({
      page: page.value,
      page_size: perPage.value,
      q: searchTerm.value
    })
    if (requestId !== fetchRequestId) return
    companies.value = response.data.data.results
    totalPages.value = response.data.data.total_pages
  } catch (error) {
    if (requestId !== fetchRequestId) return
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (requestId === fetchRequestId) loading.value = false
  }
}

const openCreateDialog = () => {
  selectedCompany.value = null
  showFormDialog.value = true
}

const openEditDialog = (company) => {
  selectedCompany.value = company
  showFormDialog.value = true
}

const confirmDelete = (company) => {
  companyToDelete.value = company
  deleteAlertOpen.value = true
}

const handleDelete = async () => {
  try {
    await api.deleteCompany(companyToDelete.value.id)
    deleteAlertOpen.value = false
    fetchCompanies()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

watch([page, perPage], fetchCompanies)

onMounted(() => {
  fetchCompanies()
})
</script>
