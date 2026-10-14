package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// Company represents a client company in the customer portfolio.
type Company struct {
	ID         int         `db:"id" json:"id"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time   `db:"updated_at" json:"updated_at"`
	ExternalID null.String `db:"external_id" json:"external_id"`
	Name       string      `db:"name" json:"name"`
	Document   null.String `db:"document" json:"document"`
	City       null.String `db:"city" json:"city"`
	System     null.String `db:"system" json:"system"`
	Total      int         `db:"total" json:"-"`
}

// CompanyCompact is a minimal representation of a company, for use in search and selection pickers.
type CompanyCompact struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}
