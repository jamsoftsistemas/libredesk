package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_12_0 adds the setting that designates a live chat inbox as the internal support inbox,
// used to surface the chat widget to agents inside the helpdesk itself.
func V2_12_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO settings ("key", value) VALUES
			('app.internal_support_inbox_id', '0'::jsonb)
		ON CONFLICT ("key") DO NOTHING;
	`); err != nil {
		return err
	}
	return nil
}
