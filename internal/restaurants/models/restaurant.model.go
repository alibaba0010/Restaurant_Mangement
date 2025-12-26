package models

import (
	"time"
	"github.com/uptrace/bun"
)

type Restaurant struct {
	bun.BaseModel `bun:"table:restaurants"`

	ID          string    `bun:",pk" json:"id"`
	Name        string    `bun:",notnull" json:"name"`
	Description string    `bun:",nullzero" json:"description,omitempty"`
	Address     string    `bun:",notnull" json:"address"`
	CuisineType string    `bun:",nullzero" json:"cuisine_type,omitempty"`
	Rating      float64   `bun:",nullzero" json:"rating"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
