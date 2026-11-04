import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { useConversationStore } from '@/stores/conversation'
import { permissions as p, STATUS_UPDATE_PERMISSIONS } from '@/constants/permissions'

export function useBulkActionPermissions () {
  const userStore = useUserStore()
  const conversationStore = useConversationStore()

  const canAssignAgent = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_USER_ASSIGNEE))
  const canAssignTeam = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE))
  const canUpdateStatus = computed(() => STATUS_UPDATE_PERMISSIONS.some((perm) => userStore.can(perm)))
  const allowedStatusOptions = computed(() =>
    conversationStore.statusOptionsNoSnooze.filter((status) => userStore.can(status.permission))
  )
  const canUpdateTags = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_TAGS))

  const canBulkAct = computed(
    () => canAssignAgent.value || canAssignTeam.value || canUpdateStatus.value || canUpdateTags.value
  )

  return { canAssignAgent, canAssignTeam, canUpdateStatus, allowedStatusOptions, canUpdateTags, canBulkAct }
}
