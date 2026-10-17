package aiagent

import (
	"context"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/aiagent/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
)

func TestNewPriorityToolNoPriorities(t *testing.T) {
	if tool := newPriorityTool(nil, cmodels.Conversation{}, models.Assistant{}, nil); tool != nil {
		t.Fatal("expected nil tool when there are no priorities to choose from")
	}
}

func TestPriorityToolRejectsUnknownPriority(t *testing.T) {
	tool := newPriorityTool(nil, cmodels.Conversation{}, models.Assistant{}, []string{"Low", "High"})
	if tool == nil {
		t.Fatal("expected a tool")
	}
	// An invalid priority must be rejected before the manager is ever touched (m is nil here, so a
	// nil-pointer panic would mean the tool skipped validation).
	msg, err := tool.Execute(context.Background(), `{"priority":"Urgent"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "not a valid priority") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestAddNoteToolRejectsEmptyNote(t *testing.T) {
	tool := &addNoteTool{}
	msg, err := tool.Execute(context.Background(), `{"note":"   "}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "No note text") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestNewAssignTeamToolNoTeams(t *testing.T) {
	if tool := newAssignTeamTool(&runOutcome{}, nil); tool != nil {
		t.Fatal("expected nil tool when there are no teams")
	}
}

func TestAssignTeamToolListsTeamsAndRejectsUnknownID(t *testing.T) {
	teams := []tmodels.Team{
		{ID: 1, Name: "Billing", Description: "Billing and refund questions"},
		{ID: 2, Name: "Support"},
	}
	outcome := &runOutcome{}
	tool := newAssignTeamTool(outcome, teams)
	if tool == nil {
		t.Fatal("expected a tool")
	}
	if !strings.Contains(tool.Description(), "Billing and refund questions") {
		t.Fatalf("description does not mention team description: %q", tool.Description())
	}
	msg, err := tool.Execute(context.Background(), `{"team_id":99,"reason":"test"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "not valid") {
		t.Fatalf("unexpected message: %q", msg)
	}
	if outcome.reassignTeamID != 0 {
		t.Fatal("outcome must not be set for an unknown team_id")
	}
	if _, err := tool.Execute(context.Background(), `{"team_id":1,"reason":"billing issue"}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.reassignTeamID != 1 || outcome.reassignReason != "billing issue" {
		t.Fatalf("outcome not recorded: %+v", outcome)
	}
}

func TestNewAssignAIAgentToolExcludesSelfAndDisabled(t *testing.T) {
	assistants := []models.Assistant{
		{ID: 1, UserID: 101, Name: "Self", Enabled: true},
		{ID: 2, UserID: 102, Name: "Disabled", Enabled: false},
	}
	if tool := newAssignAIAgentTool(&runOutcome{}, assistants, 1); tool != nil {
		t.Fatal("expected nil tool when every other assistant is self or disabled")
	}
}

func TestAssignAIAgentToolRecordsTargetUserID(t *testing.T) {
	assistants := []models.Assistant{
		{ID: 1, UserID: 101, Name: "Self", Enabled: true},
		{ID: 2, UserID: 102, Name: "Billing bot", Description: "Handles billing", Enabled: true},
	}
	outcome := &runOutcome{}
	tool := newAssignAIAgentTool(outcome, assistants, 1)
	if tool == nil {
		t.Fatal("expected a tool")
	}
	if _, err := tool.Execute(context.Background(), `{"agent_id":2,"reason":"billing"}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must record the target's user id (102), not its assistant id (2): that's what
	// UpdateConversationUserAssignee takes.
	if outcome.reassignAgentID != 102 {
		t.Fatalf("expected reassignAgentID 102, got %d", outcome.reassignAgentID)
	}
}
