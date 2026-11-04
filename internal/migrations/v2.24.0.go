package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_24_0 adds the "conversations:assign_self_from_other" permission to the Admin role. It lets an agent
// assign a conversation held by another agent to themselves without the full user-assign permission.
func V2_24_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'conversations:assign_self_from_other')
		WHERE name = 'Admin' AND NOT ('conversations:assign_self_from_other' = ANY(permissions));
	`); err != nil {
		return err
	}
	return nil
}
