package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_16_0 adds a free-text "system" field to companies (customer portfolio).
func V2_16_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		ALTER TABLE companies ADD COLUMN IF NOT EXISTS "system" TEXT NULL;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		ALTER TABLE companies DROP CONSTRAINT IF EXISTS constraint_companies_on_system;
		ALTER TABLE companies ADD CONSTRAINT constraint_companies_on_system CHECK (length("system") <= 140);
	`); err != nil {
		return err
	}
	return nil
}
