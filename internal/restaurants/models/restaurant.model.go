package models

import (
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/types"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Restaurant struct {
	bun.BaseModel `bun:"table:restaurants"`

	ID                uuid.UUID            `bun:"type:uuid,pk" json:"id"`
	Name              string               `bun:",notnull" json:"name"`
	Description       string               `bun:",nullzero" json:"description,omitempty"`
	Address           string               `bun:"type:text,notnull" json:"address"`
	AvatarURL         string               `bun:",nullzero" json:"avatar_url,omitempty"`
	Status            types.RestaurantStatus `bun:",notnull,default:'active'" json:"status"`

	UserID            *uuid.UUID       `bun:"type:uuid,nullzero" json:"user_id,omitempty"`
	Capacity          int              `bun:",nullzero" json:"capacity,omitempty"`
	DeliveryAvailable bool             `bun:",notnull,default:false" json:"delivery_available"`
	TakeawayAvailable bool             `bun:",notnull,default:false" json:"takeaway_available"`
	Rating            float64          `bun:",nullzero" json:"rating"`
	Latitude          float64          `bun:",nullzero" json:"latitude"`
	Longitude         float64          `bun:",nullzero" json:"longitude"`
	AccountNumber     string           `bun:",nullzero" json:"account_number,omitempty"`
	BankName          string           `bun:",nullzero" json:"bank_name,omitempty"`
	CreatedAt         time.Time        `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt         time.Time        `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
