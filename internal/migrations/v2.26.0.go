package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_26_0 makes search accent-insensitive and word-order independent. It installs the unaccent
// extension with an IMMUTABLE f_unaccent() wrapper (required for use in indexes) and builds
// trigram indexes over f_unaccent() of the searched columns. The old plain trigram indexes on
// conversations.subject and conversation_messages.text_content are only used by search, so they
// are replaced instead of kept alongside the new ones.
func V2_26_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE EXTENSION IF NOT EXISTS unaccent;
		CREATE OR REPLACE FUNCTION f_unaccent(text) RETURNS text AS $$
			SELECT public.unaccent('public.unaccent', $1)
		$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;
	`); err != nil {
		return err
	}

	indexes := []struct{ name, table, column string }{
		{"index_tgrm_unaccent_conversations_on_subject", "conversations", "subject"},
		{"index_tgrm_unaccent_conversation_messages_on_text_content", "conversation_messages", "text_content"},
		{"index_tgrm_unaccent_users_on_email", "users", "email"},
		{"index_tgrm_unaccent_users_on_phone_number", "users", "phone_number"},
		{"index_tgrm_unaccent_users_on_first_name", "users", "first_name"},
		{"index_tgrm_unaccent_users_on_last_name", "users", "last_name"},
	}
	for _, i := range indexes {
		ddl := `CREATE INDEX CONCURRENTLY IF NOT EXISTS ` + i.name + ` ON ` + i.table + ` USING GIN (f_unaccent(` + i.column + `) gin_trgm_ops)`
		if err := createIndexConcurrently(db, i.name, ddl); err != nil {
			return err
		}
	}

	for _, old := range []string{"index_tgrm_conversations_on_subject", "index_trgm_conversation_messages_on_text_content"} {
		if _, err := db.Exec(`DROP INDEX IF EXISTS ` + old); err != nil {
			return err
		}
	}
	return nil
}
