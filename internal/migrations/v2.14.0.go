package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_14_0 adds the companies table (customer portfolio) and its permissions.
func V2_14_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS companies (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			external_id TEXT NULL,
			"name" TEXT NOT NULL,
			"document" TEXT NULL,
			city TEXT NULL,
			CONSTRAINT constraint_companies_on_name CHECK (length("name") <= 200),
			CONSTRAINT constraint_companies_on_external_id CHECK (length(external_id) <= 100),
			CONSTRAINT constraint_companies_on_document CHECK (length("document") <= 20),
			CONSTRAINT constraint_companies_on_city CHECK (length(city) <= 140)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_cat(permissions, ARRAY['companies:read_all', 'companies:write', 'companies:delete'])
		WHERE name = 'Admin'
		AND NOT ('companies:read_all' = ANY(permissions));
	`); err != nil {
		return err
	}

	return nil
}
