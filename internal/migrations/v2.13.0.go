package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_13_0 adds trigram indexes needed for conversation search to match on
// conversation subject and contact name, not just contact email and reference number.
func V2_13_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if err := createIndexConcurrently(db, "index_tgrm_conversations_on_subject", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_tgrm_conversations_on_subject ON conversations USING GIN (subject gin_trgm_ops)`); err != nil {
		return err
	}
	if err := createIndexConcurrently(db, "index_tgrm_users_on_first_name", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_tgrm_users_on_first_name ON users USING GIN (first_name gin_trgm_ops)`); err != nil {
		return err
	}
	if err := createIndexConcurrently(db, "index_tgrm_users_on_last_name", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_tgrm_users_on_last_name ON users USING GIN (last_name gin_trgm_ops)`); err != nil {
		return err
	}
	return nil
}
