import { useRouter } from 'vue-router'

// Switching inbox lists always deselects any open conversation - on mobile that's the
// only sane option (the conversation covers the whole screen), and on desktop staying
// on a now-irrelevant conversation while the list underneath changes is just as confusing.
export function useInboxNavigation() {
  const router = useRouter()

  const navigate = (listName, params) => router.push({ name: listName, params })

  const navigateToInbox = (type) => navigate('inbox', { type })
  const navigateToTeamInbox = (teamID) => navigate('team-inbox', { teamID })
  const navigateToViewInbox = (viewID) => navigate('view-inbox', { viewID })

  return { navigateToInbox, navigateToTeamInbox, navigateToViewInbox }
}
