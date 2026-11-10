package main

import (
	"slices"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/zerodha/fastglue"
)

// handleOverviewCounts retrieves general dashboard counts for all users.
func handleOverviewCounts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agentID, err := reportScope(r, app)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	counts, err := app.report.GetOverViewCounts(agentID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(counts)
}

// handleOverviewCharts retrieves general dashboard chart data.
func handleOverviewCharts(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	agentID, err := reportScope(r, app)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	charts, err := app.report.GetOverviewChart(days, agentID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(charts)
}

// handleOverviewSLA retrieves SLA data for the dashboard.
func handleOverviewSLA(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	agentID, err := reportScope(r, app)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	sla, err := app.report.GetOverviewSLA(days, agentID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(sla)
}

// handleOverviewCSAT retrieves CSAT metrics for the dashboard.
func handleOverviewCSAT(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	agentID, err := reportScope(r, app)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	csat, err := app.report.GetOverviewCSAT(days, agentID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(csat)
}

// handleOverviewMessageVolume retrieves message volume metrics for the dashboard.
func handleOverviewMessageVolume(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	agentID, err := reportScope(r, app)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	volume, err := app.report.GetOverviewMessageVolume(days, agentID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(volume)
}

// handleOverviewTagDistribution retrieves tag distribution metrics for the dashboard.
func handleOverviewTagDistribution(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	agentID, err := reportScope(r, app)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	tags, err := app.report.GetOverviewTagDistribution(days, agentID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tags)
}

// reportScope resolves whose data a report request may see. Users with the
// reports:manage permission get the global view (nil); every other agent is
// restricted to conversations assigned to them.
func reportScope(r *fastglue.Request, app *App) (*int, error) {
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return nil, err
	}
	if slices.Contains(user.Permissions, authzModels.PermReportsManage) {
		return nil, nil
	}
	id := user.ID
	return &id, nil
}
