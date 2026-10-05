package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_10_0TagScopingMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_10_0")

	db.MustExec(`ALTER TABLE tags DROP CONSTRAINT constraint_tags_visibility_team`)
	db.MustExec(`ALTER TABLE tags DROP CONSTRAINT constraint_tags_visibility_inbox`)
	db.MustExec(`ALTER TABLE tags DROP COLUMN visibility, DROP COLUMN team_id, DROP COLUMN inbox_id`)

	for range 2 {
		if err := V2_10_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	var teamID int
	db.MustExec(`INSERT INTO teams (name, conversation_assignment_type) VALUES ('Support', 'Manual')`)
	if err := db.Get(&teamID, `SELECT id FROM teams WHERE name = 'Support'`); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`INSERT INTO tags (name, visibility) VALUES ('team-tag', 'team')`); err == nil {
		t.Fatal("expected visibility='team' without team_id to violate the check constraint")
	}

	if _, err := db.Exec(`INSERT INTO tags (name, visibility, team_id) VALUES ('team-tag', 'team', $1)`, teamID); err != nil {
		t.Fatalf("inserting valid team-scoped tag: %v", err)
	}
}
