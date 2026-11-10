package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_27_0 adds the availability:set_away and availability:set_reassign permissions and grants
// them to every existing role, so agents keep the ability to use the Away and Reassign toggles
// until an admin revokes them from a role.
func V2_27_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		UPDATE roles
		SET permissions = (
			SELECT ARRAY(
				SELECT DISTINCT p FROM unnest(
					permissions || ARRAY['availability:set_away', 'availability:set_reassign']
				) AS p
			)
		)
	`)
	return err
}
