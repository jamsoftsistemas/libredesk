package automation

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

// agentAvailable reports whether an agent can take a transfer. Offline and manually away agents are skipped.
func agentAvailable(status string) bool {
	switch status {
	case umodels.Online, umodels.Away:
		return true
	}
	return false
}

// transferPool returns the ordered, de-duplicated agent IDs of every transfer rule matching the conversation.
// Agents of all matching rules (same company) share one pool.
func (e *Engine) transferPool(rules []models.Rule, conversation cmodels.Conversation) []int {
	var (
		pool []int
		seen = map[int]struct{}{}
	)
	for _, rule := range rules {
		if !e.ruleMatches(rule, conversation) {
			continue
		}
		for _, action := range rule.Actions {
			if action.Type != models.ActionAssignUser {
				continue
			}
			for _, v := range action.Value {
				id, err := strconv.Atoi(v)
				if err != nil || id <= 0 {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				pool = append(pool, id)
			}
		}
	}
	return pool
}

// pickTransferAgent round-robins over the pool, skipping unavailable agents, starting after the last agent
// chosen for this company. Returns 0 when nobody is available.
func (e *Engine) pickTransferAgent(companyID int, pool []int) int {
	e.transferMu.Lock()
	defer e.transferMu.Unlock()

	start := 0
	if last, ok := e.transferLast[companyID]; ok {
		for i, id := range pool {
			if id == last {
				start = i + 1
				break
			}
		}
	}
	for i := range pool {
		id := pool[(start+i)%len(pool)]
		status, err := e.conversationStore.GetAgentAvailability(id)
		if err != nil {
			e.lo.Error("error fetching agent availability for transfer", "agent_id", id, "error", err)
			continue
		}
		if !agentAvailable(status) {
			e.lo.Debug("agent unavailable, skipping for transfer", "agent_id", id, "availability_status", status)
			continue
		}
		e.transferLast[companyID] = id
		return id
	}
	return 0
}

// handleTransfer assigns a new conversation to an available agent of the contact's company portfolio, round-robin.
// If every agent is unavailable the conversation is left unassigned.
func (e *Engine) handleTransfer(rules []models.Rule, conversation cmodels.Conversation) {
	if !conversation.Contact.CompanyID.Valid {
		return
	}
	pool := e.transferPool(rules, conversation)
	if len(pool) == 0 {
		return
	}
	agentID := e.pickTransferAgent(conversation.Contact.CompanyID.Int, pool)
	if agentID == 0 {
		e.lo.Info("no available agent for transfer, leaving conversation unassigned", "conversation_uuid", conversation.UUID, "company_id", conversation.Contact.CompanyID.Int)
		return
	}
	e.suppress(conversation.UUID)
	defer e.unsuppress(conversation.UUID)
	action := models.RuleAction{Type: models.ActionAssignUser, Value: []string{strconv.Itoa(agentID)}}
	if err := e.conversationStore.ApplyAction(action, conversation, umodels.User{}); err != nil {
		e.lo.Error("error applying transfer assignment", "conversation_uuid", conversation.UUID, "agent_id", agentID, "error", err)
	}
}
