package models

import "time"

// Values of the tag_visibility DB enum.
const (
	VisibilityAll   = "all"
	VisibilityTeam  = "team"
	VisibilityInbox = "inbox"
)

type Tag struct {
	ID         int       `db:"id" json:"id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdateAt   time.Time `db:"updated_at" json:"updated_at"`
	Name       string    `db:"name" json:"name"`
	Visibility string    `db:"visibility" json:"visibility"`
	TeamID     *int      `db:"team_id" json:"team_id,string"`
	InboxID    *int      `db:"inbox_id" json:"inbox_id,string"`
}
