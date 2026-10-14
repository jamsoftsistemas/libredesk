package main

import (
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/company/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetCompanies returns a list of companies from the database.
func handleGetCompanies(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		query = string(r.RequestCtx.QueryArgs().Peek("q"))
		total = 0
	)
	page, pageSize := getPagination(r)
	companies, err := app.company.GetAll(query, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(companies) > 0 {
		total = companies[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    companies,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetCompaniesCompact returns companies in a compact format, for use in search and pickers.
func handleGetCompaniesCompact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		query = string(r.RequestCtx.QueryArgs().Peek("q"))
	)
	if ids := getIDsParam(r, "ids"); len(ids) > 0 {
		companies, err := app.company.GetAllCompactByIDs(ids)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(companies)
	}
	page, pageSize := getOptionalPagination(r)
	companies, err := app.company.GetAllCompact(query, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(companies)
}

// handleGetCompany returns a single company.
func handleGetCompany(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	company, err := app.company.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(company)
}

// handleCreateCompany creates a new company.
func handleCreateCompany(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.Company{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "name"), nil, envelope.InputError)
	}
	company, err := app.company.Create(req.ExternalID, req.Name, req.Document, req.City, req.System)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(company)
}

// handleUpdateCompany updates an existing company.
func handleUpdateCompany(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		req   = models.Company{}
	)
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "name"), nil, envelope.InputError)
	}
	company, err := app.company.Update(id, req.ExternalID, req.Name, req.Document, req.City, req.System)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(company)
}

// handleDeleteCompany deletes a company.
func handleDeleteCompany(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.company.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
