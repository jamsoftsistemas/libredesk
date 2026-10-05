package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	tmodels "github.com/abhinavxd/libredesk/internal/tag/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetTags returns tags from the database, all of them without page params.
func handleGetTags(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		query = string(r.RequestCtx.QueryArgs().Peek("q"))
	)
	if ids := getIDsParam(r, "ids"); len(ids) > 0 {
		t, err := app.tag.GetByIDs(ids)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(t)
	}

	// Scoped lookup for a conversation's team/inbox context, used by tag pickers. When neither
	// param is sent the full tag list is returned, as used by the admin tag management screen.
	teamIDRaw := string(r.RequestCtx.QueryArgs().Peek("team_id"))
	inboxIDRaw := string(r.RequestCtx.QueryArgs().Peek("inbox_id"))
	if teamIDRaw != "" || inboxIDRaw != "" {
		var teamID, inboxID *int
		if teamIDRaw != "" {
			v, err := strconv.Atoi(teamIDRaw)
			if err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
			}
			teamID = &v
		}
		if inboxIDRaw != "" {
			v, err := strconv.Atoi(inboxIDRaw)
			if err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
			}
			inboxID = &v
		}
		t, err := app.tag.GetScoped(query, teamID, inboxID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(t)
	}

	page, pageSize := getOptionalPagination(r)
	t, err := app.tag.GetAll(query, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

// handleCreateTag creates a new tag in the database.
func handleCreateTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		tag = tmodels.Tag{}
	)
	if err := r.Decode(&tag, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}

	if tag.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}

	createdTag, cErr := app.tag.Create(tag.Name, toIntSlice(tag.TeamIDs), toIntSlice(tag.InboxIDs))
	if cErr != nil {
		return sendErrorEnvelope(r, cErr)
	}

	return r.SendEnvelope(createdTag)
}

// handleDeleteTag deletes a tag from the database.
func handleDeleteTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	if err = app.tag.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateTag updates an existing tag in the database.
func handleUpdateTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		tag = tmodels.Tag{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	if err := r.Decode(&tag, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}

	if tag.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}

	updatedTag, err := app.tag.Update(id, tag.Name, toIntSlice(tag.TeamIDs), toIntSlice(tag.InboxIDs))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(updatedTag)
}

// toIntSlice converts a pq.Int32Array (as decoded from a JSON request body) to a plain []int.
func toIntSlice(ids []int32) []int {
	out := make([]int, len(ids))
	for i, id := range ids {
		out[i] = int(id)
	}
	return out
}
