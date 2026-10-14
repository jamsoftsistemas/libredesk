// Package company handles the management of client companies (customer portfolio).
package company

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/abhinavxd/libredesk/internal/company/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles company-related operations.
type Manager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetCompanies             *sqlx.Stmt `query:"get-companies"`
	GetCompaniesCompact      *sqlx.Stmt `query:"get-companies-compact"`
	GetCompaniesCompactByIDs *sqlx.Stmt `query:"get-companies-compact-by-ids"`
	GetCompany               *sqlx.Stmt `query:"get-company"`
	GetCompanyByExternalID   *sqlx.Stmt `query:"get-company-by-external-id"`
	InsertCompany            *sqlx.Stmt `query:"insert-company"`
	UpdateCompany            *sqlx.Stmt `query:"update-company"`
	DeleteCompany            *sqlx.Stmt `query:"delete-company"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// GetAll retrieves companies matching query, all of them when pageSize is 0.
func (m *Manager) GetAll(query string, page, pageSize int) ([]models.Company, error) {
	var companies = make([]models.Company, 0)
	if err := m.q.GetCompanies.Select(&companies, query, pageSize, dbutil.PageOffset(page, pageSize)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return companies, nil
		}
		m.lo.Error("error fetching companies", "error", err)
		return companies, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return companies, nil
}

// GetAllCompact retrieves companies with limited fields matching query, all of them when pageSize is 0.
func (m *Manager) GetAllCompact(query string, page, pageSize int) ([]models.CompanyCompact, error) {
	var companies = make([]models.CompanyCompact, 0)
	if err := m.q.GetCompaniesCompact.Select(&companies, query, pageSize, dbutil.PageOffset(page, pageSize)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return companies, nil
		}
		m.lo.Error("error fetching companies", "error", err)
		return companies, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return companies, nil
}

// GetAllCompactByIDs retrieves the companies with the given IDs with limited fields.
func (m *Manager) GetAllCompactByIDs(ids []int) ([]models.CompanyCompact, error) {
	var companies = make([]models.CompanyCompact, 0)
	if err := m.q.GetCompaniesCompactByIDs.Select(&companies, pq.Array(ids)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return companies, nil
		}
		m.lo.Error("error fetching companies by ids", "error", err)
		return companies, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return companies, nil
}

// Get retrieves a company by ID.
func (m *Manager) Get(id int) (models.Company, error) {
	var company models.Company
	if err := m.q.GetCompany.Get(&company, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			m.lo.Error("company not found", "id", id, "error", err)
			return company, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCompany"), nil)
		}
		m.lo.Error("error fetching company", "id", id, "error", err)
		return company, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return company, nil
}

// GetByExternalID retrieves a company by its external ID.
func (m *Manager) GetByExternalID(externalID string) (models.Company, error) {
	var company models.Company
	if err := m.q.GetCompanyByExternalID.Get(&company, externalID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return company, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundCompany"), nil)
		}
		m.lo.Error("error fetching company by external id", "external_id", externalID, "error", err)
		return company, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return company, nil
}

// Create creates a new company.
func (m *Manager) Create(externalID null.String, name string, document, city, system null.String) (models.Company, error) {
	var company models.Company
	if err := m.q.InsertCompany.Get(&company, externalID, name, document, city, system); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return company, envelope.NewError(envelope.GeneralError, m.i18n.T("errors.alreadyExistsCompany"), nil)
		}
		m.lo.Error("error inserting company", "error", err)
		return company, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return company, nil
}

// Update updates an existing company.
func (m *Manager) Update(id int, externalID null.String, name string, document, city, system null.String) (models.Company, error) {
	var company models.Company
	if err := m.q.UpdateCompany.Get(&company, id, externalID, name, document, city, system); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return company, envelope.NewError(envelope.GeneralError, m.i18n.T("errors.alreadyExistsCompany"), nil)
		}
		m.lo.Error("error updating company", "error", err)
		return company, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return company, nil
}

// Delete deletes a company.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteCompany.Exec(id); err != nil {
		m.lo.Error("error deleting company", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}
