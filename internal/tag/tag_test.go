package tag

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	mgr, _ := newTestManagerWithDB(t)
	return mgr
}

func newTestManagerWithDB(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "tag")
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatalf("creating tag manager: %v", err)
	}
	return mgr, db
}

func TestGetAllPagination(t *testing.T) {
	mgr := newTestManager(t)
	for _, name := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
		if _, err := mgr.Create(name, nil, nil); err != nil {
			t.Fatalf("creating tag %q: %v", name, err)
		}
	}

	all, err := mgr.GetAll("", 0, 0)
	if err != nil {
		t.Fatalf("GetAll without paging: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("got %d tags, want all 5", len(all))
	}

	pageTwo, err := mgr.GetAll("", 2, 2)
	if err != nil {
		t.Fatalf("GetAll page 2: %v", err)
	}
	if len(pageTwo) != 2 || pageTwo[0].Name != "charlie" || pageTwo[1].Name != "delta" {
		t.Fatalf("page 2 of size 2 = %+v, want charlie and delta", pageTwo)
	}

	lastPage, err := mgr.GetAll("", 3, 2)
	if err != nil {
		t.Fatalf("GetAll page 3: %v", err)
	}
	if len(lastPage) != 1 || lastPage[0].Name != "echo" {
		t.Fatalf("page 3 of size 2 = %+v, want just echo", lastPage)
	}
}

func TestGetAllSearch(t *testing.T) {
	mgr := newTestManager(t)
	for _, name := range []string{"billing", "refund", "billing-dispute"} {
		if _, err := mgr.Create(name, nil, nil); err != nil {
			t.Fatalf("creating tag %q: %v", name, err)
		}
	}

	matches, err := mgr.GetAll("bill", 1, 30)
	if err != nil {
		t.Fatalf("GetAll with query: %v", err)
	}
	if len(matches) != 2 || matches[0].Name != "billing" || matches[1].Name != "billing-dispute" {
		t.Fatalf("query \"bill\" = %+v, want billing and billing-dispute", matches)
	}

	none, err := mgr.GetAll("nothing-matches-this", 1, 30)
	if err != nil {
		t.Fatalf("GetAll with unmatched query: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("unmatched query returned %d tags, want none", len(none))
	}
}

func TestGetByIDs(t *testing.T) {
	mgr := newTestManager(t)
	var ids []int
	for _, name := range []string{"alpha", "bravo", "charlie"} {
		tag, err := mgr.Create(name, nil, nil)
		if err != nil {
			t.Fatalf("creating tag %q: %v", name, err)
		}
		ids = append(ids, tag.ID)
	}

	got, err := mgr.GetByIDs([]int{ids[2], ids[0]})
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	if len(got) != 2 || got[0].Name != "alpha" || got[1].Name != "charlie" {
		t.Fatalf("GetByIDs = %+v, want alpha and charlie", got)
	}

	empty, err := mgr.GetByIDs([]int{})
	if err != nil {
		t.Fatalf("GetByIDs with no ids: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("GetByIDs with no ids returned %d tags, want none", len(empty))
	}
}

func createTeam(t *testing.T, db *sqlx.DB, name string) int {
	t.Helper()
	db.MustExec(`INSERT INTO teams (name, conversation_assignment_type) VALUES ($1, 'Manual')`, name)
	var id int
	if err := db.Get(&id, `SELECT id FROM teams WHERE name = $1`, name); err != nil {
		t.Fatalf("fetching team id for %q: %v", name, err)
	}
	return id
}

func createInbox(t *testing.T, db *sqlx.DB, name string) int {
	t.Helper()
	db.MustExec(`INSERT INTO inboxes (name, channel) VALUES ($1, 'email')`, name)
	var id int
	if err := db.Get(&id, `SELECT id FROM inboxes WHERE name = $1`, name); err != nil {
		t.Fatalf("fetching inbox id for %q: %v", name, err)
	}
	return id
}

func TestGetScoped(t *testing.T) {
	mgr, db := newTestManagerWithDB(t)

	teamID := createTeam(t, db, "Support")
	otherTeamID := createTeam(t, db, "Sales")
	inboxID := createInbox(t, db, "Help desk")
	otherInboxID := createInbox(t, db, "Billing desk")

	if _, err := mgr.Create("global", nil, nil); err != nil {
		t.Fatalf("creating global tag: %v", err)
	}
	if _, err := mgr.Create("support-team", []int{teamID}, nil); err != nil {
		t.Fatalf("creating team tag: %v", err)
	}
	if _, err := mgr.Create("sales-team", []int{otherTeamID}, nil); err != nil {
		t.Fatalf("creating other team tag: %v", err)
	}
	if _, err := mgr.Create("help-desk", nil, []int{inboxID}); err != nil {
		t.Fatalf("creating inbox tag: %v", err)
	}
	if _, err := mgr.Create("billing-desk", nil, []int{otherInboxID}); err != nil {
		t.Fatalf("creating other inbox tag: %v", err)
	}
	// Scoped to both a team the caller isn't in and an inbox the caller is in - should still match.
	if _, err := mgr.Create("multi-scoped", []int{otherTeamID}, []int{inboxID}); err != nil {
		t.Fatalf("creating multi-scoped tag: %v", err)
	}

	got, err := mgr.GetScoped("", &teamID, &inboxID)
	if err != nil {
		t.Fatalf("GetScoped: %v", err)
	}
	var names []string
	for _, tag := range got {
		names = append(names, tag.Name)
	}
	want := []string{"global", "help-desk", "multi-scoped", "support-team"}
	if len(names) != len(want) {
		t.Fatalf("GetScoped names = %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("GetScoped names = %v, want %v", names, want)
		}
	}

	gotNoTeam, err := mgr.GetScoped("", nil, &inboxID)
	if err != nil {
		t.Fatalf("GetScoped without team: %v", err)
	}
	names = nil
	for _, tag := range gotNoTeam {
		names = append(names, tag.Name)
	}
	want = []string{"global", "help-desk", "multi-scoped"}
	if len(names) != len(want) {
		t.Fatalf("GetScoped without team names = %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("GetScoped without team names = %v, want %v", names, want)
		}
	}
}

func TestMultiTeamMultiInboxScoping(t *testing.T) {
	mgr, db := newTestManagerWithDB(t)

	teamA := createTeam(t, db, "Team A")
	teamB := createTeam(t, db, "Team B")
	teamC := createTeam(t, db, "Team C")
	inboxX := createInbox(t, db, "Inbox X")
	inboxY := createInbox(t, db, "Inbox Y")

	created, err := mgr.Create("multi", []int{teamA, teamB}, []int{inboxX, inboxY})
	if err != nil {
		t.Fatalf("creating multi-scoped tag: %v", err)
	}
	if len(created.TeamIDs) != 2 || len(created.InboxIDs) != 2 {
		t.Fatalf("created tag = %+v, want 2 team ids and 2 inbox ids", created)
	}

	// Visible for either team.
	for _, teamID := range []int{teamA, teamB} {
		got, err := mgr.GetScoped("", &teamID, nil)
		if err != nil {
			t.Fatalf("GetScoped for team %d: %v", teamID, err)
		}
		if len(got) != 1 || got[0].Name != "multi" {
			t.Fatalf("GetScoped for team %d = %+v, want just the multi tag", teamID, got)
		}
	}

	// Not visible for an unrelated team with no matching inbox either.
	got, err := mgr.GetScoped("", &teamC, nil)
	if err != nil {
		t.Fatalf("GetScoped for unrelated team: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetScoped for unrelated team = %+v, want none", got)
	}

	// Shrinking to a single team via Update replaces the old set entirely.
	updated, err := mgr.Update(created.ID, "multi", []int{teamA}, nil)
	if err != nil {
		t.Fatalf("updating tag scope: %v", err)
	}
	if len(updated.TeamIDs) != 1 || updated.TeamIDs[0] != int32(teamA) || len(updated.InboxIDs) != 0 {
		t.Fatalf("updated tag = %+v, want only team A", updated)
	}
	if got, err := mgr.GetScoped("", &teamB, nil); err != nil {
		t.Fatalf("GetScoped after update: %v", err)
	} else if len(got) != 0 {
		t.Fatalf("GetScoped for team B after narrowing scope = %+v, want none", got)
	}
}
