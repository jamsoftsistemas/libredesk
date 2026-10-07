<template>
  <div class="h-full">
    <div class="flex flex-col space-y-5">
      <div class="space-y-1">
        <span class="sub-title">{{ $t('account.publicAvatar') }}</span>
        <p class="text-muted-foreground text-xs">{{ $t('account.changeAvatar') }}</p>
      </div>
      <AvatarUpload
        :src="userStore.avatar"
        :initials="userStore.getInitials"
        :label="$t('globals.messages.upload')"
        :disabled="isSaving"
        @upload="onCropped"
        @remove="removeAvatar"
      />

      <Button
        class="self-start"
        @click="saveUser"
        :isLoading="isSaving"
        :disabled="!pendingFile"
      >
        {{ $t('globals.messages.saveChanges') }}
      </Button>

      <div class="space-y-1 pt-5">
        <span class="sub-title">{{ $t('account.changePassword') }}</span>
        <p class="text-muted-foreground text-xs">{{ $t('account.changePasswordDescription') }}</p>
      </div>

      <form @submit.prevent="changePassword" class="space-y-3 max-w-sm">
        <div class="space-y-2">
          <Label for="currentPassword">{{ $t('account.currentPassword') }}</Label>
          <div class="relative">
            <Input
              id="currentPassword"
              :type="showCurrentPassword ? 'text' : 'password'"
              autocomplete="current-password"
              v-model="passwordForm.currentPassword"
              class="pr-10"
            />
            <button
              type="button"
              :aria-label="showCurrentPassword ? $t('auth.hidePassword') : $t('auth.showPassword')"
              class="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground"
              @click="showCurrentPassword = !showCurrentPassword"
            >
              <Eye v-if="!showCurrentPassword" class="w-5 h-5" />
              <EyeOff v-else class="w-5 h-5" />
            </button>
          </div>
        </div>

        <div class="space-y-2">
          <Label for="newPassword">{{ $t('auth.newPassword') }}</Label>
          <div class="relative">
            <Input
              id="newPassword"
              :type="showNewPassword ? 'text' : 'password'"
              autocomplete="new-password"
              v-model="passwordForm.newPassword"
              class="pr-10"
            />
            <button
              type="button"
              :aria-label="showNewPassword ? $t('auth.hidePassword') : $t('auth.showPassword')"
              class="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground"
              @click="showNewPassword = !showNewPassword"
            >
              <Eye v-if="!showNewPassword" class="w-5 h-5" />
              <EyeOff v-else class="w-5 h-5" />
            </button>
          </div>
        </div>

        <div class="space-y-2">
          <Label for="confirmNewPassword">{{ $t('auth.confirmPassword') }}</Label>
          <Input
            id="confirmNewPassword"
            type="password"
            autocomplete="new-password"
            v-model="passwordForm.confirmNewPassword"
          />
        </div>

        <Button type="submit" :isLoading="isChangingPassword" :disabled="!canSubmitPasswordForm">
          {{ $t('account.changePassword') }}
        </Button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { useUserStore } from '../../../stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { AvatarUpload } from '@shared-ui/components/ui/avatar'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { Eye, EyeOff } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const emitter = useEmitter()
const { t } = useI18n()
const isSaving = ref(false)
const userStore = useUserStore()
const pendingFile = ref(null)

const isChangingPassword = ref(false)
const showCurrentPassword = ref(false)
const showNewPassword = ref(false)
const passwordForm = ref({
  currentPassword: '',
  newPassword: '',
  confirmNewPassword: ''
})

const canSubmitPasswordForm = computed(() => {
  return (
    passwordForm.value.currentPassword &&
    passwordForm.value.newPassword &&
    passwordForm.value.confirmNewPassword
  )
})

const changePassword = async () => {
  if (isChangingPassword.value || !canSubmitPasswordForm.value) return
  if (passwordForm.value.newPassword !== passwordForm.value.confirmNewPassword) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('auth.passwordsDoNotMatch')
    })
    return
  }
  try {
    isChangingPassword.value = true
    await api.changeCurrentUserPassword({
      current_password: passwordForm.value.currentPassword,
      new_password: passwordForm.value.newPassword
    })
    passwordForm.value = { currentPassword: '', newPassword: '', confirmNewPassword: '' }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('account.passwordChanged')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isChangingPassword.value = false
  }
}

const onCropped = (file) => {
  if (isSaving.value) return
  pendingFile.value = file
  userStore.setAvatar(URL.createObjectURL(file))
}

const saveUser = async () => {
  if (!pendingFile.value) return
  const formData = new FormData()
  formData.append('files', pendingFile.value, 'avatar.png')
  try {
    isSaving.value = true
    await api.updateCurrentUser(formData)
    pendingFile.value = null
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSaving.value = false
  }
}

const removeAvatar = async () => {
  if (isSaving.value) return
  try {
    await api.deleteUserAvatar()
    pendingFile.value = null
    userStore.clearAvatar()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('account.avatarRemoved')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
