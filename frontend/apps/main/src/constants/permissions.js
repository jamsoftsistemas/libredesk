export const permissions = {
  CONVERSATIONS_READ: 'conversations:read',
  CONVERSATIONS_WRITE: 'conversations:write',
  CONVERSATIONS_READ_ASSIGNED: 'conversations:read_assigned',
  CONVERSATIONS_READ_ALL: 'conversations:read_all',
  CONVERSATIONS_READ_UNASSIGNED: 'conversations:read_unassigned',
  CONVERSATIONS_READ_TEAM_INBOX: 'conversations:read_team_inbox',
  CONVERSATIONS_READ_TEAM_ALL: 'conversations:read_team_all',
  CONVERSATIONS_UPDATE_USER_ASSIGNEE: 'conversations:update_user_assignee',
  CONVERSATIONS_UPDATE_TEAM_ASSIGNEE: 'conversations:update_team_assignee',
  CONVERSATIONS_ASSIGN_SELF_FROM_OTHER: 'conversations:assign_self_from_other',
  CONVERSATIONS_UPDATE_PRIORITY: 'conversations:update_priority',
  CONVERSATIONS_UPDATE_STATUS_OPEN: 'conversations:update_status_open',
  CONVERSATIONS_UPDATE_STATUS_SNOOZED: 'conversations:update_status_snoozed',
  CONVERSATIONS_UPDATE_STATUS_RESOLVED: 'conversations:update_status_resolved',
  CONVERSATIONS_UPDATE_STATUS_CLOSED: 'conversations:update_status_closed',
  CONVERSATIONS_UPDATE_TAGS: 'conversations:update_tags',
  MESSAGES_READ: 'messages:read',
  MESSAGES_WRITE: 'messages:write',
  MESSAGES_WRITE_PRIVATE: 'messages:write_private',
  MESSAGES_WRITE_AS_CONTACT: 'messages:write_as_contact',
  VIEW_MANAGE: 'view:manage',
  SHARED_VIEWS_MANAGE: 'shared_views:manage',
  GENERAL_SETTINGS_MANAGE: 'general_settings:manage',
  NOTIFICATION_SETTINGS_MANAGE: 'notification_settings:manage',
  STATUS_MANAGE: 'status:manage',
  OIDC_MANAGE: 'oidc:manage',
  TAGS_MANAGE: 'tags:manage',
  MACROS_MANAGE: 'macros:manage',
  USERS_MANAGE: 'users:manage',
  TEAMS_MANAGE: 'teams:manage',
  AUTOMATIONS_MANAGE: 'automations:manage',
  INBOXES_MANAGE: 'inboxes:manage',
  ROLES_MANAGE: 'roles:manage',
  TEMPLATES_MANAGE: 'templates:manage',
  REPORTS_MANAGE: 'reports:manage',
  REPORTS_VIEW_OWN: 'reports:view_own',
  BUSINESS_HOURS_MANAGE: 'business_hours:manage',
  SLA_MANAGE: 'sla:manage',
  AI_MANAGE: 'ai:manage',
  HELP_CENTER_MANAGE: 'help_center:manage',
  CUSTOM_ATTRIBUTES_MANAGE: 'custom_attributes:manage',
  CONTACTS_READ_ALL: 'contacts:read_all',
  CONTACTS_READ: 'contacts:read',
  CONTACTS_WRITE: 'contacts:write',
  CONTACTS_BLOCK: 'contacts:block',
  CONTACTS_DELETE: 'contacts:delete',
  CONTACTS_EXPORT: 'contacts:export',
  CONTACT_NOTES_READ: 'contact_notes:read',
  CONTACT_NOTES_WRITE: 'contact_notes:write',
  CONTACT_NOTES_DELETE: 'contact_notes:delete',
  COMPANIES_READ_ALL: 'companies:read_all',
  COMPANIES_WRITE: 'companies:write',
  COMPANIES_DELETE: 'companies:delete',
  ACTIVITY_LOGS_MANAGE: 'activity_logs:manage',
  WEBHOOKS_MANAGE: 'webhooks:manage',
  CONTEXT_LINKS_MANAGE: 'context_links:manage'
}

// Every permission that allows moving a conversation to some status.
export const STATUS_UPDATE_PERMISSIONS = [
  permissions.CONVERSATIONS_UPDATE_STATUS_OPEN,
  permissions.CONVERSATIONS_UPDATE_STATUS_SNOOZED,
  permissions.CONVERSATIONS_UPDATE_STATUS_RESOLVED,
  permissions.CONVERSATIONS_UPDATE_STATUS_CLOSED
]

// Permission needed to move a conversation to a status. Custom statuses follow their category.
export const statusPermission = (name, category) => {
  switch (name) {
    case 'Open':
      return permissions.CONVERSATIONS_UPDATE_STATUS_OPEN
    case 'Snoozed':
      return permissions.CONVERSATIONS_UPDATE_STATUS_SNOOZED
    case 'Resolved':
      return permissions.CONVERSATIONS_UPDATE_STATUS_RESOLVED
    case 'Closed':
      return permissions.CONVERSATIONS_UPDATE_STATUS_CLOSED
  }
  if (category === 'waiting') return permissions.CONVERSATIONS_UPDATE_STATUS_SNOOZED
  if (category === 'resolved') return permissions.CONVERSATIONS_UPDATE_STATUS_RESOLVED
  return permissions.CONVERSATIONS_UPDATE_STATUS_OPEN
}
