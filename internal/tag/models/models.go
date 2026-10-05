package models

import (
	"time"

	"github.com/lib/pq"
)

// Tag is a conversation tag. A tag with no TeamIDs/InboxIDs is global (visible to every
// conversation). Otherwise it's visible to a conversation whose assigned team or inbox matches
// any of TeamIDs/InboxIDs (OR semantics across both).
type Tag struct {
	ID        int           `db:"id" json:"id"`
	CreatedAt time.Time     `db:"created_at" json:"created_at"`
	UpdateAt  time.Time     `db:"updated_at" json:"updated_at"`
	Name      string        `db:"name" json:"name"`
	TeamIDs   pq.Int32Array `db:"team_ids" json:"team_ids"`
	InboxIDs  pq.Int32Array `db:"inbox_ids" json:"inbox_ids"`
}
