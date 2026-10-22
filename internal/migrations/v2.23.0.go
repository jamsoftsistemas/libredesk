package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_23_0 adds the "prompt_tags_on_resolve" inbox setting, which warns agents before they
// resolve a conversation that has no tags, mirroring the existing prompt-on-reply setting.
func V2_23_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS prompt_tags_on_resolve bool DEFAULT false NOT NULL;`); err != nil {
		return err
	}
	return nil
}
