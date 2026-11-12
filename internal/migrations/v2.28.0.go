package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_28_0 adds composite indexes matching the conversations list filters and its default
// ORDER BY last_message_at DESC NULLS LAST, so the list can walk an index in order and stop at LIMIT.
func V2_28_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	stmts := []string{
		`CREATE INDEX IF NOT EXISTS index_conversations_on_assigned_user_id_last_message_at ON conversations (assigned_user_id, last_message_at DESC NULLS LAST)`,
		`CREATE INDEX IF NOT EXISTS index_conversations_on_assigned_team_id_last_message_at ON conversations (assigned_team_id, last_message_at DESC NULLS LAST)`,
		`CREATE INDEX IF NOT EXISTS index_conversations_on_status_id_last_message_at ON conversations (status_id, last_message_at DESC NULLS LAST)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
