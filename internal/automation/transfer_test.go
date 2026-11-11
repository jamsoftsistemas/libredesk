package automation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/volatiletech/null/v9"
)

func transferRule(companyIDs string, agents ...string) models.Rule {
	return models.Rule{
		Type:          models.RuleTypeTransfer,
		GroupOperator: models.OperatorOR,
		Groups: []models.RuleGroup{{
			LogicalOp: models.OperatorOR,
			Rules: []models.RuleDetail{{
				Field: models.ContactCompany, FieldType: models.FieldTypeConversationField,
				Operator: models.RuleOperatorIn, Value: companyIDs,
			}},
		}},
		Actions: []models.RuleAction{{Type: models.ActionAssignUser, Value: agents}},
	}
}

func transferEngine(store *mockConversationStore) *Engine {
	e := createTestEngine(store)
	e.transferLast = map[int]int{}
	return e
}

func companyConv(id int) cmodels.Conversation {
	return createTestConversation(func(c *cmodels.Conversation) {
		c.Contact.CompanyID = null.IntFrom(id)
	})
}

func TestTransferRoundRobinSkipsOffline(t *testing.T) {
	store := new(mockConversationStore)
	store.On("GetAgentAvailability", 1).Return("online", nil)
	store.On("GetAgentAvailability", 2).Return("offline", nil)
	store.On("GetAgentAvailability", 3).Return("online", nil)
	store.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	e := transferEngine(store)

	// Two rules share company 7: pool is 1,2,3. Agent 2 is offline.
	rules := []models.Rule{transferRule("7", "1", "2"), transferRule("7,9", "3")}
	for range 4 {
		e.handleTransfer(rules, companyConv(7))
	}

	var got []string
	for _, a := range store.appliedActions {
		got = append(got, a.Value[0])
	}
	assert.Equal(t, []string{"1", "3", "1", "3"}, got)
}

func TestTransferNobodyAvailableLeavesUnassigned(t *testing.T) {
	store := new(mockConversationStore)
	store.On("GetAgentAvailability", mock.Anything).Return("away_manual", nil)
	e := transferEngine(store)

	e.handleTransfer([]models.Rule{transferRule("7", "1", "2")}, companyConv(7))
	assert.Equal(t, 0, store.callCount)
}

func TestTransferIgnoresOtherCompanies(t *testing.T) {
	store := new(mockConversationStore)
	e := transferEngine(store)

	e.handleTransfer([]models.Rule{transferRule("8", "1")}, companyConv(7))
	assert.Equal(t, 0, store.callCount)
}
