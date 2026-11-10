package models

const (
	// Conversation
	PermConversationsReadAll            = "conversations:read_all"
	PermConversationsReadUnassigned     = "conversations:read_unassigned"
	PermConversationsReadAssigned       = "conversations:read_assigned"
	PermConversationsReadTeamInbox      = "conversations:read_team_inbox"
	PermConversationsReadTeamAll        = "conversations:read_team_all"
	PermConversationsRead               = "conversations:read"
	PermConversationsUpdateUserAssignee = "conversations:update_user_assignee"
	PermConversationsUpdateTeamAssignee = "conversations:update_team_assignee"
	PermConversationsAssignSelfFromOther = "conversations:assign_self_from_other"
	PermConversationsUpdatePriority     = "conversations:update_priority"
	PermConversationsUpdateStatusOpen     = "conversations:update_status_open"
	PermConversationsUpdateStatusSnoozed  = "conversations:update_status_snoozed"
	PermConversationsUpdateStatusResolved = "conversations:update_status_resolved"
	PermConversationsUpdateStatusClosed   = "conversations:update_status_closed"
	PermConversationsUpdateTags         = "conversations:update_tags"
	PermConversationWrite               = "conversations:write"
	PermMessagesRead                    = "messages:read"
	PermMessagesWrite                   = "messages:write"
	PermMessagesWritePrivate            = "messages:write_private"
	PermMessagesWriteAsContact          = "messages:write_as_contact"

	// View
	PermViewManage        = "view:manage"
	PermSharedViewsManage = "shared_views:manage"

	// Status
	PermStatusManage = "status:manage"

	// Tags
	PermTagsManage = "tags:manage"

	// Macros
	PermMacrosManage = "macros:manage"

	// Users
	PermUsersManage = "users:manage"

	// Teams
	PermTeamsManage = "teams:manage"

	// Automations
	PermAutomationsManage = "automations:manage"

	// Inboxes
	PermInboxesManage = "inboxes:manage"

	// Roles
	PermRolesManage = "roles:manage"

	// Webhooks
	PermWebhooksManage = "webhooks:manage"

	// Context Links
	PermContextLinksManage = "context_links:manage"

	// Templates
	PermTemplatesManage = "templates:manage"

	// Reports
	PermReportsManage = "reports:manage"
	PermReportsViewOwn = "reports:view_own"

	// Business Hours
	PermBusinessHoursManage = "business_hours:manage"

	// SLA
	PermSLAManage = "sla:manage"

	// General Settings
	PermGeneralSettingsManage = "general_settings:manage"

	// Notification Settings
	PermNotificationSettingsManage = "notification_settings:manage"

	// OpenID Connect SSO
	PermOIDCManage = "oidc:manage"

	// AI
	PermAIManage = "ai:manage"

	// Help center
	PermHelpCenterManage = "help_center:manage"

	// Contacts
	PermContactsReadAll = "contacts:read_all"
	PermContactsRead    = "contacts:read"
	PermContactsWrite   = "contacts:write"
	PermContactsBlock   = "contacts:block"
	PermContactsDelete  = "contacts:delete"
	PermContactsExport  = "contacts:export"

	// Contact Notes
	PermContactNotesRead   = "contact_notes:read"
	PermContactNotesWrite  = "contact_notes:write"
	PermContactNotesDelete = "contact_notes:delete"

	// Companies
	PermCompaniesReadAll = "companies:read_all"
	PermCompaniesWrite   = "companies:write"
	PermCompaniesDelete  = "companies:delete"

	// Custom attributes
	PermCustomAttributesManage = "custom_attributes:manage"

	// Activity log
	PermActivityLogsManage = "activity_logs:manage"
)

var validPermissions = map[string]struct{}{
	PermConversationsReadAll:            {},
	PermConversationsReadUnassigned:     {},
	PermConversationsReadAssigned:       {},
	PermConversationsReadTeamInbox:      {},
	PermConversationsReadTeamAll:        {},
	PermConversationsRead:               {},
	PermConversationsUpdateUserAssignee: {},
	PermConversationsUpdateTeamAssignee: {},
	PermConversationsAssignSelfFromOther: {},
	PermConversationsUpdatePriority:     {},
	PermConversationsUpdateStatusOpen:     {},
	PermConversationsUpdateStatusSnoozed:  {},
	PermConversationsUpdateStatusResolved: {},
	PermConversationsUpdateStatusClosed:   {},
	PermConversationsUpdateTags:         {},
	PermConversationWrite:               {},
	PermMessagesRead:                    {},
	PermMessagesWrite:                   {},
	PermMessagesWritePrivate:            {},
	PermMessagesWriteAsContact:          {},
	PermViewManage:                      {},
	PermSharedViewsManage:               {},
	PermStatusManage:                    {},
	PermTagsManage:                      {},
	PermMacrosManage:                    {},
	PermUsersManage:                     {},
	PermTeamsManage:                     {},
	PermAutomationsManage:               {},
	PermInboxesManage:                   {},
	PermRolesManage:                     {},
	PermTemplatesManage:                 {},
	PermReportsManage:                   {},
	PermReportsViewOwn:                  {},
	PermBusinessHoursManage:             {},
	PermSLAManage:                       {},
	PermGeneralSettingsManage:           {},
	PermNotificationSettingsManage:      {},
	PermOIDCManage:                      {},
	PermAIManage:                        {},
	PermHelpCenterManage:                {},
	PermCustomAttributesManage:          {},
	PermContactsReadAll:                 {},
	PermContactsRead:                    {},
	PermContactsWrite:                   {},
	PermContactsBlock:                   {},
	PermContactsDelete:                  {},
	PermContactsExport:                  {},
	PermContactNotesRead:                {},
	PermContactNotesWrite:               {},
	PermContactNotesDelete:              {},
	PermCompaniesReadAll:                {},
	PermCompaniesWrite:                  {},
	PermCompaniesDelete:                 {},
	PermActivityLogsManage:              {},
	PermWebhooksManage:                  {},
	PermContextLinksManage:              {},
}

// PermissionExists returns true if the permission exists else false
func PermissionExists(permission string) bool {
	_, exists := validPermissions[permission]
	return exists
}

// StatusUpdatePermissions lists every permission that allows changing a conversation to some status.
var StatusUpdatePermissions = []string{
	PermConversationsUpdateStatusOpen,
	PermConversationsUpdateStatusSnoozed,
	PermConversationsUpdateStatusResolved,
	PermConversationsUpdateStatusClosed,
}

// StatusUpdatePermission returns the permission required to move a conversation to the given status.
// Custom statuses fall back to their category: open, waiting (snoozed-like) or resolved.
func StatusUpdatePermission(name, category string) string {
	switch name {
	case "Open":
		return PermConversationsUpdateStatusOpen
	case "Snoozed":
		return PermConversationsUpdateStatusSnoozed
	case "Resolved":
		return PermConversationsUpdateStatusResolved
	case "Closed":
		return PermConversationsUpdateStatusClosed
	}
	switch category {
	case "waiting":
		return PermConversationsUpdateStatusSnoozed
	case "resolved":
		return PermConversationsUpdateStatusResolved
	}
	return PermConversationsUpdateStatusOpen
}
