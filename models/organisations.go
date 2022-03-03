package models

import (
	"time"
)

// Org is our Organisation object
type Org struct {
	ID       int64     `db:"id"`
	UID      string    `db:"uid" json:"uid"`
	Name     string    `db:"name" json:"name"`
	IsActive bool      `db:"is_active" json:"is_active"`
	Created  time.Time `db:"created" json:"created"`
	Updated  time.Time `db:"updated" json:"updated"`
}
