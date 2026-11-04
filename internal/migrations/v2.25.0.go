package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_25_0 splits the "conversations:update_status" permission into one permission per status
// (open, snoozed, resolved, closed). Roles that had the old permission keep full access by
// receiving all four, and the old permission is removed.
func V2_25_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_remove(permissions, 'conversations:update_status') || ARRAY[
			'conversations:update_status_open',
			'conversations:update_status_snoozed',
			'conversations:update_status_resolved',
			'conversations:update_status_closed'
		]
		WHERE 'conversations:update_status' = ANY(permissions);
	`); err != nil {
		return err
	}
	return nil
}
