package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

// V2_10_0 added a single, mutually-exclusive team_id/inbox_id scoping to tags. It was superseded
// by V2_11_0 (many-to-many tag_teams/tag_inboxes) before release, so a fresh install's schema.sql
// no longer has these columns - this only verifies the migration itself stays idempotent for
// anyone upgrading from a version before it ever ran.
func TestV2_10_0TagScopingMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_10_0")

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
