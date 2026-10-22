package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_22_0 adds the "new_conversation" notification type, sent to agents authorized to read a
// team's or the unassigned queue when a new conversation arrives there.
func V2_22_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`ALTER TYPE user_notification_type ADD VALUE IF NOT EXISTS 'new_conversation';`); err != nil {
		return err
	}
	return nil
}
