package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'tag_visibility') THEN
				CREATE TYPE tag_visibility AS ENUM ('all', 'team', 'inbox');
			END IF;
		END$$;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE tags ADD COLUMN IF NOT EXISTS visibility tag_visibility NOT NULL DEFAULT 'all';`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE tags ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE NULL;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE tags ADD COLUMN IF NOT EXISTS inbox_id INT REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE NULL;`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'constraint_tags_visibility_team') THEN
				ALTER TABLE tags ADD CONSTRAINT constraint_tags_visibility_team CHECK (visibility != 'team' OR team_id IS NOT NULL);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'constraint_tags_visibility_inbox') THEN
				ALTER TABLE tags ADD CONSTRAINT constraint_tags_visibility_inbox CHECK (visibility != 'inbox' OR inbox_id IS NOT NULL);
			END IF;
		END$$;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_tags_on_team_id ON tags(team_id);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_tags_on_inbox_id ON tags(inbox_id);`); err != nil {
		return err
	}

	return nil
}
