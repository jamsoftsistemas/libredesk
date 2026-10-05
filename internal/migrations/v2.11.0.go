package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_11_0 replaces the single, mutually-exclusive tags.team_id/tags.inbox_id scoping (added in
// v2.10.0) with many-to-many tag_teams/tag_inboxes join tables, so a tag can be scoped to
// multiple teams and/or multiple inboxes at once.
func V2_11_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tag_teams (
			tag_id INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE ON UPDATE CASCADE,
			team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
			PRIMARY KEY (tag_id, team_id)
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_tag_teams_on_team_id ON tag_teams(team_id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tag_inboxes (
			tag_id INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE ON UPDATE CASCADE,
			inbox_id INT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE,
			PRIMARY KEY (tag_id, inbox_id)
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_tag_inboxes_on_inbox_id ON tag_inboxes(inbox_id);`); err != nil {
		return err
	}

	// Backfill from the v2.10.0 single team_id/inbox_id columns, if they still exist.
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tags' AND column_name = 'team_id') THEN
				INSERT INTO tag_teams (tag_id, team_id)
				SELECT id, team_id FROM tags WHERE team_id IS NOT NULL
				ON CONFLICT DO NOTHING;
			END IF;
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tags' AND column_name = 'inbox_id') THEN
				INSERT INTO tag_inboxes (tag_id, inbox_id)
				SELECT id, inbox_id FROM tags WHERE inbox_id IS NOT NULL
				ON CONFLICT DO NOTHING;
			END IF;
		END$$;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE tags DROP CONSTRAINT IF EXISTS constraint_tags_visibility_team;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE tags DROP CONSTRAINT IF EXISTS constraint_tags_visibility_inbox;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE tags DROP COLUMN IF EXISTS visibility;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE tags DROP COLUMN IF EXISTS team_id;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE tags DROP COLUMN IF EXISTS inbox_id;`); err != nil {
		return err
	}
	if _, err := db.Exec(`DROP TYPE IF EXISTS tag_visibility;`); err != nil {
		return err
	}

	return nil
}
