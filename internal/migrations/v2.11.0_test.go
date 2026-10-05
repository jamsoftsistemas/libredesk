package migrations

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_11_0TagMultiScopingMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_11_0")

	// Simulate an install that already ran v2.10.0's single team_id/inbox_id scoping.
	if err := V2_10_0(db, nil, nil); err != nil {
		t.Fatalf("simulating v2.10.0: %v", err)
	}

	var teamID, inboxID int
	db.MustExec(`INSERT INTO teams (name, conversation_assignment_type) VALUES ('Support', 'Manual')`)
	if err := db.Get(&teamID, `SELECT id FROM teams WHERE name = 'Support'`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO inboxes (name, channel) VALUES ('Help desk', 'email')`)
	if err := db.Get(&inboxID, `SELECT id FROM inboxes WHERE name = 'Help desk'`); err != nil {
		t.Fatal(err)
	}

	var teamTagID, inboxTagID, globalTagID int
	db.MustExec(`INSERT INTO tags (name, visibility, team_id) VALUES ('team-tag', 'team', $1)`, teamID)
	if err := db.Get(&teamTagID, `SELECT id FROM tags WHERE name = 'team-tag'`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO tags (name, visibility, inbox_id) VALUES ('inbox-tag', 'inbox', $1)`, inboxID)
	if err := db.Get(&inboxTagID, `SELECT id FROM tags WHERE name = 'inbox-tag'`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO tags (name, visibility) VALUES ('global-tag', 'all')`)
	if err := db.Get(&globalTagID, `SELECT id FROM tags WHERE name = 'global-tag'`); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := V2_11_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	var teamIDs []int
	if err := db.Select(&teamIDs, `SELECT team_id FROM tag_teams WHERE tag_id = $1`, teamTagID); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(teamIDs, teamID) {
		t.Fatalf("tag_teams after migration = %v, want to contain team %d", teamIDs, teamID)
	}

	var inboxIDs []int
	if err := db.Select(&inboxIDs, `SELECT inbox_id FROM tag_inboxes WHERE tag_id = $1`, inboxTagID); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(inboxIDs, inboxID) {
		t.Fatalf("tag_inboxes after migration = %v, want to contain inbox %d", inboxIDs, inboxID)
	}

	var globalScopeCount int
	if err := db.Get(&globalScopeCount, `
		SELECT (
			(SELECT COUNT(*) FROM tag_teams WHERE tag_id = $1) +
			(SELECT COUNT(*) FROM tag_inboxes WHERE tag_id = $1)
		)
	`, globalTagID); err != nil {
		t.Fatal(err)
	}
	if globalScopeCount != 0 {
		t.Fatalf("global tag got %d scoping rows after migration, want 0", globalScopeCount)
	}

	var columnsLeft int
	if err := db.Get(&columnsLeft, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'tags' AND column_name IN ('visibility', 'team_id', 'inbox_id')
	`); err != nil {
		t.Fatal(err)
	}
	if columnsLeft != 0 {
		t.Fatalf("expected visibility/team_id/inbox_id columns to be dropped, %d remain", columnsLeft)
	}
}
