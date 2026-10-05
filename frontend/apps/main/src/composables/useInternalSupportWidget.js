import api from '@main/api'
import { useAppSettingsStore } from '@main/stores/appSettings'

const WIDGET_SCRIPT_ID = 'libredesk-internal-support-widget-script'

let loadPromise = null

const loadWidgetScript = () => {
  if (window.initLibredesk) return Promise.resolve()
  if (loadPromise) return loadPromise

  loadPromise = new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.id = WIDGET_SCRIPT_ID
    script.src = '/widget.js'
    script.async = true
    script.onload = resolve
    script.onerror = () => {
      loadPromise = null
      reject(new Error('Failed to load internal support widget script'))
    }
    document.body.appendChild(script)
  })
  return loadPromise
}

// Surfaces the same chat widget used by external customers inside the helpdesk, so agents can
// reach the inbox designated as internal support (configured in admin > general settings).
export const initInternalSupportWidget = async () => {
  const settingsStore = useAppSettingsStore()
  const inboxID = Number(settingsStore.settings['app.internal_support_inbox_id'] || 0)
  if (!inboxID) return

  try {
    const [, sessionResp] = await Promise.all([loadWidgetScript(), api.getInternalSupportWidgetSession()])
    const { inbox_uuid: inboxUUID, jwt } = sessionResp.data.data
    window.initLibredesk({
      baseURL: window.location.origin,
      inboxID: inboxUUID,
      userJWT: jwt
    })
  } catch {
    // Internal support widget is a convenience, not critical to the dashboard, so fail silently.
  }
}
