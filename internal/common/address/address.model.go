package address

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AddressModel struct {
	bun.BaseModel `bun:"table:addresses"`

	ID               uuid.UUID  `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	UserID           *uuid.UUID `bun:"type:uuid,nullzero" json:"user_id,omitempty"`
	RestaurantID     *uuid.UUID `bun:"type:uuid,nullzero" json:"restaurant_id,omitempty"`
	FormattedAddress string     `bun:",notnull" json:"formatted_address"`
	RawAddress       string     `bun:",notnull" json:"raw_address"`
	Latitude         float64    `bun:",nullzero" json:"latitude,omitempty"`
	Longitude        float64    `bun:",nullzero" json:"longitude,omitempty"`
	IsDefault        bool       `bun:",notnull,default:false" json:"is_default"`
	CreatedAt        time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt        time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
