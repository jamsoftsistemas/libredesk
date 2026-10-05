// Package tag handles the management of tags.
package tag

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/tag/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllTags    *sqlx.Stmt `query:"get-all-tags"`
	GetTagsByIDs  *sqlx.Stmt `query:"get-tags-by-ids"`
	GetScopedTags *sqlx.Stmt `query:"get-scoped-tags"`
	InsertTag     *sqlx.Stmt `query:"insert-tag"`
	DeleteTag     *sqlx.Stmt `query:"delete-tag"`
	UpdateTag     *sqlx.Stmt `query:"update-tag"`
	SetTagTeams   *sqlx.Stmt `query:"set-tag-teams"`
	SetTagInboxes *sqlx.Stmt `query:"set-tag-inboxes"`
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

// GetAll retrieves tags matching query, all of them when pageSize is 0.
func (t *Manager) GetAll(query string, page, pageSize int) ([]models.Tag, error) {
	var tags = make([]models.Tag, 0)
	if err := t.q.GetAllTags.Select(&tags, query, pageSize, dbutil.PageOffset(page, pageSize)); err != nil {
		t.lo.Error("error fetching tags", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return tags, nil
}

// GetByIDs retrieves the tags with the given IDs.
func (t *Manager) GetByIDs(ids []int) ([]models.Tag, error) {
	var tags = make([]models.Tag, 0)
	if err := t.q.GetTagsByIDs.Select(&tags, pq.Array(ids)); err != nil {
		t.lo.Error("error fetching tags by ids", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return tags, nil
}

// GetScoped retrieves tags visible in the given team/inbox context i.e. tags with no team/inbox
// scoping (global), plus tags scoped to the given team or inbox. Pass nil for teamID when the
// context has no team (e.g. an unassigned conversation).
func (t *Manager) GetScoped(query string, teamID, inboxID *int) ([]models.Tag, error) {
	var tags = make([]models.Tag, 0)
	if err := t.q.GetScopedTags.Select(&tags, query, teamID, inboxID); err != nil {
		t.lo.Error("error fetching scoped tags", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return tags, nil
}

// Create creates a new tag scoped to the given teams/inboxes. Empty teamIDs and inboxIDs makes
// the tag global.
func (t *Manager) Create(name string, teamIDs, inboxIDs []int) (models.Tag, error) {
	var tag models.Tag
	if err := t.q.InsertTag.Get(&tag, name); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return tag, envelope.NewError(envelope.ConflictError, t.i18n.T("errors.alreadyExistsTag"), nil)
		}
		t.lo.Error("error inserting tag", "error", err)
		return tag, envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := t.setScope(tag.ID, teamIDs, inboxIDs); err != nil {
		return tag, err
	}
	tag.TeamIDs, tag.InboxIDs = toInt32Array(teamIDs), toInt32Array(inboxIDs)
	return tag, nil
}

// Delete deletes a tag by ID.
func (t *Manager) Delete(id int) error {
	if _, err := t.q.DeleteTag.Exec(id); err != nil {
		t.lo.Error("error deleting tag", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// Update updates a tag's name and team/inbox scoping by id.
func (t *Manager) Update(id int, name string, teamIDs, inboxIDs []int) (models.Tag, error) {
	var tag models.Tag
	if err := t.q.UpdateTag.Get(&tag, id, name); err != nil {
		t.lo.Error("error updating tag", "error", err)
		return tag, envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := t.setScope(id, teamIDs, inboxIDs); err != nil {
		return tag, err
	}
	tag.TeamIDs, tag.InboxIDs = toInt32Array(teamIDs), toInt32Array(inboxIDs)
	return tag, nil
}

// setScope replaces a tag's team and inbox associations.
func (t *Manager) setScope(tagID int, teamIDs, inboxIDs []int) error {
	if _, err := t.q.SetTagTeams.Exec(tagID, pq.Array(teamIDs)); err != nil {
		t.lo.Error("error setting tag teams", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := t.q.SetTagInboxes.Exec(tagID, pq.Array(inboxIDs)); err != nil {
		t.lo.Error("error setting tag inboxes", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func toInt32Array(ids []int) pq.Int32Array {
	out := make(pq.Int32Array, len(ids))
	for i, id := range ids {
		out[i] = int32(id)
	}
	return out
}
