package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_13_0 adds the "uazapi" channel, an unofficial WhatsApp gateway connected via QR code.
func V2_13_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`ALTER TYPE channels ADD VALUE IF NOT EXISTS 'uazapi';`); err != nil {
		return err
	}
	return nil
}
