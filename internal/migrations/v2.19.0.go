package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_19_0 applies the upstream v2.9.0-rc.9 and v2.9.0-rc.10 migrations on databases
// that were already past them in the fork's version numbering. Both are idempotent.
func V2_19_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if err := V2_9_0_RC9(db, fs, ko); err != nil {
		return err
	}
	return V2_9_0_RC10(db, fs, ko)
}
