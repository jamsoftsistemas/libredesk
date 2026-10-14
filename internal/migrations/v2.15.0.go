package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_15_0 adds an optional company association on users, so a contact can be linked to a client company.
func V2_15_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS company_id INT NULL REFERENCES companies(id) ON DELETE SET NULL ON UPDATE CASCADE;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_users_on_company_id ON users(company_id) WHERE company_id IS NOT NULL;
	`); err != nil {
		return err
	}
	return nil
}
