<template>
  <div class="box p-4 space-y-4">
    <h3 class="font-semibold">{{ $t('admin.inbox.uazapi.connection') }}</h3>

    <div class="flex items-center gap-2 text-sm">
      <span
        class="inline-block size-2 rounded-full"
        :class="statusDotClass"
      ></span>
      <span>{{ statusLabel }}</span>
    </div>

    <div v-if="qrCode" class="flex flex-col items-center gap-2 py-2">
      <img :src="qrCode" alt="QR code" class="w-48 h-48 border border-border rounded" />
      <p class="text-sm text-muted-foreground">{{ $t('admin.inbox.uazapi.scanQrCode') }}</p>
    </div>

    <div class="flex gap-2">
      <Button
        type="button"
        variant="outline"
        :is-loading="connecting"
        :disabled="connecting || connected"
        @click="connect"
      >
        {{ connected ? $t('admin.inbox.uazapi.connected') : $t('admin.inbox.uazapi.connect') }}
      </Button>
      <Button
        v-if="connected"
        type="button"
        variant="outline"
        :is-loading="disconnecting"
        @click="disconnect"
      >
        {{ $t('admin.inbox.uazapi.disconnect') }}
      </Button>
    </div>

    <div v-if="webhookURL" class="space-y-2 pt-2 border-t border-border">
      <div class="flex items-center gap-2">
        <Input :model-value="webhookURL" readonly class="font-mono text-xs" />
        <CopyButton :text="webhookURL" />
        <Button
          type="button"
          variant="outline"
          size="sm"
          :is-loading="registeringWebhook"
          @click="registerWebhook"
        >
          {{ $t('admin.inbox.uazapi.registerWebhook') }}
        </Button>
      </div>
      <p v-if="webhookRegistered" class="text-sm text-green-600 flex items-start gap-1.5">
        <Check class="size-4 mt-0.5 shrink-0" />
        <span>{{ $t('admin.inbox.uazapi.webhookRegistered') }}</span>
      </p>
      <p v-else-if="webhookError" class="text-sm text-destructive flex items-start gap-1.5">
        <TriangleAlert class="size-4 mt-0.5 shrink-0" />
        <span>{{ webhookError }}</span>
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import CopyButton from '@/components/button/CopyButton.vue'
import { Check, TriangleAlert } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const POLL_INTERVAL_MS = 2000

const props = defineProps({
  inboxId: {
    type: [String, Number],
    required: true
  }
})

const { t } = useI18n()
const emitter = useEmitter()

const status = ref('disconnected') // disconnected | connecting | connected | hibernated
const qrCode = ref('')
const connecting = ref(false)
const disconnecting = ref(false)
const registeringWebhook = ref(false)
const webhookURL = ref('')
const webhookRegistered = ref(false)
const webhookError = ref('')
let pollTimer = null

const connected = computed(() => status.value === 'connected')

const statusLabel = computed(() => {
  switch (status.value) {
    case 'connected':
      return t('admin.inbox.uazapi.status.connected')
    case 'connecting':
      return t('admin.inbox.uazapi.status.connecting')
    case 'hibernated':
      return t('admin.inbox.uazapi.status.hibernated')
    default:
      return t('admin.inbox.uazapi.status.disconnected')
  }
})

const statusDotClass = computed(() => {
  switch (status.value) {
    case 'connected':
      return 'bg-green-500'
    case 'connecting':
      return 'bg-yellow-500'
    default:
      return 'bg-destructive'
  }
})

const applyStatus = (data) => {
  const instance = data?.instance || {}
  const connState = data?.status || {}
  if (connState.connected) {
    status.value = 'connected'
    qrCode.value = ''
  } else {
    status.value = instance.status || 'disconnected'
    const code = instance.qrcode || ''
    qrCode.value = code ? (code.startsWith('data:') ? code : `data:image/png;base64,${code}`) : ''
  }
  if (data?.webhook_url) webhookURL.value = data.webhook_url
  if ('webhook_registered' in (data || {})) {
    webhookRegistered.value = !!data.webhook_registered
    webhookError.value = data.webhook_error || ''
  }
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const startPolling = () => {
  stopPolling()
  pollTimer = setInterval(async () => {
    try {
      const resp = await api.getUazapiInboxStatus(props.inboxId)
      applyStatus(resp.data.data)
      if (resp.data.data?.status?.connected) stopPolling()
    } catch {
      // Transient polling errors are ignored; the next tick retries.
    }
  }, POLL_INTERVAL_MS)
}

const fetchStatus = async () => {
  try {
    const resp = await api.getUazapiInboxStatus(props.inboxId)
    applyStatus(resp.data.data)
    if (status.value === 'connecting') startPolling()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const connect = async () => {
  connecting.value = true
  try {
    const resp = await api.connectUazapiInbox(props.inboxId)
    applyStatus(resp.data.data)
    startPolling()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    connecting.value = false
  }
}

const disconnect = async () => {
  disconnecting.value = true
  try {
    await api.disconnectUazapiInbox(props.inboxId)
    status.value = 'disconnected'
    qrCode.value = ''
    stopPolling()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    disconnecting.value = false
  }
}

const registerWebhook = async () => {
  registeringWebhook.value = true
  try {
    const resp = await api.registerUazapiWebhook(props.inboxId)
    webhookURL.value = resp.data.data?.webhook_url || webhookURL.value
    webhookRegistered.value = !!resp.data.data?.webhook_registered
    webhookError.value = resp.data.data?.webhook_error || ''
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    registeringWebhook.value = false
  }
}

onMounted(fetchStatus)
onUnmounted(stopPolling)
</script>
