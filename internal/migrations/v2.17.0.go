package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_17_0 enforces document (CNPJ/CPF) uniqueness across companies, ignoring blank values.
func V2_17_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_companies_on_document
			ON companies("document") WHERE "document" IS NOT NULL AND "document" != '';
	`); err != nil {
		return err
	}
	return nil
}
